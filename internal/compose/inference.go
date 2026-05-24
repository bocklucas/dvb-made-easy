package compose

import "strings"

// InferenceResult holds auto-detected configuration derived from a compose file.
type InferenceResult struct {
	Volumes []InferredVolume   `json:"volumes"`
	Storage *StorageSuggestion `json:"storage,omitempty"`
}

// InferredVolume represents a volume with all auto-detected metadata.
type InferredVolume struct {
	Name             string `json:"name"`
	ComposeService   string `json:"compose_service"`
	ComposeMountPath string `json:"compose_mount_path"`
	BackupPattern    string `json:"backup_pattern,omitempty"`
	TargetVolumeName string `json:"target_volume_name"`
	IsEncrypted      bool   `json:"is_encrypted"`
}

// StorageSuggestion holds a detected storage backend configuration.
type StorageSuggestion struct {
	Type       string `json:"type"`                  // "local" or "smb"
	Confidence string `json:"confidence"`            // "high", "medium", "low"
	Source     string `json:"source"`                // e.g. "BACKUP_ARCHIVE env var"
	LocalPath  string `json:"local_path,omitempty"`
	SMBHost    string `json:"smb_host,omitempty"`
	SMBShare   string `json:"smb_share,omitempty"`
	SMBUser    string `json:"smb_username,omitempty"`
}

// Infer parses a compose file and returns auto-detected configuration for volumes
// and storage backend.
func Infer(content string) (*InferenceResult, error) {
	parsed, err := Parse(content)
	if err != nil {
		return nil, err
	}

	// Build a lookup: volume name → backup job
	backupByVolume := make(map[string]BackupJob, len(parsed.BackupJobs))
	for _, job := range parsed.BackupJobs {
		backupByVolume[job.SourceVolume] = job
	}

	result := &InferenceResult{}

	for _, vm := range parsed.Volumes {
		iv := InferredVolume{
			Name:             vm.Name,
			ComposeService:   vm.Service,
			ComposeMountPath: vm.MountPath,
			TargetVolumeName: vm.Name,
		}
		if job, ok := backupByVolume[vm.Name]; ok {
			iv.BackupPattern = job.FilenameFormat
			iv.IsEncrypted = strings.HasSuffix(job.FilenameFormat, ".gpg")
		}
		result.Volumes = append(result.Volumes, iv)
	}

	result.Storage = inferStorage(parsed)

	return result, nil
}

// inferStorage scans the backup service environment and volume mounts for
// storage backend clues.
func inferStorage(parsed *ParseResult) *StorageSuggestion {
	env := parsed.BackupServiceEnv
	vols := parsed.BackupServiceVolumes

	if len(env) == 0 && len(vols) == 0 {
		return nil
	}

	// SMB: explicit env vars take priority
	smbHost := env["SMB_HOST"]
	smbShare := env["SMB_SHARE"]
	smbUser := env["SMB_USERNAME"]
	if smbHost != "" {
		confidence := "high"
		source := "SMB_HOST env var"
		if smbShare == "" {
			confidence = "medium"
		}
		return &StorageSuggestion{
			Type:       "smb",
			Confidence: confidence,
			Source:     source,
			SMBHost:    smbHost,
			SMBShare:   smbShare,
			SMBUser:    smbUser,
		}
	}

	// SMB: NOTIFICATION_URLS containing smb://
	if notif := env["NOTIFICATION_URLS"]; strings.Contains(notif, "smb://") {
		return &StorageSuggestion{
			Type:       "smb",
			Confidence: "medium",
			Source:     "NOTIFICATION_URLS env var",
		}
	}

	// Local: BACKUP_ARCHIVE env var
	if archivePath := env["BACKUP_ARCHIVE"]; archivePath != "" {
		return &StorageSuggestion{
			Type:       "local",
			Confidence: "high",
			Source:     "BACKUP_ARCHIVE env var",
			LocalPath:  archivePath,
		}
	}

	// Local: volume mount to /backup or /archive paths
	for _, volMount := range vols {
		parts := strings.SplitN(volMount, ":", 3)
		if len(parts) < 2 {
			continue
		}
		mountTarget := parts[1]
		// Strip any options (e.g., :ro)
		if colonIdx := strings.Index(mountTarget, ":"); colonIdx >= 0 {
			mountTarget = mountTarget[:colonIdx]
		}
		if strings.HasPrefix(mountTarget, "/backup") || strings.HasPrefix(mountTarget, "/archive") {
			// Only suggest local if the source looks like a bind-mount path (starts with / or .)
			source := parts[0]
			if strings.HasPrefix(source, "/") || strings.HasPrefix(source, ".") {
				return &StorageSuggestion{
					Type:       "local",
					Confidence: "medium",
					Source:     "volume mount to " + mountTarget,
					LocalPath:  source,
				}
			}
		}
	}

	return nil
}
