package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/bocklucas/dvb-made-easy/internal/config"
)

const baseCompose = `services:
  db:
    image: postgres:16
    volumes:
      - pgdata:/var/lib/postgresql/data
  cache:
    image: redis:7
    volumes:
      - redis-data:/data
volumes:
  pgdata:
  redis-data:
`

func TestUpdateCompose_UnchangedVolumes(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	p := m.AddProject(config.Project{
		Name:           "test",
		ComposeContent: baseCompose,
		Volumes: []config.Volume{
			{Name: "pgdata", ComposeService: "db", ComposeMountPath: "/var/lib/postgresql/data", BackupPattern: "pg-*.tar.gz", Passphrase: "secret"},
			{Name: "redis-data", ComposeService: "cache", ComposeMountPath: "/data"},
		},
	})

	body, _ := json.Marshal(map[string]string{
		"compose_content": baseCompose,
	})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/projects/"+p.ID+"/compose", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}

	var diff struct {
		Added     []struct {
			Name string `json:"name"`
		} `json:"added"`
		Removed []struct {
			Name string `json:"name"`
		} `json:"removed"`
		Unchanged []struct {
			Name string `json:"name"`
		} `json:"unchanged"`
	}
	json.NewDecoder(resp.Body).Decode(&diff)

	if len(diff.Added) != 0 {
		t.Fatalf("added: got %d, want 0", len(diff.Added))
	}
	if len(diff.Removed) != 0 {
		t.Fatalf("removed: got %d, want 0", len(diff.Removed))
	}
	if len(diff.Unchanged) != 2 {
		t.Fatalf("unchanged: got %d, want 2", len(diff.Unchanged))
	}

	// Verify passphrase and backup pattern are preserved.
	updated, _ := m.GetProject(p.ID)
	for _, v := range updated.Volumes {
		if v.Name == "pgdata" {
			if v.Passphrase != "secret" {
				t.Fatalf("passphrase: got %q, want %q", v.Passphrase, "secret")
			}
			if v.BackupPattern != "pg-*.tar.gz" {
				t.Fatalf("backup_pattern: got %q, want %q", v.BackupPattern, "pg-*.tar.gz")
			}
		}
	}
}

func TestUpdateCompose_NewVolume(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	// Start with only pgdata in Volumes — redis-data is in compose but not in project volumes.
	p := m.AddProject(config.Project{
		Name:           "test",
		ComposeContent: baseCompose,
		Volumes: []config.Volume{
			{Name: "pgdata", ComposeService: "db", ComposeMountPath: "/var/lib/postgresql/data"},
		},
	})

	body, _ := json.Marshal(map[string]string{
		"compose_content": baseCompose,
	})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/projects/"+p.ID+"/compose", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}

	var diff struct {
		Added []struct {
			Name string `json:"name"`
		} `json:"added"`
		Removed []struct {
			Name string `json:"name"`
		} `json:"removed"`
		Unchanged []struct {
			Name string `json:"name"`
		} `json:"unchanged"`
	}
	json.NewDecoder(resp.Body).Decode(&diff)

	if len(diff.Added) != 1 {
		t.Fatalf("added: got %d, want 1", len(diff.Added))
	}
	if diff.Added[0].Name != "redis-data" {
		t.Fatalf("added[0].name: got %q, want %q", diff.Added[0].Name, "redis-data")
	}
	if len(diff.Unchanged) != 1 {
		t.Fatalf("unchanged: got %d, want 1", len(diff.Unchanged))
	}
}

func TestUpdateCompose_RemovedVolume(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	p := m.AddProject(config.Project{
		Name:           "test",
		ComposeContent: baseCompose,
		Volumes: []config.Volume{
			{Name: "pgdata", ComposeService: "db", ComposeMountPath: "/var/lib/postgresql/data"},
			{Name: "redis-data", ComposeService: "cache", ComposeMountPath: "/data"},
		},
	})

	smallerCompose := `services:
  db:
    image: postgres:16
    volumes:
      - pgdata:/var/lib/postgresql/data
volumes:
  pgdata:
`

	body, _ := json.Marshal(map[string]string{
		"compose_content": smallerCompose,
	})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/projects/"+p.ID+"/compose", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}

	var diff struct {
		Added []struct {
			Name string `json:"name"`
		} `json:"added"`
		Removed []struct {
			Name string `json:"name"`
		} `json:"removed"`
		Unchanged []struct {
			Name string `json:"name"`
		} `json:"unchanged"`
	}
	json.NewDecoder(resp.Body).Decode(&diff)

	if len(diff.Removed) != 1 {
		t.Fatalf("removed: got %d, want 1", len(diff.Removed))
	}
	if diff.Removed[0].Name != "redis-data" {
		t.Fatalf("removed[0].name: got %q, want %q", diff.Removed[0].Name, "redis-data")
	}

	// The removed volume must still be in manifest (not auto-deleted).
	updated, _ := m.GetProject(p.ID)
	found := false
	for _, v := range updated.Volumes {
		if v.Name == "redis-data" {
			found = true
		}
	}
	if !found {
		t.Fatal("removed volume must still be in manifest until user confirms removal")
	}
}

func TestUpdateCompose_ExistingConfigPreserved(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	p := m.AddProject(config.Project{
		Name:           "test",
		ComposeContent: baseCompose,
		Volumes: []config.Volume{
			{Name: "pgdata", BackupPattern: "pg-*.tar.gz", Passphrase: "mypass", ComposeService: "db", ComposeMountPath: "/var/lib/postgresql/data"},
		},
	})

	// Update to compose that includes both volumes.
	body, _ := json.Marshal(map[string]string{
		"compose_content": baseCompose,
	})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/projects/"+p.ID+"/compose", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}

	updated, _ := m.GetProject(p.ID)
	for _, v := range updated.Volumes {
		if v.Name == "pgdata" {
			if v.BackupPattern != "pg-*.tar.gz" {
				t.Fatalf("BackupPattern: got %q, want %q", v.BackupPattern, "pg-*.tar.gz")
			}
			if v.Passphrase != "mypass" {
				t.Fatalf("Passphrase: got %q, want %q", v.Passphrase, "mypass")
			}
		}
	}
}

func TestConfirmVolumeRemoval(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	p := m.AddProject(config.Project{
		Name:           "test",
		ComposeContent: baseCompose,
		Volumes: []config.Volume{
			{Name: "pgdata", ComposeService: "db"},
			{Name: "redis-data", ComposeService: "cache", Passphrase: "pass"},
		},
	})

	body, _ := json.Marshal(map[string][]string{
		"volumes": {"redis-data"},
	})
	resp, err := http.Post(srv.URL+"/api/projects/"+p.ID+"/compose/confirm-removal", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status: got %d, want 204", resp.StatusCode)
	}

	updated, _ := m.GetProject(p.ID)
	if len(updated.Volumes) != 1 {
		t.Fatalf("volumes after removal: got %d, want 1", len(updated.Volumes))
	}
	if updated.Volumes[0].Name != "pgdata" {
		t.Fatalf("remaining volume: got %q, want %q", updated.Volumes[0].Name, "pgdata")
	}
}

func TestUpdateCompose_ProjectNotFound(t *testing.T) {
	srv, _, _ := setupTestServer(t)

	body, _ := json.Marshal(map[string]string{"compose_content": "services: {}"})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/projects/nonexistent/compose", bytes.NewReader(body))
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

func TestUpdateCompose_EmptyContent(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	p := m.AddProject(config.Project{Name: "test", ComposeContent: baseCompose})

	body, _ := json.Marshal(map[string]string{"compose_content": ""})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/projects/"+p.ID+"/compose", bytes.NewReader(body))
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

func TestConfirmVolumeRemoval_ProjectNotFound(t *testing.T) {
	srv, _, _ := setupTestServer(t)

	body, _ := json.Marshal(map[string][]string{"volumes": {"vol1"}})
	resp, err := http.Post(srv.URL+"/api/projects/nonexistent/compose/confirm-removal", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", resp.StatusCode)
	}
}

func TestConfirmVolumeRemoval_EmptyList(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	p := m.AddProject(config.Project{
		Name:           "test",
		ComposeContent: baseCompose,
		Volumes: []config.Volume{
			{Name: "pgdata", ComposeService: "db"},
		},
	})

	body, _ := json.Marshal(map[string][]string{"volumes": {}})
	resp, err := http.Post(srv.URL+"/api/projects/"+p.ID+"/compose/confirm-removal", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", resp.StatusCode)
	}
}
