package storage_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/offen/restore-manager/internal/storage"
)

func TestWebDAVBackend(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PROPFIND" {
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
			w.WriteHeader(http.StatusMultiStatus)
			xml := `<?xml version="1.0" encoding="utf-8"?>
<d:multistatus xmlns:d="DAV:">
  <d:response>
    <d:href>/backups/</d:href>
    <d:propstat>
      <d:prop>
        <d:resourcetype><d:collection/></d:resourcetype>
      </d:prop>
      <d:status>HTTP/1.1 200 OK</d:status>
    </d:propstat>
  </d:response>
  <d:response>
    <d:href>/backups/backup-db-2026-05-25T12-00-00.tar.gz</d:href>
    <d:propstat>
      <d:prop>
        <d:resourcetype/>
        <d:getcontentlength>12345</d:getcontentlength>
        <d:getlastmodified>Mon, 25 May 2026 12:00:00 GMT</d:getlastmodified>
      </d:prop>
      <d:status>HTTP/1.1 200 OK</d:status>
    </d:propstat>
  </d:response>
  <d:response>
    <d:href>/backups/other.txt</d:href>
    <d:propstat>
      <d:prop>
        <d:resourcetype/>
        <d:getcontentlength>500</d:getcontentlength>
        <d:getlastmodified>Mon, 25 May 2026 12:00:00 GMT</d:getlastmodified>
      </d:prop>
      <d:status>HTTP/1.1 200 OK</d:status>
    </d:propstat>
  </d:response>
</d:multistatus>`
			w.Write([]byte(xml))
			return
		}

		if r.Method == "GET" && r.URL.Path == "/backups/backup-db-2026-05-25T12-00-00.tar.gz" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("mock webdav payload"))
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	backend, err := storage.NewWebDAVBackend(&storage.WebDAVCreds{
		URL:      ts.URL,
		Username: "user",
		Password: "pass",
		Path:     "/backups",
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
	if backups[0].Key != "backups/backup-db-2026-05-25T12-00-00.tar.gz" {
		t.Errorf("expected key backups/backup-db-2026-05-25T12-00-00.tar.gz, got %s", backups[0].Key)
	}
	if backups[0].Size != 12345 {
		t.Errorf("expected size 12345, got %d", backups[0].Size)
	}

	// Test download
	var buf bytes.Buffer
	err = backend.Download(context.Background(), "backups/backup-db-2026-05-25T12-00-00.tar.gz", &buf)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}
	if buf.String() != "mock webdav payload" {
		t.Errorf("expected payload 'mock webdav payload', got %q", buf.String())
	}
}
