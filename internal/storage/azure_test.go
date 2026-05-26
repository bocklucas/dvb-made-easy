package storage_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/offen/restore-manager/internal/storage"
)

func TestAzureBackend(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// List request (comp=list)
		if r.Method == "GET" && r.URL.Query().Get("comp") == "list" {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			xml := `<?xml version="1.0" encoding="utf-8"?>
<EnumerationResults ContainerName="test-container">
  <Blobs>
    <Blob>
      <Name>backup-db-2026-05-25T12-00-00.tar.gz</Name>
      <Properties>
        <Content-Length>12345</Content-Length>
        <Last-Modified>Mon, 25 May 2026 12:00:00 GMT</Last-Modified>
      </Properties>
    </Blob>
    <Blob>
      <Name>other.txt</Name>
      <Properties>
        <Content-Length>500</Content-Length>
        <Last-Modified>Mon, 25 May 2026 12:00:00 GMT</Last-Modified>
      </Properties>
    </Blob>
  </Blobs>
  <NextMarker />
</EnumerationResults>`
			w.Write([]byte(xml))
			return
		}

		// Download request
		if r.Method == "GET" && r.URL.Path == "/test/test-container/backup-db-2026-05-25T12-00-00.tar.gz" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("mock azure payload"))
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	connStr := fmt.Sprintf("DefaultEndpointsProtocol=http;AccountName=test;AccountKey=key;BlobEndpoint=%s/test;", ts.URL)
	backend, err := storage.NewAzureBackend(&storage.AzureCreds{
		ConnectionString: connStr,
		Container:        "test-container",
	})
	if err != nil {
		t.Fatalf("failed to create backend: %v", err)
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
	if buf.String() != "mock azure payload" {
		t.Errorf("expected payload 'mock azure payload', got %q", buf.String())
	}
}
