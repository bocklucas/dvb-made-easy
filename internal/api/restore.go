package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/offen/restore-manager/internal/compose"
	"github.com/offen/restore-manager/internal/restore"
	"github.com/offen/restore-manager/internal/storage"
)

type restoreRequest struct {
	Mode             string   `json:"mode"`
	BackupKey        string   `json:"backup_key"`
	TargetVolumeName string   `json:"target_volume_name"`
	Passphrase       string   `json:"passphrase"`
	ContainerIDs     []string `json:"container_ids"`
}

func (s *Server) handleRestore(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")
	volumeName := chi.URLParam(r, "vol")

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

	if project.Credentials == nil {
		http.Error(w, `{"error":"no storage credentials configured"}`, http.StatusBadRequest)
		return
	}

	var req restoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.BackupKey == "" {
		http.Error(w, `{"error":"backup_key is required"}`, http.StatusBadRequest)
		return
	}

	mode := restore.RestoreMode(req.Mode)
	if mode != restore.ModeNewVolume && mode != restore.ModeFullStack {
		http.Error(w, `{"error":"mode must be new_volume or full_stack"}`, http.StatusBadRequest)
		return
	}

	if strings.HasSuffix(req.BackupKey, ".gpg") && req.Passphrase == "" {
		// Look up stored passphrase: volume-specific takes priority over project default.
		for _, v := range project.Volumes {
			if v.Name == volumeName && v.Passphrase != "" {
				req.Passphrase = v.Passphrase
				break
			}
		}
		if req.Passphrase == "" && project.DefaultPassphrase != "" {
			req.Passphrase = project.DefaultPassphrase
		}
	}

	if strings.HasSuffix(req.BackupKey, ".gpg") && req.Passphrase == "" {
		http.Error(w, `{"error":"passphrase required for encrypted backup"}`, http.StatusBadRequest)
		return
	}

	backend, err := storage.NewBackend(project.Credentials)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var volPattern *regexp.Regexp
	for _, v := range project.Volumes {
		if v.Name == volumeName && v.BackupPattern != "" {
			compiled, err := storage.CompileBackupPattern(v.BackupPattern)
			if err == nil {
				volPattern = compiled
			}
			break
		}
	}

	backups, err := backend.ListBackups(r.Context(), volPattern)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var backupSize int64
	for _, b := range backups {
		if b.Key == req.BackupKey {
			backupSize = b.Size
			break
		}
	}

	parsed, _ := compose.Parse(project.ComposeContent)
	var services []string
	var dependsOn map[string][]string
	if parsed != nil {
		services = parsed.Services
		dependsOn = parsed.DependsOn
	}

	token := "rst-" + uuid.New().String()

	restoreReq := restore.RestoreRequest{
		Token:          token,
		Project:        project.Name,
		StackName:      project.StackName(),
		Services:       services,
		DependsOn:      dependsOn,
		VolumeName:     volumeName,
		BackupKey:      req.BackupKey,
		Mode:           mode,
		TargetName:     req.TargetVolumeName,
		Passphrase:     req.Passphrase,
		ContainerIDs:   req.ContainerIDs,
		BackupSize:     backupSize,
		DeploymentMode: project.DeploymentMode,
	}

	s.broadcaster.Register(token)

	if s.dockerClient != nil {
		orch := restore.NewOrchestrator(s.dockerClient, s.broadcaster)
		if s.stagingDir != "" {
			orch.SetStagingDir(s.stagingDir)
		}
		go orch.Run(context.Background(), restoreReq, backend)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"restore_token": token,
		"status":        "queued",
	})
}

func (s *Server) handleRestoreStream(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	log.Printf("[api] SSE stream requested for token %s", token)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	ch := s.broadcaster.Subscribe(token)
	defer s.broadcaster.Unsubscribe(token, ch)

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	ctx := r.Context()
	for {
		select {
		case event, ok := <-ch:
			if !ok {
				log.Printf("[api] SSE channel closed for token %s", token)
				return
			}
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()

			if event.Step == "complete" || event.Step == "failed" {
				log.Printf("[api] SSE stream ending for token %s (step=%s)", token, event.Step)
				return
			}
		case <-heartbeat.C:
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		case <-ctx.Done():
			log.Printf("[api] SSE client disconnected for token %s", token)
			return
		}
	}
}
