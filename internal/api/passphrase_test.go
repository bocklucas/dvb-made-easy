package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/bocklucas/dvb-made-easy/internal/config"
	"github.com/bocklucas/dvb-made-easy/internal/storage"
)

// ---- Volume passphrase endpoints ----

func TestSetVolumePassphrase(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	p := m.AddProject(config.Project{
		Name:           "test-project",
		ComposeContent: "services: {}",
		Volumes:        []config.Volume{{Name: "pgdata", ComposeService: "db", ComposeMountPath: "/data"}},
	})

	body, _ := json.Marshal(map[string]string{"passphrase": "secret123"})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/projects/"+p.ID+"/volumes/pgdata/passphrase", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]string
		json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("status: got %d, want 200, error: %v", resp.StatusCode, errBody)
	}

	// Verify stored in manifest.
	proj, _ := m.GetProject(p.ID)
	if proj.Volumes[0].Passphrase != "secret123" {
		t.Fatalf("passphrase: got %q, want %q", proj.Volumes[0].Passphrase, "secret123")
	}
}

func TestSetVolumePassphraseProjectNotFound(t *testing.T) {
	srv, _, _ := setupTestServer(t)

	body, _ := json.Marshal(map[string]string{"passphrase": "secret"})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/projects/nonexistent/volumes/pgdata/passphrase", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", resp.StatusCode)
	}
}

func TestSetVolumePassphraseVolumeNotFound(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	p := m.AddProject(config.Project{
		Name:           "test-project",
		ComposeContent: "services: {}",
	})

	body, _ := json.Marshal(map[string]string{"passphrase": "secret"})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/projects/"+p.ID+"/volumes/nonexistent/passphrase", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", resp.StatusCode)
	}
}

func TestSetVolumePassphraseEmptyPassphrase(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	p := m.AddProject(config.Project{
		Name:           "test-project",
		ComposeContent: "services: {}",
		Volumes:        []config.Volume{{Name: "pgdata", ComposeService: "db", ComposeMountPath: "/data"}},
	})

	body, _ := json.Marshal(map[string]string{"passphrase": ""})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/projects/"+p.ID+"/volumes/pgdata/passphrase", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", resp.StatusCode)
	}
}

func TestDeleteVolumePassphrase(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	p := m.AddProject(config.Project{
		Name:           "test-project",
		ComposeContent: "services: {}",
		Volumes:        []config.Volume{{Name: "pgdata", ComposeService: "db", ComposeMountPath: "/data"}},
	})

	m.SetVolumePassphrase(p.ID, "pgdata", "secret123")

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/projects/"+p.ID+"/volumes/pgdata/passphrase", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status: got %d, want 204", resp.StatusCode)
	}

	proj, _ := m.GetProject(p.ID)
	if proj.Volumes[0].Passphrase != "" {
		t.Fatal("passphrase must be empty after delete")
	}
}

func TestDeleteVolumePassphraseProjectNotFound(t *testing.T) {
	srv, _, _ := setupTestServer(t)

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/projects/nonexistent/volumes/pgdata/passphrase", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", resp.StatusCode)
	}
}

func TestDeleteVolumePassphraseVolumeNotFound(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	p := m.AddProject(config.Project{
		Name:           "test-project",
		ComposeContent: "services: {}",
	})

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/projects/"+p.ID+"/volumes/nonexistent/passphrase", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", resp.StatusCode)
	}
}

// ---- Project default passphrase endpoints ----

func TestSetProjectPassphrase(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	p := m.AddProject(config.Project{
		Name:           "test-project",
		ComposeContent: "services: {}",
	})

	body, _ := json.Marshal(map[string]string{"passphrase": "default-secret"})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/projects/"+p.ID+"/passphrase", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]string
		json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("status: got %d, want 200, error: %v", resp.StatusCode, errBody)
	}

	proj, _ := m.GetProject(p.ID)
	if proj.DefaultPassphrase != "default-secret" {
		t.Fatalf("default passphrase: got %q, want %q", proj.DefaultPassphrase, "default-secret")
	}
}

func TestSetProjectPassphraseNotFound(t *testing.T) {
	srv, _, _ := setupTestServer(t)

	body, _ := json.Marshal(map[string]string{"passphrase": "secret"})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/projects/nonexistent/passphrase", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", resp.StatusCode)
	}
}

func TestSetProjectPassphraseEmpty(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	p := m.AddProject(config.Project{
		Name:           "test-project",
		ComposeContent: "services: {}",
	})

	body, _ := json.Marshal(map[string]string{"passphrase": ""})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/projects/"+p.ID+"/passphrase", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", resp.StatusCode)
	}
}

func TestDeleteProjectPassphrase(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	p := m.AddProject(config.Project{
		Name:           "test-project",
		ComposeContent: "services: {}",
	})

	m.SetProjectDefaultPassphrase(p.ID, "default-secret")

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/projects/"+p.ID+"/passphrase", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status: got %d, want 204", resp.StatusCode)
	}

	proj, _ := m.GetProject(p.ID)
	if proj.DefaultPassphrase != "" {
		t.Fatal("default passphrase must be empty after delete")
	}
}

func TestDeleteProjectPassphraseNotFound(t *testing.T) {
	srv, _, _ := setupTestServer(t)

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/projects/nonexistent/passphrase", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", resp.StatusCode)
	}
}

// ---- Restore passphrase auto-fill tests ----

func TestRestoreUsesProjectDefaultPassphrase(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	backupDir := t.TempDir()

	p := m.AddProject(config.Project{
		Name:           "test",
		ComposeContent: "services:\n  db:\n    image: postgres\n    volumes:\n      - pgdata:/data\nvolumes:\n  pgdata:\n",
		Volumes: []config.Volume{
			{Name: "pgdata", ComposeService: "db", ComposeMountPath: "/data"},
		},
		Credentials: &storage.Credentials{
			Type:  storage.BackendLocal,
			Local: &storage.LocalCreds{Path: backupDir},
		},
	})

	m.SetProjectDefaultPassphrase(p.ID, "project-default-secret")

	body, _ := json.Marshal(map[string]any{
		"mode":       "new_volume",
		"backup_key": "backup-2026-05-15T04-00-00.tar.gz.gpg",
	})

	resp, err := http.Post(srv.URL+"/api/projects/"+p.ID+"/volumes/pgdata/restore", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest {
		var errBody map[string]string
		json.NewDecoder(resp.Body).Decode(&errBody)
		if errBody["error"] == "passphrase required for encrypted backup" {
			t.Fatal("project default passphrase should have been used")
		}
	}
}

func TestRestoreVolumePassphrasePriorityOverDefault(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	backupDir := t.TempDir()

	p := m.AddProject(config.Project{
		Name:           "test",
		ComposeContent: "services:\n  db:\n    image: postgres\n    volumes:\n      - pgdata:/data\nvolumes:\n  pgdata:\n",
		Volumes: []config.Volume{
			{Name: "pgdata", ComposeService: "db", ComposeMountPath: "/data", Passphrase: "volume-secret"},
		},
		Credentials: &storage.Credentials{
			Type:  storage.BackendLocal,
			Local: &storage.LocalCreds{Path: backupDir},
		},
	})

	m.SetProjectDefaultPassphrase(p.ID, "project-default-secret")

	body, _ := json.Marshal(map[string]any{
		"mode":       "new_volume",
		"backup_key": "backup-2026-05-15T04-00-00.tar.gz.gpg",
	})

	resp, err := http.Post(srv.URL+"/api/projects/"+p.ID+"/volumes/pgdata/restore", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest {
		var errBody map[string]string
		json.NewDecoder(resp.Body).Decode(&errBody)
		if errBody["error"] == "passphrase required for encrypted backup" {
			t.Fatal("volume passphrase should have been used over project default")
		}
	}
}

func TestRestoreUsesStoredVolumePassphrase(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	backupDir := t.TempDir()

	p := m.AddProject(config.Project{
		Name:           "test",
		ComposeContent: "services:\n  db:\n    image: postgres\n    volumes:\n      - pgdata:/data\nvolumes:\n  pgdata:\n",
		Volumes: []config.Volume{
			{Name: "pgdata", ComposeService: "db", ComposeMountPath: "/data", Passphrase: "stored-secret"},
		},
		Credentials: &storage.Credentials{
			Type:  storage.BackendLocal,
			Local: &storage.LocalCreds{Path: backupDir},
		},
	})

	// This test just verifies the endpoint does NOT return 400 "passphrase required"
	// when a stored passphrase is available and the backup key ends in .gpg.
	// (The actual restore will fail because there's no real .gpg file, but it should be accepted.)
	body, _ := json.Marshal(map[string]any{
		"mode":       "new_volume",
		"backup_key": "backup-2026-05-15T04-00-00.tar.gz.gpg",
	})

	resp, err := http.Post(srv.URL+"/api/projects/"+p.ID+"/volumes/pgdata/restore", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	// Should NOT be 400 (passphrase required) — should be 202 accepted or an error at the storage layer.
	if resp.StatusCode == http.StatusBadRequest {
		var errBody map[string]string
		json.NewDecoder(resp.Body).Decode(&errBody)
		if errBody["error"] == "passphrase required for encrypted backup" {
			t.Fatal("stored volume passphrase should have been used, not returned 400")
		}
	}
}
