package config_test

import (
	"testing"

	"github.com/offen/restore-manager/internal/config"
	"github.com/offen/restore-manager/internal/encrypt"
	"github.com/offen/restore-manager/internal/storage"
)

func TestSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	key, err := encrypt.LoadOrCreateKey(dir)
	if err != nil {
		t.Fatalf("key: %v", err)
	}

	m := config.NewManifest()
	m.AddProject(config.Project{
		Name:           "test-project",
		ComposeContent: "services:\n  db:\n    image: postgres",
	})

	if err := config.Save(dir, key, m); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := config.Load(dir, key)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	projects := loaded.ListProjects()
	if len(projects) != 1 {
		t.Fatalf("projects: got %d, want 1", len(projects))
	}

	if projects[0].Name != "test-project" {
		t.Fatalf("name: got %q, want %q", projects[0].Name, "test-project")
	}

	if projects[0].ID == "" {
		t.Fatal("project ID must be set")
	}
}

func TestCredentialPersistenceRoundTrip(t *testing.T) {
	dir := t.TempDir()
	key, err := encrypt.LoadOrCreateKey(dir)
	if err != nil {
		t.Fatalf("key: %v", err)
	}

	m := config.NewManifest()
	p := m.AddProject(config.Project{
		Name:           "test-project",
		ComposeContent: "services: {}",
	})

	creds := &storage.Credentials{
		Type: storage.BackendSMB,
		SMB: &storage.SMBCreds{
			Host:     "nas.local",
			Share:    "backups",
			Path:     "offen",
			Username: "admin",
			Password: "secret",
			Port:     445,
		},
	}

	if !m.SetProjectCredentials(p.ID, creds) {
		t.Fatal("set credentials must return true for existing project")
	}

	if err := config.Save(dir, key, m); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := config.Load(dir, key)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	proj, ok := loaded.GetProject(p.ID)
	if !ok {
		t.Fatal("project not found after reload")
	}

	if proj.Credentials == nil {
		t.Fatal("credentials must survive persistence")
	}

	if proj.Credentials.Type != storage.BackendSMB {
		t.Fatalf("type: got %q, want %q", proj.Credentials.Type, storage.BackendSMB)
	}

	if proj.Credentials.SMB.Host != "nas.local" {
		t.Fatalf("host: got %q, want %q", proj.Credentials.SMB.Host, "nas.local")
	}

	if proj.Credentials.SMB.Password != "secret" {
		t.Fatalf("password: got %q, want %q", proj.Credentials.SMB.Password, "secret")
	}
}

func TestSetCredentialsNonexistentProject(t *testing.T) {
	m := config.NewManifest()
	creds := &storage.Credentials{
		Type:  storage.BackendLocal,
		Local: &storage.LocalCreds{Path: "/tmp"},
	}
	if m.SetProjectCredentials("nonexistent-id", creds) {
		t.Fatal("set credentials must return false for nonexistent project")
	}
}

func TestClearProjectCredentials(t *testing.T) {
	m := config.NewManifest()
	p := m.AddProject(config.Project{
		Name:           "test-project",
		ComposeContent: "services: {}",
	})

	m.SetProjectCredentials(p.ID, &storage.Credentials{
		Type:  storage.BackendLocal,
		Local: &storage.LocalCreds{Path: "/tmp"},
	})

	if !m.ClearProjectCredentials(p.ID) {
		t.Fatal("clear credentials must return true for existing project")
	}

	proj, _ := m.GetProject(p.ID)
	if proj.Credentials != nil {
		t.Fatal("credentials must be nil after clear")
	}
}

func TestUpdateProjectCompose(t *testing.T) {
	m := config.NewManifest()
	p := m.AddProject(config.Project{
		Name:           "test",
		ComposeContent: "old content",
		Volumes: []config.Volume{
			{Name: "pgdata", BackupPattern: "pg-*.tar.gz", Passphrase: "secret"},
		},
	})

	newVolumes := []config.Volume{
		{Name: "pgdata", BackupPattern: "pg-*.tar.gz", Passphrase: "secret"},
		{Name: "redis", BackupPattern: ""},
	}

	ok := m.UpdateProjectCompose(p.ID, "new content", newVolumes)
	if !ok {
		t.Fatal("UpdateProjectCompose must return true for existing project")
	}

	updated, _ := m.GetProject(p.ID)
	if updated.ComposeContent != "new content" {
		t.Fatalf("ComposeContent: got %q, want %q", updated.ComposeContent, "new content")
	}
	if updated.ComposeHash == "" {
		t.Fatal("ComposeHash must be set after update")
	}
	if len(updated.Volumes) != 2 {
		t.Fatalf("Volumes: got %d, want 2", len(updated.Volumes))
	}

	// non-existent project returns false
	ok = m.UpdateProjectCompose("no-such-id", "content", nil)
	if ok {
		t.Fatal("UpdateProjectCompose must return false for missing project")
	}
}

func TestLoadReturnsNewManifestWhenNoFileExists(t *testing.T) {
	dir := t.TempDir()
	key := make([]byte, 32)

	m, err := config.Load(dir, key)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if m.InstallationID == "" {
		t.Fatal("installation ID must be set on fresh manifest")
	}

	if len(m.ListProjects()) != 0 {
		t.Fatal("fresh manifest must have no projects")
	}
}
