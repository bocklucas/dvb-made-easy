package api

import (
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/go-chi/chi/v5"
	"github.com/offen/restore-manager/internal/storage"
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
