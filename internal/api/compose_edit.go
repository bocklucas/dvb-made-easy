package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/offen/restore-manager/internal/compose"
	"github.com/offen/restore-manager/internal/config"
)

type updateComposeRequest struct {
	ComposeContent string `json:"compose_content"`
}

type composeDiffResponse struct {
	Added     []volumeResponse `json:"added"`
	Removed   []volumeResponse `json:"removed"`
	Unchanged []volumeResponse `json:"unchanged"`
}

type confirmRemovalRequest struct {
	Volumes []string `json:"volumes"`
}

func (s *Server) handleUpdateCompose(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	project, ok := s.manifest.GetProject(id)
	if !ok {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	var req updateComposeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if req.ComposeContent == "" {
		http.Error(w, `{"error":"compose_content is required"}`, http.StatusBadRequest)
		return
	}

	result, err := compose.Parse(req.ComposeContent)
	if err != nil {
		jsonError(w, "failed to parse compose file: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Build lookup of existing volumes by name.
	existing := make(map[string]config.Volume, len(project.Volumes))
	for _, v := range project.Volumes {
		existing[v.Name] = v
	}

	// Build lookup of backup patterns from new compose BackupJobs.
	patternMap := make(map[string]string)
	for _, job := range result.BackupJobs {
		patternMap[job.SourceVolume] = job.FilenameFormat
	}

	// Deduplicate incoming volumes and track insertion order.
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
			// Unchanged — preserve existing BackupPattern and Passphrase.
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
			// New volume — pick up pattern from BackupJobs if available.
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

	// Detect removed volumes — NOT deleted yet. They stay in the manifest until
	// the user explicitly confirms removal via the confirm-removal endpoint.
	var removed []volumeResponse
	for _, prev := range project.Volumes {
		if !newSet[prev.Name] {
			removed = append(removed, volumeResponse{
				Name:             prev.Name,
				ComposeService:   prev.ComposeService,
				ComposeMountPath: prev.ComposeMountPath,
				BackupPattern:    prev.BackupPattern,
			})
			// Retain in finalVolumes to preserve stored passphrase.
			finalVolumes = append(finalVolumes, prev)
		}
	}

	// Persist updated compose content and merged volume list.
	s.manifest.UpdateProjectCompose(id, req.ComposeContent, finalVolumes)
	s.persistConfig()

	diff := composeDiffResponse{
		Added:     added,
		Removed:   removed,
		Unchanged: unchanged,
	}
	if diff.Added == nil {
		diff.Added = []volumeResponse{}
	}
	if diff.Removed == nil {
		diff.Removed = []volumeResponse{}
	}
	if diff.Unchanged == nil {
		diff.Unchanged = []volumeResponse{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(diff)
}

func (s *Server) handleConfirmVolumeRemoval(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	project, ok := s.manifest.GetProject(id)
	if !ok {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	var req confirmRemovalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if len(req.Volumes) == 0 {
		http.Error(w, `{"error":"volumes list is required"}`, http.StatusBadRequest)
		return
	}

	toRemove := make(map[string]bool, len(req.Volumes))
	for _, name := range req.Volumes {
		toRemove[name] = true
	}

	var remaining []config.Volume
	for _, v := range project.Volumes {
		if !toRemove[v.Name] {
			remaining = append(remaining, v)
		}
	}
	if remaining == nil {
		remaining = []config.Volume{}
	}

	s.manifest.UpdateProjectVolumes(id, remaining)
	s.persistConfig()

	w.WriteHeader(http.StatusNoContent)
}
