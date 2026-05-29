package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/bocklucas/dvb-made-easy/internal/config"
	"github.com/bocklucas/dvb-made-easy/internal/storage"
)

func TestComposeRestoreAccepted(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	backupDir := t.TempDir()
	os.WriteFile(filepath.Join(backupDir, "pgdata-2026-05-15T04-00-00.tar.gz"), []byte("fake"), 0644)
	os.WriteFile(filepath.Join(backupDir, "redis-2026-05-15T04-00-00.tar.gz"), []byte("fake"), 0644)

	p := m.AddProject(config.Project{
		Name:           "test",
		ComposeContent: "services:\n  db:\n    image: postgres\n    volumes:\n      - pgdata:/data\n  cache:\n    image: redis\n    volumes:\n      - redis-data:/data\nvolumes:\n  pgdata:\n  redis-data:\n",
		Volumes: []config.Volume{
			{Name: "pgdata", ComposeService: "db", ComposeMountPath: "/data"},
			{Name: "redis-data", ComposeService: "cache", ComposeMountPath: "/data"},
		},
		Credentials: &storage.Credentials{Type: storage.BackendLocal, Local: &storage.LocalCreds{Path: backupDir}},
	})

	body, _ := json.Marshal(map[string]any{
		"mode": "new_volume",
		"volumes": []map[string]string{
			{
				"volume_name": "pgdata",
				"backup_key":  "pgdata-2026-05-15T04-00-00.tar.gz",
			},
			{
				"volume_name": "redis-data",
				"backup_key":  "redis-2026-05-15T04-00-00.tar.gz",
			},
		},
	})

	resp, err := http.Post(srv.URL+"/api/projects/"+p.ID+"/compose-restore", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status: got %d, want 202", resp.StatusCode)
	}

	var result struct {
		RestoreToken string `json:"restore_token"`
		Status       string `json:"status"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.RestoreToken == "" {
		t.Fatal("expected non-empty restore_token")
	}
	if result.Status != "queued" {
		t.Fatalf("status: got %q, want %q", result.Status, "queued")
	}
}

func TestComposeRestoreProjectNotFound(t *testing.T) {
	srv, _, _ := setupTestServer(t)

	body, _ := json.Marshal(map[string]any{
		"mode": "new_volume",
		"volumes": []map[string]string{
			{
				"volume_name": "pgdata",
				"backup_key":  "backup.tar.gz",
			},
		},
	})

	resp, err := http.Post(srv.URL+"/api/projects/nonexistent/compose-restore", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", resp.StatusCode)
	}
}

func TestComposeRestoreInvalidMode(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	backupDir := t.TempDir()

	p := m.AddProject(config.Project{
		Name:           "test",
		ComposeContent: "services:\n  db:\n    image: postgres\n    volumes:\n      - pgdata:/data\nvolumes:\n  pgdata:\n",
		Volumes:        []config.Volume{{Name: "pgdata", ComposeService: "db", ComposeMountPath: "/data"}},
		Credentials:    &storage.Credentials{Type: storage.BackendLocal, Local: &storage.LocalCreds{Path: backupDir}},
	})

	body, _ := json.Marshal(map[string]any{
		"mode": "invalid",
		"volumes": []map[string]string{
			{
				"volume_name": "pgdata",
				"backup_key":  "backup.tar.gz",
			},
		},
	})

	resp, err := http.Post(srv.URL+"/api/projects/"+p.ID+"/compose-restore", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", resp.StatusCode)
	}
}

func TestComposeRestoreEmptyVolumes(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	backupDir := t.TempDir()

	p := m.AddProject(config.Project{
		Name:           "test",
		ComposeContent: "services:\n  db:\n    image: postgres\n    volumes:\n      - pgdata:/data\nvolumes:\n  pgdata:\n",
		Volumes:        []config.Volume{{Name: "pgdata", ComposeService: "db", ComposeMountPath: "/data"}},
		Credentials:    &storage.Credentials{Type: storage.BackendLocal, Local: &storage.LocalCreds{Path: backupDir}},
	})

	body, _ := json.Marshal(map[string]any{
		"mode":    "new_volume",
		"volumes": []map[string]string{},
	})

	resp, err := http.Post(srv.URL+"/api/projects/"+p.ID+"/compose-restore", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", resp.StatusCode)
	}
}
