package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/offen/restore-manager/internal/storage"
)

func (s *Server) handleSaveCredentials(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	_, ok := s.manifest.GetProject(id)
	if !ok {
		http.Error(w, `{"error":"project not found"}`, http.StatusNotFound)
		return
	}

	var creds storage.Credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if creds.SavedBackendID != "" {
		saved, ok := s.manifest.GetSavedBackend(creds.SavedBackendID)
		if ok {
			mergeCredentials(&creds, &saved.Credentials)
		}
	}

	backend, err := storage.NewBackend(&creds)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := backend.TestConnection(r.Context()); err != nil {
		jsonError(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	s.manifest.SetProjectCredentials(id, &creds)
	s.persistConfig()

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func (s *Server) handleGetCredentials(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	project, ok := s.manifest.GetProject(id)
	if !ok {
		http.Error(w, `{"error":"project not found"}`, http.StatusNotFound)
		return
	}

	if project.Credentials == nil {
		http.Error(w, `{"error":"no credentials configured"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(project.Credentials.Sanitize())
}

func (s *Server) handleUpdateCredentials(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	project, ok := s.manifest.GetProject(id)
	if !ok {
		http.Error(w, `{"error":"project not found"}`, http.StatusNotFound)
		return
	}

	var creds storage.Credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	// Resolve saved backend if provided
	if creds.SavedBackendID != "" {
		saved, ok := s.manifest.GetSavedBackend(creds.SavedBackendID)
		if ok {
			mergeCredentials(&creds, &saved.Credentials)
		}
	}

	// Preserve existing stored secrets if sent empty from the frontend.
	if project.Credentials != nil {
		mergeCredentials(&creds, project.Credentials)
	}

	backend, err := storage.NewBackend(&creds)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := backend.TestConnection(r.Context()); err != nil {
		jsonError(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	s.manifest.SetProjectCredentials(id, &creds)
	s.persistConfig()

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func (s *Server) handleDeleteCredentials(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if !s.manifest.ClearProjectCredentials(id) {
		http.Error(w, `{"error":"project not found"}`, http.StatusNotFound)
		return
	}

	s.persistConfig()
	w.WriteHeader(http.StatusNoContent)
}
