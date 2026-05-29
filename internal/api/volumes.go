package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/bocklucas/dvb-made-easy/internal/storage"
)

func (s *Server) handleListBackups(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	volName := chi.URLParam(r, "vol")

	project, ok := s.manifest.GetProject(id)
	if !ok {
		http.Error(w, `{"error":"project not found"}`, http.StatusNotFound)
		return
	}

	if project.Credentials == nil {
		http.Error(w, `{"error":"no storage credentials configured"}`, http.StatusBadRequest)
		return
	}

	var pattern *regexp.Regexp
	for _, v := range project.Volumes {
		if v.Name == volName && v.BackupPattern != "" {
			compiled, err := storage.CompileBackupPattern(v.BackupPattern)
			if err == nil {
				pattern = compiled
			}
			break
		}
	}

	backend, err := storage.NewBackend(project.Credentials)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	backups, err := backend.ListBackups(r.Context(), pattern)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(backups)
}

func (s *Server) handleCheckVolumesExist(w http.ResponseWriter, r *http.Request) {
	namesParam := r.URL.Query().Get("names")
	if namesParam == "" {
		http.Error(w, `{"error":"names query parameter is required"}`, http.StatusBadRequest)
		return
	}

	exists := make(map[string]bool)
	for _, volName := range strings.Split(namesParam, ",") {
		volName = strings.TrimSpace(volName)
		if volName == "" {
			continue
		}
		_, err := s.dockerClient.InspectVolume(r.Context(), volName)
		exists[volName] = (err == nil)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(exists)
}

