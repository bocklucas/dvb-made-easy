package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/offen/restore-manager/internal/compose"
	"github.com/offen/restore-manager/internal/storage"
)

type timestampBackup struct {
	VolumeName  string `json:"volume_name"`
	Key         string `json:"key"`
	Size        int64  `json:"size"`
	IsEncrypted bool   `json:"is_encrypted"`
}

type timestampGroup struct {
	Timestamp string            `json:"timestamp"`
	Backups   []timestampBackup `json:"backups"`
	Complete  bool              `json:"complete"`
}

func (s *Server) handleBackupTimestamps(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	project, ok := s.manifest.GetProject(id)
	if !ok {
		http.Error(w, `{"error":"project not found"}`, http.StatusNotFound)
		return
	}

	if project.Credentials == nil {
		http.Error(w, `{"error":"no storage credentials configured"}`, http.StatusBadRequest)
		return
	}

	parsed, err := compose.Parse(project.ComposeContent)
	if err != nil || len(parsed.BackupJobs) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}

	backend, err := storage.NewBackend(project.Credentials)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Build capture patterns for each volume with a BackupJob.
	// Use the project volume's BackupPattern if set, otherwise the job's FilenameFormat.
	type volPattern struct {
		volumeName string
		re         *regexp.Regexp
	}
	var volumePatterns []volPattern

	for _, job := range parsed.BackupJobs {
		pattern := job.FilenameFormat
		for _, v := range project.Volumes {
			if v.Name == job.SourceVolume && v.BackupPattern != "" {
				pattern = v.BackupPattern
				break
			}
		}
		re, err := storage.CompileBackupPatternWithCapture(pattern)
		if err != nil {
			continue
		}
		volumePatterns = append(volumePatterns, volPattern{volumeName: job.SourceVolume, re: re})
	}

	// Group backups by timestamp. Fetch all backups for each volume pattern.
	groups := make(map[string][]timestampBackup)
	seen := make(map[string]bool) // avoid double-counting the same file

	for _, vp := range volumePatterns {
		backupsForVol, err := backend.ListBackups(r.Context(), vp.re)
		if err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for _, b := range backupsForVol {
			key := vp.volumeName + ":" + b.Key
			if seen[key] {
				continue
			}
			seen[key] = true

			ts, matched := storage.ExtractTimestamp(b.Key, vp.re)
			if !matched {
				continue
			}
			groups[ts] = append(groups[ts], timestampBackup{
				VolumeName:  vp.volumeName,
				Key:         b.Key,
				Size:        b.Size,
				IsEncrypted: strings.HasSuffix(b.Key, ".gpg"),
			})
		}
	}

	// Determine completeness: a group is complete if every volume with a backup job
	// has at least one backup in that group.
	volumesWithJobs := make(map[string]bool)
	for _, job := range parsed.BackupJobs {
		volumesWithJobs[job.SourceVolume] = true
	}

	result := make([]timestampGroup, 0, len(groups))
	for ts, backups := range groups {
		present := make(map[string]bool)
		for _, b := range backups {
			present[b.VolumeName] = true
		}

		complete := true
		for vol := range volumesWithJobs {
			if !present[vol] {
				complete = false
				break
			}
		}

		result = append(result, timestampGroup{
			Timestamp: ts,
			Backups:   backups,
			Complete:  complete,
		})
	}

	// Sort by timestamp descending (newest first).
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp > result[j].Timestamp
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
