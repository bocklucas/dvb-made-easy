package storage_test

import (
	"testing"

	"github.com/offen/restore-manager/internal/storage"
)

func TestNewSMBBackendDefaultPort(t *testing.T) {
	backend, err := storage.NewSMBBackend(&storage.SMBCreds{
		Host:     "nas.local",
		Share:    "backups",
		Path:     "offen",
		Username: "admin",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	if backend == nil {
		t.Fatal("backend must not be nil")
	}
}

func TestNewSMBBackendCustomPort(t *testing.T) {
	backend, err := storage.NewSMBBackend(&storage.SMBCreds{
		Host:     "nas.local",
		Share:    "backups",
		Path:     "offen",
		Username: "admin",
		Password: "secret",
		Port:     4455,
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	if backend == nil {
		t.Fatal("backend must not be nil")
	}
}

func TestNewSMBBackendMissingHost(t *testing.T) {
	_, err := storage.NewSMBBackend(&storage.SMBCreds{
		Share:    "backups",
		Username: "admin",
		Password: "secret",
	})
	if err == nil {
		t.Fatal("expected error for missing host")
	}
}

func TestNewSMBBackendMissingShare(t *testing.T) {
	_, err := storage.NewSMBBackend(&storage.SMBCreds{
		Host:     "nas.local",
		Username: "admin",
		Password: "secret",
	})
	if err == nil {
		t.Fatal("expected error for missing share")
	}
}

func TestNewSMBBackendNilCreds(t *testing.T) {
	_, err := storage.NewSMBBackend(nil)
	if err == nil {
		t.Fatal("expected error for nil creds")
	}
}

func TestSMBBackendImplementsInterface(t *testing.T) {
	backend, _ := storage.NewSMBBackend(&storage.SMBCreds{
		Host:     "nas.local",
		Share:    "backups",
		Username: "admin",
		Password: "secret",
	})

	var _ storage.Backend = backend
}
