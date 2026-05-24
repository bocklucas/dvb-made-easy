package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type passphraseRequest struct {
	Passphrase string `json:"passphrase"`
}

func (s *Server) handleSetVolumePassphrase(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")
	volumeName := chi.URLParam(r, "vol")

	var req passphraseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Passphrase == "" {
		http.Error(w, `{"error":"passphrase must not be empty"}`, http.StatusBadRequest)
		return
	}

	project, ok := s.manifest.GetProject(projectID)
	if !ok {
		http.Error(w, `{"error":"project not found"}`, http.StatusNotFound)
		return
	}

	volumeFound := false
	for _, v := range project.Volumes {
		if v.Name == volumeName {
			volumeFound = true
			break
		}
	}
	if !volumeFound {
		http.Error(w, `{"error":"volume not found"}`, http.StatusNotFound)
		return
	}

	s.manifest.SetVolumePassphrase(projectID, volumeName, req.Passphrase)
	s.persistConfig()

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func (s *Server) handleDeleteVolumePassphrase(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")
	volumeName := chi.URLParam(r, "vol")

	if !s.manifest.ClearVolumePassphrase(projectID, volumeName) {
		http.Error(w, `{"error":"project or volume not found"}`, http.StatusNotFound)
		return
	}

	s.persistConfig()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleSetProjectPassphrase(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")

	if _, ok := s.manifest.GetProject(projectID); !ok {
		http.Error(w, `{"error":"project not found"}`, http.StatusNotFound)
		return
	}

	var req passphraseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Passphrase == "" {
		http.Error(w, `{"error":"passphrase must not be empty"}`, http.StatusBadRequest)
		return
	}

	s.manifest.SetProjectDefaultPassphrase(projectID, req.Passphrase)
	s.persistConfig()

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func (s *Server) handleDeleteProjectPassphrase(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")

	if !s.manifest.ClearProjectDefaultPassphrase(projectID) {
		http.Error(w, `{"error":"project not found"}`, http.StatusNotFound)
		return
	}

	s.persistConfig()
	w.WriteHeader(http.StatusNoContent)
}
