package storage_test

import (
	"testing"

	"github.com/offen/restore-manager/internal/storage"
)

func TestNewBackendLocal(t *testing.T) {
	dir := t.TempDir()
	creds := &storage.Credentials{
		Type:  storage.BackendLocal,
		Local: &storage.LocalCreds{Path: dir},
	}

	backend, err := storage.NewBackend(creds)
	if err != nil {
		t.Fatalf("new backend: %v", err)
	}

	if backend == nil {
		t.Fatal("backend must not be nil")
	}
}

func TestNewBackendSMB(t *testing.T) {
	creds := &storage.Credentials{
		Type: storage.BackendSMB,
		SMB: &storage.SMBCreds{
			Host:     "nas.local",
			Share:    "backups",
			Username: "admin",
			Password: "secret",
		},
	}

	backend, err := storage.NewBackend(creds)
	if err != nil {
		t.Fatalf("new backend: %v", err)
	}

	if backend == nil {
		t.Fatal("backend must not be nil")
	}
}

func TestNewBackendUnknownType(t *testing.T) {
	creds := &storage.Credentials{
		Type: "s3",
	}

	_, err := storage.NewBackend(creds)
	if err == nil {
		t.Fatal("expected error for unknown backend type")
	}
}

func TestNewBackendNilCreds(t *testing.T) {
	_, err := storage.NewBackend(nil)
	if err == nil {
		t.Fatal("expected error for nil credentials")
	}
}

func TestNewBackendLocalMissingCreds(t *testing.T) {
	creds := &storage.Credentials{
		Type: storage.BackendLocal,
	}

	_, err := storage.NewBackend(creds)
	if err == nil {
		t.Fatal("expected error for missing local credentials")
	}
}

func TestNewBackendSMBMissingCreds(t *testing.T) {
	creds := &storage.Credentials{
		Type: storage.BackendSMB,
	}

	_, err := storage.NewBackend(creds)
	if err == nil {
		t.Fatal("expected error for missing SMB credentials")
	}
}
