package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/offen/restore-manager/internal/storage"
)

type credentialResponse struct {
	Type  storage.BackendType `json:"type"`
	Local *localCredResponse  `json:"local,omitempty"`
	SMB   *smbCredResponse    `json:"smb,omitempty"`
}

type localCredResponse struct {
	Path string `json:"path"`
}

type smbCredResponse struct {
	Host     string `json:"host"`
	Share    string `json:"share"`
	Path     string `json:"path"`
	Username string `json:"username"`
	Port     int    `json:"port"`
}

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

	resp := credentialResponse{
		Type: project.Credentials.Type,
	}

	if project.Credentials.Local != nil {
		resp.Local = &localCredResponse{
			Path: project.Credentials.Local.Path,
		}
	}

	if project.Credentials.SMB != nil {
		resp.SMB = &smbCredResponse{
			Host:     project.Credentials.SMB.Host,
			Share:    project.Credentials.SMB.Share,
			Path:     project.Credentials.SMB.Path,
			Username: project.Credentials.SMB.Username,
			Port:     project.Credentials.SMB.Port,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
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

	// If SMB password is empty, preserve the existing stored password.
	if creds.Type == storage.BackendSMB && creds.SMB != nil && creds.SMB.Password == "" {
		if project.Credentials != nil && project.Credentials.SMB != nil {
			creds.SMB.Password = project.Credentials.SMB.Password
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

func (s *Server) handleDeleteCredentials(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if !s.manifest.ClearProjectCredentials(id) {
		http.Error(w, `{"error":"project not found"}`, http.StatusNotFound)
		return
	}

	s.persistConfig()
	w.WriteHeader(http.StatusNoContent)
}
