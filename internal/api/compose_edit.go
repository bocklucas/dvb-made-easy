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

	finalVolumes, diff := diffComposeVolumes(project.Volumes, result)

	// Persist updated compose content and merged volume list.
	s.manifest.UpdateProjectCompose(id, req.ComposeContent, finalVolumes)
	s.persistConfig()

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
