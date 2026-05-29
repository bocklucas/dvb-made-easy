package storage_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/bocklucas/dvb-made-easy/internal/storage"
)

func TestS3Backend(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "HEAD" && r.URL.Path == "/test-bucket" {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method == "GET" && r.URL.Path == "/test-bucket" && r.URL.Query().Get("list-type") == "2" {
			w.Header().Set("Content-Type", "application/xml")
			xml := `<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
    <Name>test-bucket</Name>
    <IsTruncated>false</IsTruncated>
    <Contents>
        <Key>backup-db-2026-05-25T12-00-00.tar.gz</Key>
        <LastModified>2026-05-25T12:00:00.000Z</LastModified>
        <Size>12345</Size>
    </Contents>
    <Contents>
        <Key>other-file.txt</Key>
        <LastModified>2026-05-25T12:00:00.000Z</LastModified>
        <Size>500</Size>
    </Contents>
</ListBucketResult>`
			w.Write([]byte(xml))
			return
		}

		if r.Method == "GET" && r.URL.Path == "/test-bucket/backup-db-2026-05-25T12-00-00.tar.gz" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("mock s3 payload"))
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	backend, err := storage.NewS3Backend(&storage.S3Creds{
		Bucket:    "test-bucket",
		AccessKey: "key",
		SecretKey: "secret",
		Endpoint:  ts.URL,
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
	if buf.String() != "mock s3 payload" {
		t.Errorf("expected payload 'mock s3 payload', got %q", buf.String())
	}
}
