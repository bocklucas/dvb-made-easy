package storage

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type mockGDriveTransport struct {
	serverURL string
}

func (t *mockGDriveTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	target, _ := url.Parse(t.serverURL)
	req.URL.Scheme = target.Scheme
	req.URL.Host = target.Host
	return http.DefaultTransport.RoundTrip(req)
}

func TestGDriveBackend(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// List / query files
		if r.Method == "GET" && r.URL.Path == "/drive/v3/files" {
			q := r.URL.Query().Get("q")
			// Download queries specific file by name
			if q == "name = 'backup-db-2026-05-25T12-00-00.tar.gz' and 'test-folder-id' in parents and trashed = false" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"files": [{"id": "file-id-123"}]}`))
				return
			}

			// Listing files query
			if q == "'test-folder-id' in parents and trashed = false" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{
					"files": [
						{
							"id": "file-id-123",
							"name": "backup-db-2026-05-25T12-00-00.tar.gz",
							"size": 12345,
							"modifiedTime": "2026-05-25T12:00:00Z"
						},
						{
							"id": "file-id-456",
							"name": "other.txt",
							"size": 500,
							"modifiedTime": "2026-05-25T12:00:00Z"
						}
					]
				}`))
				return
			}
		}

		// Download file content (alt=media)
		if r.Method == "GET" && r.URL.Path == "/drive/v3/files/file-id-123" && r.URL.Query().Get("alt") == "media" {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("mock gdrive payload"))
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	client := &http.Client{Transport: &mockGDriveTransport{serverURL: ts.URL}}
	srv, err := drive.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		t.Fatalf("failed to create drive service: %v", err)
	}

	backend := &GDriveBackend{
		srv:      srv,
		folderID: "test-folder-id",
	}

	// Test connection
	if err := backend.TestConnection(context.Background()); err != nil {
		t.Errorf("TestConnection failed: %v", err)
	}

	// Test list backups
	pattern := regexp.MustCompile(`^backup-db-.*\.tar\.gz$`)
	backups, err := backend.ListBackups(context.Background(), pattern)
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}

	if len(backups) != 1 {
		t.Fatalf("expected 1 backup, got %d", len(backups))
	}
	if backups[0].Key != "backup-db-2026-05-25T12-00-00.tar.gz" {
		t.Errorf("expected key backup-db-2026-05-25T12-00-00.tar.gz, got %s", backups[0].Key)
	}
	if backups[0].Size != 12345 {
		t.Errorf("expected size 12345, got %d", backups[0].Size)
	}

	// Test download
	var buf bytes.Buffer
	err = backend.Download(context.Background(), "backup-db-2026-05-25T12-00-00.tar.gz", &buf)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}
	if buf.String() != "mock gdrive payload" {
		t.Errorf("expected payload 'mock gdrive payload', got %q", buf.String())
	}
}
