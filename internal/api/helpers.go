package api

import (
	"github.com/offen/restore-manager/internal/compose"
	"github.com/offen/restore-manager/internal/config"
	"github.com/offen/restore-manager/internal/gitimport"
	"github.com/offen/restore-manager/internal/portainer"
	"github.com/offen/restore-manager/internal/storage"
)

// mergeCredentials copies non-empty secret credentials from source to target
// if they are empty in target.
func mergeCredentials(target, source *storage.Credentials) {
	if target.Type != source.Type {
		return
	}
	switch target.Type {
	case storage.BackendSMB:
		if target.SMB != nil && source.SMB != nil {
			if target.SMB.Password == "" {
				target.SMB.Password = source.SMB.Password
			}
		}
	case storage.BackendS3:
		if target.S3 != nil && source.S3 != nil {
			if target.S3.SecretKey == "" {
				target.S3.SecretKey = source.S3.SecretKey
			}
		}
	case storage.BackendWebDAV:
		if target.WebDAV != nil && source.WebDAV != nil {
			if target.WebDAV.Password == "" {
				target.WebDAV.Password = source.WebDAV.Password
			}
		}
	case storage.BackendAzure:
		if target.Azure != nil && source.Azure != nil {
			if target.Azure.ConnectionString == "" {
				target.Azure.ConnectionString = source.Azure.ConnectionString
			}
		}
	case storage.BackendDropbox:
		if target.Dropbox != nil && source.Dropbox != nil {
			if target.Dropbox.AccessToken == "" {
				target.Dropbox.AccessToken = source.Dropbox.AccessToken
			}
			if target.Dropbox.AppSecret == "" {
				target.Dropbox.AppSecret = source.Dropbox.AppSecret
			}
		}
	case storage.BackendGDrive:
		if target.GDrive != nil && source.GDrive != nil {
			if target.GDrive.Credentials == "" {
				target.GDrive.Credentials = source.GDrive.Credentials
			}
		}
	case storage.BackendSFTP:
		if target.SFTP != nil && source.SFTP != nil {
			if target.SFTP.Password == "" {
				target.SFTP.Password = source.SFTP.Password
			}
			if target.SFTP.PrivateKey == "" {
				target.SFTP.PrivateKey = source.SFTP.PrivateKey
			}
		}
	}
}

// mergeGitConfigs merges secret credentials from source gitimport.GitSource to target.
func mergeGitConfigs(target, source *gitimport.GitSource) {
	if target.AuthToken == "" {
		target.AuthToken = source.AuthToken
	}
	if target.SSHPrivateKey == "" {
		target.SSHPrivateKey = source.SSHPrivateKey
	}
}

// mergePortainerConfigs merges secret credentials from source portainer.PortainerSource to target.
func mergePortainerConfigs(target, source *portainer.PortainerSource) {
	if target.APIKey == "" {
		target.APIKey = source.APIKey
	}
}

func parseComposeVolumes(result *compose.ParseResult) ([]config.Volume, []volumeResponse) {
	patternMap := make(map[string]string)
	for _, job := range result.BackupJobs {
		patternMap[job.SourceVolume] = job.FilenameFormat
	}

	seen := make(map[string]bool)
	var volumes []config.Volume
	var volResponses []volumeResponse
	for _, v := range result.Volumes {
		if seen[v.Name] {
			continue
		}
		seen[v.Name] = true
		vol := config.Volume{
			Name:             v.Name,
			ComposeService:   v.Service,
			ComposeMountPath: v.MountPath,
			BackupPattern:    patternMap[v.Name],
		}
		volumes = append(volumes, vol)
		volResponses = append(volResponses, volumeResponse{
			Name:             v.Name,
			ComposeService:   v.Service,
			ComposeMountPath: v.MountPath,
			BackupPattern:    vol.BackupPattern,
		})
	}
	return volumes, volResponses
}

func diffComposeVolumes(currentVolumes []config.Volume, result *compose.ParseResult) ([]config.Volume, composeDiffResponse) {
	existing := make(map[string]config.Volume, len(currentVolumes))
	for _, v := range currentVolumes {
		existing[v.Name] = v
	}

	patternMap := make(map[string]string)
	for _, job := range result.BackupJobs {
		patternMap[job.SourceVolume] = job.FilenameFormat
	}

	seen := make(map[string]bool)
	var newVolumeNames []string
	newVolumeMeta := make(map[string]compose.VolumeMapping)
	for _, vm := range result.Volumes {
		if seen[vm.Name] {
			continue
		}
		seen[vm.Name] = true
		newVolumeNames = append(newVolumeNames, vm.Name)
		newVolumeMeta[vm.Name] = vm
	}

	newSet := make(map[string]bool, len(newVolumeNames))
	for _, n := range newVolumeNames {
		newSet[n] = true
	}

	var added, unchanged []volumeResponse
	var finalVolumes []config.Volume

	for _, name := range newVolumeNames {
		vm := newVolumeMeta[name]
		if prev, exists := existing[name]; exists {
			unchanged = append(unchanged, volumeResponse{
				Name:             name,
				ComposeService:   vm.Service,
				ComposeMountPath: vm.MountPath,
				BackupPattern:    prev.BackupPattern,
			})
			finalVolumes = append(finalVolumes, config.Volume{
				Name:             name,
				ComposeService:   vm.Service,
				ComposeMountPath: vm.MountPath,
				BackupPattern:    prev.BackupPattern,
				Passphrase:       prev.Passphrase,
			})
		} else {
			pattern := patternMap[name]
			added = append(added, volumeResponse{
				Name:             name,
				ComposeService:   vm.Service,
				ComposeMountPath: vm.MountPath,
				BackupPattern:    pattern,
			})
			finalVolumes = append(finalVolumes, config.Volume{
				Name:             name,
				ComposeService:   vm.Service,
				ComposeMountPath: vm.MountPath,
				BackupPattern:    pattern,
			})
		}
	}

	var removed []volumeResponse
	for _, prev := range currentVolumes {
		if !newSet[prev.Name] {
			removed = append(removed, volumeResponse{
				Name:             prev.Name,
				ComposeService:   prev.ComposeService,
				ComposeMountPath: prev.ComposeMountPath,
				BackupPattern:    prev.BackupPattern,
			})
			finalVolumes = append(finalVolumes, prev)
		}
	}

	if added == nil {
		added = []volumeResponse{}
	}
	if removed == nil {
		removed = []volumeResponse{}
	}
	if unchanged == nil {
		unchanged = []volumeResponse{}
	}

	return finalVolumes, composeDiffResponse{
		Added:     added,
		Removed:   removed,
		Unchanged: unchanged,
	}
}
