package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/offen/restore-manager/internal/config"
)

const giteatComposeContent = `services:
  server:
    image: gitea/gitea:1.23
    volumes:
      - data:/var/lib/gitea
      - config:/etc/gitea
  backup_data:
    image: offen/docker-volume-backup:v2
    environment:
      BACKUP_FILENAME: "backup-data-%Y-%m-%dT%H-%M-%S.{{ .Extension }}"
    volumes:
      - data:/backup/data:ro
      - archive:/archive
  backup_config:
    image: offen/docker-volume-backup:v2
    environment:
      BACKUP_FILENAME: "backup-config-%Y-%m-%dT%H-%M-%S.{{ .Extension }}"
    volumes:
      - config:/backup/config:ro
      - archive:/archive
volumes:
  data:
  config:
  archive:
`

func TestBackupTimestampsGrouped(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "backup-data-2026-05-22T03-00-00.tar.gz"), []byte("data1"), 0644)
	os.WriteFile(filepath.Join(dir, "backup-config-2026-05-22T03-00-00.tar.gz"), []byte("data2"), 0644)
	os.WriteFile(filepath.Join(dir, "backup-data-2026-05-21T03-00-00.tar.gz"), []byte("data3"), 0644)

	p := m.AddProject(config.Project{
		Name:           "gitea",
		ComposeContent: giteatComposeContent,
		Volumes: []config.Volume{
			{Name: "data", ComposeService: "server", ComposeMountPath: "/var/lib/gitea", BackupPattern: "backup-data-%Y-%m-%dT%H-%M-%S.{{ .Extension }}"},
			{Name: "config", ComposeService: "server", ComposeMountPath: "/etc/gitea", BackupPattern: "backup-config-%Y-%m-%dT%H-%M-%S.{{ .Extension }}"},
		},
	})

	credsBody, _ := json.Marshal(map[string]any{
		"type": "local",
		"local": map[string]string{
			"path": dir,
		},
	})
	credsResp, _ := http.Post(srv.URL+"/api/projects/"+p.ID+"/credentials", "application/json", bytes.NewReader(credsBody))
	credsResp.Body.Close()

	resp, err := http.Get(srv.URL + "/api/projects/" + p.ID + "/backup-timestamps")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}

	var groups []struct {
		Timestamp string `json:"timestamp"`
		Backups   []struct {
			VolumeName  string `json:"volume_name"`
			Key         string `json:"key"`
			Size        int64  `json:"size"`
			IsEncrypted bool   `json:"is_encrypted"`
		} `json:"backups"`
		Complete bool `json:"complete"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&groups); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(groups) != 2 {
		t.Fatalf("groups: got %d, want 2", len(groups))
	}

	// Sorted newest first
	if groups[0].Timestamp != "2026-05-22T03-00-00" {
		t.Fatalf("first group timestamp: got %q, want %q", groups[0].Timestamp, "2026-05-22T03-00-00")
	}
	if groups[1].Timestamp != "2026-05-21T03-00-00" {
		t.Fatalf("second group timestamp: got %q, want %q", groups[1].Timestamp, "2026-05-21T03-00-00")
	}

	// First group is complete (both volumes present)
	if !groups[0].Complete {
		t.Fatal("first group must be complete (both volumes have backups)")
	}
	if len(groups[0].Backups) != 2 {
		t.Fatalf("first group backups: got %d, want 2", len(groups[0].Backups))
	}

	// Second group is incomplete (only data volume)
	if groups[1].Complete {
		t.Fatal("second group must be incomplete (only data volume)")
	}
	if len(groups[1].Backups) != 1 {
		t.Fatalf("second group backups: got %d, want 1", len(groups[1].Backups))
	}
}

func TestBackupTimestampsNoCredentials(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	p := m.AddProject(config.Project{
		Name:           "gitea",
		ComposeContent: giteatComposeContent,
		Volumes: []config.Volume{
			{Name: "data", ComposeService: "server", ComposeMountPath: "/var/lib/gitea"},
		},
	})

	resp, err := http.Get(srv.URL + "/api/projects/" + p.ID + "/backup-timestamps")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", resp.StatusCode)
	}
}

func TestBackupTimestampsProjectNotFound(t *testing.T) {
	srv, _, _ := setupTestServer(t)

	resp, err := http.Get(srv.URL + "/api/projects/nonexistent/backup-timestamps")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", resp.StatusCode)
	}
}
