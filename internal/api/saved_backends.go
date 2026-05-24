package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/offen/restore-manager/internal/config"
	"github.com/offen/restore-manager/internal/storage"
)

type createSavedBackendRequest struct {
	Name        string              `json:"name"`
	Credentials storage.Credentials `json:"credentials"`
}

func (s *Server) handleCreateSavedBackend(w http.ResponseWriter, r *http.Request) {
	var req createSavedBackendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		jsonError(w, "name is required", http.StatusBadRequest)
		return
	}

	backend, err := storage.NewBackend(&req.Credentials)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := backend.TestConnection(r.Context()); err != nil {
		jsonError(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	saved := config.SavedBackend{
		Name:        req.Name,
		Credentials: req.Credentials,
	}

	created := s.manifest.AddSavedBackend(saved)
	s.persistConfig()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

func (s *Server) handleListSavedBackends(w http.ResponseWriter, r *http.Request) {
	backends := s.manifest.ListSavedBackends()
	sanitized := make([]config.SavedBackend, len(backends))
	for i, b := range backends {
		sanitized[i] = sanitizeSavedBackend(b)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sanitized)
}

func (s *Server) handleDeleteSavedBackend(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if !s.manifest.RemoveSavedBackend(id) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	s.persistConfig()
	w.WriteHeader(http.StatusNoContent)
}

func sanitizeSavedBackend(b config.SavedBackend) config.SavedBackend {
	if b.Credentials.SMB != nil {
		smbCopy := *b.Credentials.SMB
		smbCopy.Password = ""
		b.Credentials.SMB = &smbCopy
	}
	return b
}
