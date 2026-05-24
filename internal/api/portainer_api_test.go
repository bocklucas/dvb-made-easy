package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/offen/restore-manager/internal/portainer"
)

const testComposeContent = `services:
  db:
    image: postgres
    volumes:
      - pgdata:/data
  backup:
    image: offen/docker-volume-backup:v2
    volumes:
      - pgdata:/backup/pgdata:ro
    environment:
      BACKUP_FILENAME: 'backup-%Y%m%d.tar.gz'
volumes:
  pgdata:
`

func mockPortainerServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/api/system/status", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"Version":"2.19.0"}`))
	})

	mux.HandleFunc("/api/endpoints", func(w http.ResponseWriter, r *http.Request) {
		endpoints := []portainer.Endpoint{
			{ID: 1, Name: "local"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(endpoints)
	})

	mux.HandleFunc("/api/stacks", func(w http.ResponseWriter, r *http.Request) {
		stacks := []portainer.Stack{
			{ID: 1, Name: "mystack", EndpointID: 1, Status: 1},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stacks)
	})

	mux.HandleFunc("/api/stacks/1/file", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{
			"StackFileContent": testComposeContent,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	return httptest.NewServer(mux)
}

func TestPortainerConnect(t *testing.T) {
	mockSrv := mockPortainerServer(t)
	defer mockSrv.Close()

	appSrv, _, _ := setupTestServer(t)

	body, _ := json.Marshal(map[string]string{
		"url":     mockSrv.URL,
		"api_key": "test-key",
	})

	resp, err := http.Post(appSrv.URL+"/api/portainer/connect", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}

	var result struct {
		Endpoints []struct {
			ID   int    `json:"Id"`
			Name string `json:"Name"`
		} `json:"endpoints"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(result.Endpoints) != 1 {
		t.Fatalf("endpoints: got %d, want 1", len(result.Endpoints))
	}
	if result.Endpoints[0].Name != "local" {
		t.Fatalf("endpoint name: got %q, want %q", result.Endpoints[0].Name, "local")
	}
}

func TestPortainerConnectMissingFields(t *testing.T) {
	appSrv, _, _ := setupTestServer(t)

	body, _ := json.Marshal(map[string]string{
		"url": "http://portainer.example.com",
		// missing api_key
	})

	resp, err := http.Post(appSrv.URL+"/api/portainer/connect", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", resp.StatusCode)
	}
}

func TestPortainerStacks(t *testing.T) {
	mockSrv := mockPortainerServer(t)
	defer mockSrv.Close()

	appSrv, _, _ := setupTestServer(t)

	body, _ := json.Marshal(map[string]interface{}{
		"url":         mockSrv.URL,
		"api_key":     "test-key",
		"endpoint_id": 1,
	})

	resp, err := http.Post(appSrv.URL+"/api/portainer/stacks", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}

	var stacks []struct {
		ID           int    `json:"Id"`
		Name         string `json:"Name"`
		IsOfenBacked bool   `json:"is_offen_backed"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&stacks); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(stacks) != 1 {
		t.Fatalf("stacks: got %d, want 1", len(stacks))
	}
	if stacks[0].Name != "mystack" {
		t.Fatalf("stack name: got %q, want %q", stacks[0].Name, "mystack")
	}
	if !stacks[0].IsOfenBacked {
		t.Fatal("stack should be detected as offen-backed")
	}
}

func TestImportPortainer(t *testing.T) {
	mockSrv := mockPortainerServer(t)
	defer mockSrv.Close()

	appSrv, manifest, _ := setupTestServer(t)

	body, _ := json.Marshal(map[string]interface{}{
		"portainer_url": mockSrv.URL,
		"api_key":       "test-key",
		"stack_id":      1,
		"endpoint_id":   1,
		"project_name":  "my-portainer-project",
	})

	resp, err := http.Post(appSrv.URL+"/api/projects/import-portainer", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}

	var result struct {
		ID        string `json:"id"`
		DriftHash string `json:"drift_hash"`
		Volumes   []struct {
			Name string `json:"name"`
		} `json:"volumes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if result.ID == "" {
		t.Fatal("id must be set")
	}
	if result.DriftHash == "" {
		t.Fatal("drift_hash must be set")
	}
	if len(result.Volumes) != 1 {
		t.Fatalf("volumes: got %d, want 1", len(result.Volumes))
	}
	if result.Volumes[0].Name != "pgdata" {
		t.Fatalf("volume name: got %q, want %q", result.Volumes[0].Name, "pgdata")
	}

	// Verify source and portainer_source are stored in manifest.
	project, ok := manifest.GetProject(result.ID)
	if !ok {
		t.Fatal("project not found in manifest")
	}
	if project.Source != "portainer" {
		t.Fatalf("source: got %q, want %q", project.Source, "portainer")
	}
	if project.PortainerSource == nil {
		t.Fatal("portainer_source must be set")
	}
	if project.PortainerSource.StackID != 1 {
		t.Fatalf("stack_id: got %d, want 1", project.PortainerSource.StackID)
	}
	if project.PortainerSource.EndpointID != 1 {
		t.Fatalf("endpoint_id: got %d, want 1", project.PortainerSource.EndpointID)
	}
	if project.Name != "my-portainer-project" {
		t.Fatalf("name: got %q, want %q", project.Name, "my-portainer-project")
	}
}

func TestImportPortainerMissingFields(t *testing.T) {
	appSrv, _, _ := setupTestServer(t)

	tests := []struct {
		name string
		body map[string]interface{}
	}{
		{
			name: "missing portainer_url",
			body: map[string]interface{}{
				"api_key":      "test-key",
				"stack_id":     1,
				"endpoint_id":  1,
				"project_name": "test",
			},
		},
		{
			name: "missing api_key",
			body: map[string]interface{}{
				"portainer_url": "http://portainer.example.com",
				"stack_id":      1,
				"endpoint_id":   1,
				"project_name":  "test",
			},
		},
		{
			name: "missing stack_id",
			body: map[string]interface{}{
				"portainer_url": "http://portainer.example.com",
				"api_key":       "test-key",
				"endpoint_id":   1,
				"project_name":  "test",
			},
		},
		{
			name: "missing endpoint_id",
			body: map[string]interface{}{
				"portainer_url": "http://portainer.example.com",
				"api_key":       "test-key",
				"stack_id":      1,
				"project_name":  "test",
			},
		},
		{
			name: "missing project_name",
			body: map[string]interface{}{
				"portainer_url": "http://portainer.example.com",
				"api_key":       "test-key",
				"stack_id":      1,
				"endpoint_id":   1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, _ := json.Marshal(tt.body)
			resp, err := http.Post(appSrv.URL+"/api/projects/import-portainer", "application/json", bytes.NewReader(data))
			if err != nil {
				t.Fatalf("post: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("status: got %d, want 400", resp.StatusCode)
			}
		})
	}
}
