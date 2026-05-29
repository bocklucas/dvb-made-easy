package storage_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bocklucas/dvb-made-easy/internal/storage"
)

func createFixtureFiles(t *testing.T, dir string, names []string) {
	t.Helper()
	for _, name := range names {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("content-"+name), 0644); err != nil {
			t.Fatalf("write fixture %s: %v", name, err)
		}
	}
}

func TestLocalListBackups(t *testing.T) {
	dir := t.TempDir()
	createFixtureFiles(t, dir, []string{
		"backup-2024-01-15T10-30-00.tar.gz",
		"backup-2024-01-16T08-00-00.tar.gz",
		"backup-2024-01-14T12-00-00.tar.gz.gpg",
		"not-a-backup.txt",
		"readme.md",
	})

	backend, err := storage.NewLocalBackend(&storage.LocalCreds{Path: dir})
	if err != nil {
		t.Fatalf("new local backend: %v", err)
	}

	backups, err := backend.ListBackups(context.Background(), nil)
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if len(backups) != 3 {
		t.Fatalf("backups: got %d, want 3", len(backups))
	}

	if backups[0].Key != "backup-2024-01-16T08-00-00.tar.gz" {
		t.Fatalf("first backup: got %q, want newest", backups[0].Key)
	}

	if !backups[2].IsEncrypted {
		t.Fatal("gpg backup must have IsEncrypted=true")
	}
}

func TestLocalListBackupsWithPattern(t *testing.T) {
	dir := t.TempDir()
	createFixtureFiles(t, dir, []string{
		"backup-data-2024-01-15T10-30-00.tar.gz",
		"backup-config-2024-01-15T10-30-00.tar.gz",
		"backup-data-2024-01-16T08-00-00.tar.gz",
	})

	backend, err := storage.NewLocalBackend(&storage.LocalCreds{Path: dir})
	if err != nil {
		t.Fatalf("new local backend: %v", err)
	}

	pattern, err := storage.CompileBackupPattern("backup-data-%Y-%m-%dT%H-%M-%S.{{ .Extension }}")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	backups, err := backend.ListBackups(context.Background(), pattern)
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if len(backups) != 2 {
		t.Fatalf("backups: got %d, want 2 (only data backups)", len(backups))
	}

	for _, b := range backups {
		if b.Key == "backup-config-2024-01-15T10-30-00.tar.gz" {
			t.Fatal("config backup should not be included with data pattern")
		}
	}
}

func TestLocalListBackupsEmptyDir(t *testing.T) {
	dir := t.TempDir()

	backend, err := storage.NewLocalBackend(&storage.LocalCreds{Path: dir})
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	backups, err := backend.ListBackups(context.Background(), nil)
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if len(backups) != 0 {
		t.Fatalf("backups: got %d, want 0", len(backups))
	}
}

func TestLocalListBackupsMissingDir(t *testing.T) {
	backend, err := storage.NewLocalBackend(&storage.LocalCreds{Path: "/nonexistent/path"})
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	_, err = backend.ListBackups(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing directory")
	}
}

func TestLocalDownload(t *testing.T) {
	dir := t.TempDir()
	content := "backup-file-content-here"
	os.WriteFile(filepath.Join(dir, "backup-2024-01-15T10-30-00.tar.gz"), []byte(content), 0644)

	backend, err := storage.NewLocalBackend(&storage.LocalCreds{Path: dir})
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	var buf bytes.Buffer
	err = backend.Download(context.Background(), "backup-2024-01-15T10-30-00.tar.gz", &buf)
	if err != nil {
		t.Fatalf("download: %v", err)
	}

	if buf.String() != content {
		t.Fatalf("content: got %q, want %q", buf.String(), content)
	}
}

func TestLocalDownloadNonexistent(t *testing.T) {
	dir := t.TempDir()

	backend, err := storage.NewLocalBackend(&storage.LocalCreds{Path: dir})
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	var buf bytes.Buffer
	err = backend.Download(context.Background(), "nonexistent.tar.gz", &buf)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestLocalTestConnection(t *testing.T) {
	dir := t.TempDir()

	backend, err := storage.NewLocalBackend(&storage.LocalCreds{Path: dir})
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	if err := backend.TestConnection(context.Background()); err != nil {
		t.Fatalf("test connection: %v", err)
	}
}

func TestLocalTestConnectionMissingDir(t *testing.T) {
	backend, err := storage.NewLocalBackend(&storage.LocalCreds{Path: "/nonexistent/path"})
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	if err := backend.TestConnection(context.Background()); err == nil {
		t.Fatal("expected error for missing directory")
	}
}
