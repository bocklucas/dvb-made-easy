package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/offen/restore-manager/internal/api"
	"github.com/offen/restore-manager/internal/config"
	"github.com/offen/restore-manager/internal/encrypt"
)

func setupTestServer(t *testing.T) (*httptest.Server, *config.Manifest, string) {
	t.Helper()
	dir := t.TempDir()
	key, err := encrypt.LoadOrCreateKey(dir)
	if err != nil {
		t.Fatalf("key: %v", err)
	}
	m := config.NewManifest()
	router := api.NewRouter(m, key, dir, nil)
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	return srv, m, dir
}

func TestImportCompose(t *testing.T) {
	srv, _, _ := setupTestServer(t)

	body := map[string]string{
		"compose_content": "services:\n  db:\n    image: postgres:16\n    volumes:\n      - pgdata:/var/lib/postgresql/data\nvolumes:\n  pgdata:\n",
	}
	data, _ := json.Marshal(body)

	resp, err := http.Post(srv.URL+"/api/projects/import", "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}

	var result struct {
		ID      string `json:"id"`
		Volumes []struct {
			Name             string `json:"name"`
			ComposeService   string `json:"compose_service"`
			ComposeMountPath string `json:"compose_mount_path"`
		} `json:"volumes"`
		DriftHash string `json:"drift_hash"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if result.ID == "" {
		t.Fatal("import must return an ID")
	}

	if len(result.Volumes) != 1 {
		t.Fatalf("volumes: got %d, want 1", len(result.Volumes))
	}

	if result.Volumes[0].Name != "pgdata" {
		t.Fatalf("volume name: got %q, want %q", result.Volumes[0].Name, "pgdata")
	}

	if result.DriftHash == "" {
		t.Fatal("drift hash must be set")
	}

	// Verify the project was created with Source == "paste"
	getResp, err := http.Get(srv.URL + "/api/projects/" + result.ID)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	defer getResp.Body.Close()

	var project struct {
		Source string `json:"source"`
	}
	if err := json.NewDecoder(getResp.Body).Decode(&project); err != nil {
		t.Fatalf("decode project: %v", err)
	}
	if project.Source != "paste" {
		t.Fatalf("source: got %q, want %q", project.Source, "paste")
	}
}

func TestCreateAndListProjects(t *testing.T) {
	srv, _, _ := setupTestServer(t)

	importBody, _ := json.Marshal(map[string]string{
		"compose_content": "services:\n  db:\n    image: postgres:16\n    volumes:\n      - pgdata:/var/lib/postgresql/data\nvolumes:\n  pgdata:\n",
	})
	importResp, err := http.Post(srv.URL+"/api/projects/import", "application/json", bytes.NewReader(importBody))
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	var importResult struct {
		ID string `json:"id"`
	}
	json.NewDecoder(importResp.Body).Decode(&importResult)
	importResp.Body.Close()

	createBody, _ := json.Marshal(map[string]string{
		"id":   importResult.ID,
		"name": "my-homelab",
	})
	createResp, err := http.Post(srv.URL+"/api/projects", "application/json", bytes.NewReader(createBody))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer createResp.Body.Close()

	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("create status: got %d, want 201", createResp.StatusCode)
	}

	listResp, err := http.Get(srv.URL + "/api/projects")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	defer listResp.Body.Close()

	var projects []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	json.NewDecoder(listResp.Body).Decode(&projects)

	if len(projects) != 1 {
		t.Fatalf("projects: got %d, want 1", len(projects))
	}

	if projects[0].Name != "my-homelab" {
		t.Fatalf("name: got %q, want %q", projects[0].Name, "my-homelab")
	}
}

func TestDeleteProject(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	p := m.AddProject(config.Project{
		Name:           "to-delete",
		ComposeContent: "services: {}",
	})

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/projects/"+p.ID, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status: got %d, want 204", resp.StatusCode)
	}

	if len(m.ListProjects()) != 0 {
		t.Fatal("project must be removed after delete")
	}
}

func TestFullProjectLifecycle(t *testing.T) {
	srv, _, _ := setupTestServer(t)

	// 1. Import compose
	importBody, _ := json.Marshal(map[string]string{
		"compose_content": "services:\n  postgres:\n    image: postgres:16\n    volumes:\n      - pgdata:/var/lib/postgresql/data\n  redis:\n    image: redis:7\n    volumes:\n      - redis-data:/data\nvolumes:\n  pgdata:\n  redis-data:\n",
	})

	resp, err := http.Post(srv.URL+"/api/projects/import", "application/json", bytes.NewReader(importBody))
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	var importResult struct {
		ID      string `json:"id"`
		Volumes []struct {
			Name string `json:"name"`
		} `json:"volumes"`
	}
	json.NewDecoder(resp.Body).Decode(&importResult)
	resp.Body.Close()

	if len(importResult.Volumes) != 2 {
		t.Fatalf("import volumes: got %d, want 2", len(importResult.Volumes))
	}

	// 2. Create project with a name
	createBody, _ := json.Marshal(map[string]string{
		"id":   importResult.ID,
		"name": "homelab",
	})
	resp, err = http.Post(srv.URL+"/api/projects", "application/json", bytes.NewReader(createBody))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status: got %d, want 201", resp.StatusCode)
	}

	// 3. List projects
	resp, err = http.Get(srv.URL + "/api/projects")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var projects []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	json.NewDecoder(resp.Body).Decode(&projects)
	resp.Body.Close()

	if len(projects) != 1 {
		t.Fatalf("list: got %d, want 1", len(projects))
	}
	if projects[0].Name != "homelab" {
		t.Fatalf("name: got %q, want %q", projects[0].Name, "homelab")
	}

	// 4. Get single project
	resp, err = http.Get(srv.URL + "/api/projects/" + projects[0].ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	var single struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Volumes []struct {
			Name string `json:"name"`
		} `json:"volumes"`
	}
	json.NewDecoder(resp.Body).Decode(&single)
	resp.Body.Close()

	if len(single.Volumes) != 2 {
		t.Fatalf("get volumes: got %d, want 2", len(single.Volumes))
	}

	// 5. Delete project
	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/projects/"+projects[0].ID, nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status: got %d, want 204", resp.StatusCode)
	}

	// 6. Confirm empty
	resp, err = http.Get(srv.URL + "/api/projects")
	if err != nil {
		t.Fatalf("final list: %v", err)
	}
	var empty []any
	json.NewDecoder(resp.Body).Decode(&empty)
	resp.Body.Close()

	if len(empty) != 0 {
		t.Fatalf("after delete: got %d projects, want 0", len(empty))
	}
}
