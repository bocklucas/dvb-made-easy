package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

func TestDropboxBackend(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == "POST" && r.URL.Path == "/2/users/get_current_account" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
			return
		}

		if r.Method == "POST" && r.URL.Path == "/2/files/list_folder" {
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			if body["path"] != "/backups" {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`"invalid path"`))
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"entries": [
					{
						".tag": "file",
						"name": "backup-db-2026-05-25T12-00-00.tar.gz",
						"path_display": "/backups/backup-db-2026-05-25T12-00-00.tar.gz",
						"size": 12345,
						"client_modified": "2026-05-25T12:00:00Z"
					}
				],
				"cursor": "cursor_page_1",
				"has_more": true
			}`))
			return
		}

		if r.Method == "POST" && r.URL.Path == "/2/files/list_folder/continue" {
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			if body["cursor"] != "cursor_page_1" {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`"invalid cursor"`))
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"entries": [
					{
						".tag": "file",
						"name": "other.txt",
						"path_display": "/backups/other.txt",
						"size": 500,
						"client_modified": "2026-05-25T12:00:00Z"
					}
				],
				"cursor": "cursor_page_2",
				"has_more": false
			}`))
			return
		}

		if r.Method == "POST" && r.URL.Path == "/2/files/download" {
			argHeader := r.Header.Get("Dropbox-API-Arg")
			var arg map[string]string
			json.Unmarshal([]byte(argHeader), &arg)
			if arg["path"] != "/backups/backup-db-2026-05-25T12-00-00.tar.gz" {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`"invalid download path"`))
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("mock dropbox payload"))
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	backend, err := NewDropboxBackend(&DropboxCreds{
		AccessToken: "token",
		RemotePath:  "/backups",
	})
	if err != nil {
		t.Fatalf("failed to create backend: %v", err)
	}

	backend.apiURL = ts.URL
	backend.contentURL = ts.URL

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
	if backups[0].Key != "/backups/backup-db-2026-05-25T12-00-00.tar.gz" {
		t.Errorf("expected key /backups/backup-db-2026-05-25T12-00-00.tar.gz, got %s", backups[0].Key)
	}
	if backups[0].Size != 12345 {
		t.Errorf("expected size 12345, got %d", backups[0].Size)
	}

	// Test download
	var buf bytes.Buffer
	err = backend.Download(context.Background(), "/backups/backup-db-2026-05-25T12-00-00.tar.gz", &buf)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}
	if buf.String() != "mock dropbox payload" {
		t.Errorf("expected payload 'mock dropbox payload', got %q", buf.String())
	}
}
