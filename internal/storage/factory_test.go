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

func TestNewBackendS3(t *testing.T) {
	creds := &storage.Credentials{
		Type: storage.BackendS3,
		S3: &storage.S3Creds{
			Bucket:    "test-bucket",
			AccessKey: "key",
			SecretKey: "secret",
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

func TestNewBackendWebDAV(t *testing.T) {
	creds := &storage.Credentials{
		Type: storage.BackendWebDAV,
		WebDAV: &storage.WebDAVCreds{
			URL:      "http://webdav.local",
			Username: "user",
			Password: "pwd",
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

func TestNewBackendAzure(t *testing.T) {
	creds := &storage.Credentials{
		Type: storage.BackendAzure,
		Azure: &storage.AzureCreds{
			ConnectionString: "DefaultEndpointsProtocol=https;AccountName=test;AccountKey=key;BlobEndpoint=https://test.blob.core.windows.net/;",
			Container:        "container",
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

func TestNewBackendDropbox(t *testing.T) {
	creds := &storage.Credentials{
		Type: storage.BackendDropbox,
		Dropbox: &storage.DropboxCreds{
			AccessToken: "token",
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

func TestNewBackendGDrive(t *testing.T) {
	dummyCreds := `{
		"type": "service_account",
		"project_id": "test",
		"private_key_id": "123",
		"private_key": "-----BEGIN PRIVATE KEY-----\nMIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQC3\n-----END PRIVATE KEY-----\n",
		"client_email": "test@test.iam.gserviceaccount.com",
		"client_id": "123",
		"auth_uri": "https://accounts.google.com/o/oauth2/auth",
		"token_uri": "https://oauth2.googleapis.com/token",
		"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
		"client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/test"
	}`
	creds := &storage.Credentials{
		Type: storage.BackendGDrive,
		GDrive: &storage.GDriveCreds{
			FolderID:    "folder",
			Credentials: dummyCreds,
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

func TestNewBackendSFTP(t *testing.T) {
	creds := &storage.Credentials{
		Type: storage.BackendSFTP,
		SFTP: &storage.SFTPCreds{
			Host:     "127.0.0.1",
			User:     "user",
			Password: "pwd",
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

