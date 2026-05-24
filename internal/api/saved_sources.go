package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/offen/restore-manager/internal/config"
	"github.com/offen/restore-manager/internal/gitimport"
	"github.com/offen/restore-manager/internal/portainer"
)

type createSavedSourceRequest struct {
	Name            string                     `json:"name"`
	Type            string                     `json:"type"`
	GitConfig       *gitimport.GitSource       `json:"git_config,omitempty"`
	PortainerConfig *portainer.PortainerSource `json:"portainer_config,omitempty"`
}

func (s *Server) handleCreateSavedSource(w http.ResponseWriter, r *http.Request) {
	var req createSavedSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		jsonError(w, "name is required", http.StatusBadRequest)
		return
	}

	if req.Type != "git" && req.Type != "portainer" {
		jsonError(w, "type must be \"git\" or \"portainer\"", http.StatusBadRequest)
		return
	}

	if req.Type == "git" && req.GitConfig == nil {
		jsonError(w, "git_config is required for type \"git\"", http.StatusBadRequest)
		return
	}

	if req.Type == "portainer" && req.PortainerConfig == nil {
		jsonError(w, "portainer_config is required for type \"portainer\"", http.StatusBadRequest)
		return
	}

	source := config.SavedSource{
		Name:            req.Name,
		Type:            req.Type,
		GitConfig:       req.GitConfig,
		PortainerConfig: req.PortainerConfig,
	}

	created := s.manifest.AddSavedSource(source)
	s.persistConfig()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

func (s *Server) handleListSavedSources(w http.ResponseWriter, r *http.Request) {
	sources := s.manifest.ListSavedSources()
	sanitized := make([]config.SavedSource, len(sources))
	for i, src := range sources {
		sanitized[i] = sanitizeSavedSource(src)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sanitized)
}

func (s *Server) handleDeleteSavedSource(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if !s.manifest.RemoveSavedSource(id) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	s.persistConfig()
	w.WriteHeader(http.StatusNoContent)
}

func sanitizeSavedSource(src config.SavedSource) config.SavedSource {
	if src.GitConfig != nil {
		safe := *src.GitConfig
		safe.AuthToken = ""
		safe.SSHPrivateKey = ""
		src.GitConfig = &safe
	}
	if src.PortainerConfig != nil {
		safe := *src.PortainerConfig
		safe.APIKey = ""
		src.PortainerConfig = &safe
	}
	return src
}
