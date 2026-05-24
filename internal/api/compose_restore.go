package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/offen/restore-manager/internal/compose"
	"github.com/offen/restore-manager/internal/restore"
	"github.com/offen/restore-manager/internal/storage"
)

type composeRestoreRequest struct {
	Mode         string                 `json:"mode"`
	Timestamp    string                 `json:"timestamp"`
	Volumes      []composeVolumeRestore `json:"volumes"`
	ContainerIDs []string               `json:"container_ids"`
}

type composeVolumeRestore struct {
	VolumeName       string `json:"volume_name"`
	BackupKey        string `json:"backup_key"`
	Passphrase       string `json:"passphrase"`
	TargetVolumeName string `json:"target_volume_name"`
}

func (s *Server) handleComposeRestore(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")

	project, ok := s.manifest.GetProject(projectID)
	if !ok {
		http.Error(w, `{"error":"project not found"}`, http.StatusNotFound)
		return
	}

	if project.Credentials == nil {
		http.Error(w, `{"error":"no storage credentials configured"}`, http.StatusBadRequest)
		return
	}

	var req composeRestoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	mode := restore.RestoreMode(req.Mode)
	if mode != restore.ModeNewVolume && mode != restore.ModeFullStack {
		http.Error(w, `{"error":"mode must be new_volume or full_stack"}`, http.StatusBadRequest)
		return
	}

	if len(req.Volumes) == 0 {
		http.Error(w, `{"error":"at least one volume is required"}`, http.StatusBadRequest)
		return
	}

	// Build a volume passphrase lookup from manifest.
	volumePassphrases := make(map[string]string, len(project.Volumes))
	for _, v := range project.Volumes {
		if v.Passphrase != "" {
			volumePassphrases[v.Name] = v.Passphrase
		}
	}

	for i, v := range req.Volumes {
		if v.BackupKey == "" {
			http.Error(w, `{"error":"backup_key is required for each volume"}`, http.StatusBadRequest)
			return
		}
		if strings.HasSuffix(v.BackupKey, ".gpg") && v.Passphrase == "" {
			// Volume-specific passphrase takes priority over project default.
			if p, ok := volumePassphrases[v.VolumeName]; ok {
				req.Volumes[i].Passphrase = p
			} else if project.DefaultPassphrase != "" {
				req.Volumes[i].Passphrase = project.DefaultPassphrase
			}
		}
	}

	for _, v := range req.Volumes {
		if strings.HasSuffix(v.BackupKey, ".gpg") && v.Passphrase == "" {
			http.Error(w, `{"error":"passphrase required for encrypted backup"}`, http.StatusBadRequest)
			return
		}
	}

	backend, err := storage.NewBackend(project.Credentials)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	backups, err := backend.ListBackups(r.Context(), nil)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Build a size lookup map.
	sizeByKey := make(map[string]int64, len(backups))
	for _, b := range backups {
		sizeByKey[b.Key] = b.Size
	}

	parsed, _ := compose.Parse(project.ComposeContent)
	var services []string
	var dependsOn map[string][]string
	if parsed != nil {
		services = parsed.Services
		dependsOn = parsed.DependsOn
	}

	token := "rst-" + uuid.New().String()

	volumes := make([]restore.VolumeRestore, 0, len(req.Volumes))
	for _, v := range req.Volumes {
		targetName := v.TargetVolumeName
		if mode == restore.ModeFullStack && targetName == "" {
			targetName = v.VolumeName
		}
		volumes = append(volumes, restore.VolumeRestore{
			VolumeName: v.VolumeName,
			BackupKey:  v.BackupKey,
			Passphrase: v.Passphrase,
			TargetName: targetName,
			BackupSize: sizeByKey[v.BackupKey],
		})
	}

	composeReq := restore.ComposeRestoreRequest{
		Token:        token,
		Project:      project.Name,
		Services:     services,
		DependsOn:    dependsOn,
		Mode:         mode,
		Volumes:      volumes,
		ContainerIDs: req.ContainerIDs,
	}

	s.broadcaster.Register(token)

	if s.dockerClient != nil {
		orch := restore.NewOrchestrator(s.dockerClient, s.broadcaster)
		go orch.RunCompose(context.Background(), composeReq, backend)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"restore_token": token,
		"status":        "queued",
	})
}
