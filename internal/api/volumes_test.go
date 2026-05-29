package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/bocklucas/dvb-made-easy/internal/config"
)

func TestListBackups(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "backup-2024-01-15T10-30-00.tar.gz"), []byte("data1"), 0644)
	os.WriteFile(filepath.Join(dir, "backup-2024-01-16T08-00-00.tar.gz"), []byte("data22"), 0644)
	os.WriteFile(filepath.Join(dir, "backup-2024-01-14T12-00-00.tar.gz.gpg"), []byte("enc"), 0644)
	os.WriteFile(filepath.Join(dir, "random-file.txt"), []byte("nope"), 0644)

	p := m.AddProject(config.Project{
		Name:           "test-project",
		ComposeContent: "services:\n  db:\n    image: postgres:16\n    volumes:\n      - pgdata:/var/lib/postgresql/data\nvolumes:\n  pgdata:\n",
		Volumes: []config.Volume{
			{Name: "pgdata", ComposeService: "db", ComposeMountPath: "/var/lib/postgresql/data"},
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

	resp, err := http.Get(srv.URL + "/api/projects/" + p.ID + "/volumes/pgdata/backups")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}

	var backups []struct {
		Key         string `json:"key"`
		Size        int64  `json:"size"`
		IsEncrypted bool   `json:"is_encrypted"`
	}
	json.NewDecoder(resp.Body).Decode(&backups)

	if len(backups) != 3 {
		t.Fatalf("backups: got %d, want 3", len(backups))
	}

	if backups[0].Key != "backup-2024-01-16T08-00-00.tar.gz" {
		t.Fatalf("first: got %q, want newest", backups[0].Key)
	}

	if !backups[2].IsEncrypted {
		t.Fatal("last backup must be encrypted (.gpg)")
	}
}

func TestListBackupsNoCredentials(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	p := m.AddProject(config.Project{
		Name:           "test-project",
		ComposeContent: "services: {}",
		Volumes: []config.Volume{
			{Name: "pgdata", ComposeService: "db", ComposeMountPath: "/var/lib/postgresql/data"},
		},
	})

	resp, err := http.Get(srv.URL + "/api/projects/" + p.ID + "/volumes/pgdata/backups")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", resp.StatusCode)
	}
}

func TestListBackupsProjectNotFound(t *testing.T) {
	srv, _, _ := setupTestServer(t)

	resp, err := http.Get(srv.URL + "/api/projects/nonexistent/volumes/pgdata/backups")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", resp.StatusCode)
	}
}
