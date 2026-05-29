package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/bocklucas/dvb-made-easy/internal/config"
	"github.com/bocklucas/dvb-made-easy/internal/gitimport"
	"github.com/bocklucas/dvb-made-easy/internal/portainer"
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

	gitConfig := req.GitConfig
	if gitConfig != nil {
		gitCopy := *gitConfig
		gitCopy.FilePath = ""
		gitConfig = &gitCopy
	}

	source := config.SavedSource{
		Name:            req.Name,
		Type:            req.Type,
		GitConfig:       gitConfig,
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

func (s *Server) handleGetSavedSource(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	source, ok := s.manifest.GetSavedSource(id)
	if !ok {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sanitizeSavedSource(source))
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

func (s *Server) handleUpdateSavedSource(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

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

	existing, ok := s.manifest.GetSavedSource(id)
	if !ok {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	gitConfig := req.GitConfig
	if gitConfig != nil {
		gitCopy := *gitConfig
		gitCopy.FilePath = ""
		if existing.GitConfig != nil {
			mergeGitConfigs(&gitCopy, existing.GitConfig)
		}
		gitConfig = &gitCopy
	}

	portainerConfig := req.PortainerConfig
	if portainerConfig != nil {
		if existing.PortainerConfig != nil {
			mergePortainerConfigs(portainerConfig, existing.PortainerConfig)
		}
	}

	source := config.SavedSource{
		Name:            req.Name,
		Type:            req.Type,
		GitConfig:       gitConfig,
		PortainerConfig: portainerConfig,
	}

	if !s.manifest.UpdateSavedSource(id, source) {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	s.persistConfig()

	updated, _ := s.manifest.GetSavedSource(id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sanitizeSavedSource(updated))
}

func (s *Server) handleTestSavedSource(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	saved, ok := s.manifest.GetSavedSource(id)
	if !ok {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	var testErr error
	switch saved.Type {
	case "git":
		if saved.GitConfig == nil {
			jsonError(w, "no git configuration", http.StatusBadRequest)
			return
		}
		testErr = gitimport.TestConnection(*saved.GitConfig)
	case "portainer":
		if saved.PortainerConfig == nil {
			jsonError(w, "no portainer configuration", http.StatusBadRequest)
			return
		}
		client := portainer.NewClient(saved.PortainerConfig.PortainerURL, saved.PortainerConfig.APIKey)
		testErr = client.TestConnection(r.Context())
	default:
		jsonError(w, "unknown source type", http.StatusBadRequest)
		return
	}

	if testErr != nil {
		jsonError(w, testErr.Error(), http.StatusUnprocessableEntity)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func sanitizeSavedSource(src config.SavedSource) config.SavedSource {
	if src.GitConfig != nil {
		src.HasCredentials = src.GitConfig.AuthToken != "" || src.GitConfig.SSHPrivateKey != ""
		safe := *src.GitConfig
		safe.HasAuthToken = safe.AuthToken != ""
		safe.HasSSHKey = safe.SSHPrivateKey != ""
		safe.AuthToken = ""
		safe.SSHPrivateKey = ""
		src.GitConfig = &safe
	}
	if src.PortainerConfig != nil {
		src.HasCredentials = src.PortainerConfig.APIKey != ""
		safe := *src.PortainerConfig
		safe.HasAPIKey = safe.APIKey != ""
		safe.APIKey = ""
		src.PortainerConfig = &safe
	}
	return src
}
