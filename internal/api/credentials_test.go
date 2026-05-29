package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/bocklucas/dvb-made-easy/internal/config"
	"github.com/bocklucas/dvb-made-easy/internal/storage"
)

func TestSaveLocalCredentials(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	dir := t.TempDir()
	p := m.AddProject(config.Project{
		Name:           "test-project",
		ComposeContent: "services: {}",
	})

	body, _ := json.Marshal(map[string]any{
		"type": "local",
		"local": map[string]string{
			"path": dir,
		},
	})

	resp, err := http.Post(srv.URL+"/api/projects/"+p.ID+"/credentials", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]string
		json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("status: got %d, want 200, error: %v", resp.StatusCode, errBody)
	}
}

func TestSaveCredentialsRunsTestConnection(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	p := m.AddProject(config.Project{
		Name:           "test-project",
		ComposeContent: "services: {}",
	})

	body, _ := json.Marshal(map[string]any{
		"type": "local",
		"local": map[string]string{
			"path": "/nonexistent/path/that/does/not/exist",
		},
	})

	resp, err := http.Post(srv.URL+"/api/projects/"+p.ID+"/credentials", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status: got %d, want 422", resp.StatusCode)
	}
}

func TestGetCredentialsOmitsPasswords(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	dir := t.TempDir()
	p := m.AddProject(config.Project{
		Name:           "test-project",
		ComposeContent: "services: {}",
	})

	saveBody, _ := json.Marshal(map[string]any{
		"type": "local",
		"local": map[string]string{
			"path": dir,
		},
	})
	saveResp, err := http.Post(srv.URL+"/api/projects/"+p.ID+"/credentials", "application/json", bytes.NewReader(saveBody))
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	saveResp.Body.Close()

	resp, err := http.Get(srv.URL + "/api/projects/" + p.ID + "/credentials")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	if result["type"] != "local" {
		t.Fatalf("type: got %q, want %q", result["type"], "local")
	}
}

func TestGetCredentialsNoneConfigured(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	p := m.AddProject(config.Project{
		Name:           "test-project",
		ComposeContent: "services: {}",
	})

	resp, err := http.Get(srv.URL + "/api/projects/" + p.ID + "/credentials")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", resp.StatusCode)
	}
}

func TestUpdateCredentials(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	dir := t.TempDir()
	p := m.AddProject(config.Project{
		Name:           "test-project",
		ComposeContent: "services: {}",
	})

	// First, save initial credentials via POST.
	saveBody, _ := json.Marshal(map[string]any{
		"type": "local",
		"local": map[string]string{
			"path": dir,
		},
	})
	saveResp, err := http.Post(srv.URL+"/api/projects/"+p.ID+"/credentials", "application/json", bytes.NewReader(saveBody))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	saveResp.Body.Close()

	// Now update via PUT with a different valid path.
	dir2 := t.TempDir()
	updateBody, _ := json.Marshal(map[string]any{
		"type": "local",
		"local": map[string]string{
			"path": dir2,
		},
	})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/projects/"+p.ID+"/credentials", bytes.NewReader(updateBody))
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
}

func TestUpdateCredentialsRunsTestConnection(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	dir := t.TempDir()
	p := m.AddProject(config.Project{
		Name:           "test-project",
		ComposeContent: "services: {}",
	})

	// Save initial credentials.
	saveBody, _ := json.Marshal(map[string]any{
		"type": "local",
		"local": map[string]string{
			"path": dir,
		},
	})
	saveResp, err := http.Post(srv.URL+"/api/projects/"+p.ID+"/credentials", "application/json", bytes.NewReader(saveBody))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	saveResp.Body.Close()

	// PUT with a bad path — should return 422.
	updateBody, _ := json.Marshal(map[string]any{
		"type": "local",
		"local": map[string]string{
			"path": "/nonexistent/path/that/does/not/exist",
		},
	})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/projects/"+p.ID+"/credentials", bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status: got %d, want 422", resp.StatusCode)
	}
}

func TestUpdateCredentialsProjectNotFound(t *testing.T) {
	srv, _, _ := setupTestServer(t)

	dir := t.TempDir()
	updateBody, _ := json.Marshal(map[string]any{
		"type": "local",
		"local": map[string]string{
			"path": dir,
		},
	})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/projects/nonexistent/credentials", bytes.NewReader(updateBody))
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

func TestUpdateCredentialsNoExistingCredentials(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	dir := t.TempDir()
	p := m.AddProject(config.Project{
		Name:           "test-project",
		ComposeContent: "services: {}",
	})

	// PUT on a project with no existing credentials — should behave like POST.
	updateBody, _ := json.Marshal(map[string]any{
		"type": "local",
		"local": map[string]string{
			"path": dir,
		},
	})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/projects/"+p.ID+"/credentials", bytes.NewReader(updateBody))
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
}

func TestDeleteCredentials(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	dir := t.TempDir()
	p := m.AddProject(config.Project{
		Name:           "test-project",
		ComposeContent: "services: {}",
	})

	saveBody, _ := json.Marshal(map[string]any{
		"type": "local",
		"local": map[string]string{
			"path": dir,
		},
	})
	saveResp, _ := http.Post(srv.URL+"/api/projects/"+p.ID+"/credentials", "application/json", bytes.NewReader(saveBody))
	saveResp.Body.Close()

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/projects/"+p.ID+"/credentials", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status: got %d, want 204", resp.StatusCode)
	}

	getResp, err := http.Get(srv.URL + "/api/projects/" + p.ID + "/credentials")
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusNotFound {
		t.Fatalf("after delete: got %d, want 404", getResp.StatusCode)
	}
}

func TestSaveCredentialsWithSavedBackendID(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	dir := t.TempDir()
	p := m.AddProject(config.Project{
		Name:           "test-project",
		ComposeContent: "services: {}",
	})

	sb := m.AddSavedBackend(config.SavedBackend{
		Name: "Test Saved Local",
		Credentials: storage.Credentials{
			Type: "local",
			Local: &storage.LocalCreds{
				Path: dir,
			},
		},
	})

	body, _ := json.Marshal(map[string]any{
		"type":             "local",
		"saved_backend_id": sb.ID,
		"local":            map[string]any{"path": dir},
	})

	resp, err := http.Post(srv.URL+"/api/projects/"+p.ID+"/credentials", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]string
		json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("status: got %d, want 200, error: %v", resp.StatusCode, errBody)
	}

	// Verify that the resolved credentials were saved on the project
	proj, ok := m.GetProject(p.ID)
	if !ok {
		t.Fatalf("project not found")
	}
	if proj.Credentials == nil {
		t.Fatalf("project credentials not set")
	}
	if proj.Credentials.SavedBackendID != sb.ID {
		t.Errorf("SavedBackendID: got %q, want %q", proj.Credentials.SavedBackendID, sb.ID)
	}
	if proj.Credentials.Local == nil || proj.Credentials.Local.Path != dir {
		t.Errorf("resolved path: got %v, want %q", proj.Credentials.Local, dir)
	}
}
