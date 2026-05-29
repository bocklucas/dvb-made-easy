package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/bocklucas/dvb-made-easy/internal/config"
	"github.com/bocklucas/dvb-made-easy/internal/storage"
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

	creds := req.Credentials
	if creds.SMB != nil {
		smbCopy := *creds.SMB
		smbCopy.Path = ""
		creds.SMB = &smbCopy
	}

	saved := config.SavedBackend{
		Name:        req.Name,
		Credentials: creds,
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

func (s *Server) handleGetSavedBackend(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	backend, ok := s.manifest.GetSavedBackend(id)
	if !ok {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sanitizeSavedBackend(backend))
}

func (s *Server) handleUpdateSavedBackend(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	existing, ok := s.manifest.GetSavedBackend(id)
	if !ok {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	var req createSavedBackendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		jsonError(w, "name is required", http.StatusBadRequest)
		return
	}

	// Preserve existing stored secrets if sent empty from the frontend.
	mergeCredentials(&req.Credentials, &existing.Credentials)

	backend, err := storage.NewBackend(&req.Credentials)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := backend.TestConnection(r.Context()); err != nil {
		jsonError(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	creds := req.Credentials
	if creds.SMB != nil {
		smbCopy := *creds.SMB
		smbCopy.Path = ""
		creds.SMB = &smbCopy
	}

	saved := config.SavedBackend{
		Name:        req.Name,
		Credentials: creds,
	}

	if !s.manifest.UpdateSavedBackend(id, saved) {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	s.persistConfig()

	updated, _ := s.manifest.GetSavedBackend(id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sanitizeSavedBackend(updated))
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
	b.HasCredentials = b.Credentials.HasSecrets()
	b.Credentials = b.Credentials.Sanitize()
	return b
}

func (s *Server) handleTestSavedBackend(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	saved, ok := s.manifest.GetSavedBackend(id)
	if !ok {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	backend, err := storage.NewBackend(&saved.Credentials)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := backend.TestConnection(r.Context()); err != nil {
		jsonError(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
