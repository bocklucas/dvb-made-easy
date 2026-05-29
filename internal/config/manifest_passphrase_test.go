package config_test

import (
	"testing"

	"github.com/bocklucas/dvb-made-easy/internal/config"
	"github.com/bocklucas/dvb-made-easy/internal/encrypt"
)

func TestSetVolumePassphrase(t *testing.T) {
	m := config.NewManifest()
	p := m.AddProject(config.Project{
		Name:           "test-project",
		ComposeContent: "services: {}",
		Volumes: []config.Volume{
			{Name: "pgdata", ComposeService: "db", ComposeMountPath: "/data"},
		},
	})

	if !m.SetVolumePassphrase(p.ID, "pgdata", "secret123") {
		t.Fatal("SetVolumePassphrase must return true for existing project and volume")
	}

	proj, ok := m.GetProject(p.ID)
	if !ok {
		t.Fatal("project not found")
	}
	if proj.Volumes[0].Passphrase != "secret123" {
		t.Fatalf("passphrase: got %q, want %q", proj.Volumes[0].Passphrase, "secret123")
	}
}

func TestSetVolumePassphraseProjectNotFound(t *testing.T) {
	m := config.NewManifest()
	if m.SetVolumePassphrase("nonexistent", "pgdata", "secret") {
		t.Fatal("SetVolumePassphrase must return false for nonexistent project")
	}
}

func TestSetVolumePassphraseVolumeNotFound(t *testing.T) {
	m := config.NewManifest()
	p := m.AddProject(config.Project{
		Name:           "test-project",
		ComposeContent: "services: {}",
	})

	if m.SetVolumePassphrase(p.ID, "nonexistent-volume", "secret") {
		t.Fatal("SetVolumePassphrase must return false for nonexistent volume")
	}
}

func TestClearVolumePassphrase(t *testing.T) {
	m := config.NewManifest()
	p := m.AddProject(config.Project{
		Name:           "test-project",
		ComposeContent: "services: {}",
		Volumes: []config.Volume{
			{Name: "pgdata", ComposeService: "db", ComposeMountPath: "/data"},
		},
	})

	m.SetVolumePassphrase(p.ID, "pgdata", "secret123")

	if !m.ClearVolumePassphrase(p.ID, "pgdata") {
		t.Fatal("ClearVolumePassphrase must return true for existing project and volume")
	}

	proj, _ := m.GetProject(p.ID)
	if proj.Volumes[0].Passphrase != "" {
		t.Fatal("passphrase must be empty after clear")
	}
}

func TestClearVolumePassphraseNotFound(t *testing.T) {
	m := config.NewManifest()
	if m.ClearVolumePassphrase("nonexistent", "pgdata") {
		t.Fatal("ClearVolumePassphrase must return false for nonexistent project")
	}

	p := m.AddProject(config.Project{
		Name:           "test-project",
		ComposeContent: "services: {}",
	})
	if m.ClearVolumePassphrase(p.ID, "nonexistent-volume") {
		t.Fatal("ClearVolumePassphrase must return false for nonexistent volume")
	}
}

func TestSetProjectDefaultPassphrase(t *testing.T) {
	m := config.NewManifest()
	p := m.AddProject(config.Project{
		Name:           "test-project",
		ComposeContent: "services: {}",
	})

	if !m.SetProjectDefaultPassphrase(p.ID, "default-secret") {
		t.Fatal("SetProjectDefaultPassphrase must return true for existing project")
	}

	proj, _ := m.GetProject(p.ID)
	if proj.DefaultPassphrase != "default-secret" {
		t.Fatalf("default passphrase: got %q, want %q", proj.DefaultPassphrase, "default-secret")
	}
}

func TestSetProjectDefaultPassphraseNotFound(t *testing.T) {
	m := config.NewManifest()
	if m.SetProjectDefaultPassphrase("nonexistent", "secret") {
		t.Fatal("SetProjectDefaultPassphrase must return false for nonexistent project")
	}
}

func TestClearProjectDefaultPassphrase(t *testing.T) {
	m := config.NewManifest()
	p := m.AddProject(config.Project{
		Name:           "test-project",
		ComposeContent: "services: {}",
	})

	m.SetProjectDefaultPassphrase(p.ID, "default-secret")

	if !m.ClearProjectDefaultPassphrase(p.ID) {
		t.Fatal("ClearProjectDefaultPassphrase must return true for existing project")
	}

	proj, _ := m.GetProject(p.ID)
	if proj.DefaultPassphrase != "" {
		t.Fatal("default passphrase must be empty after clear")
	}
}

func TestClearProjectDefaultPassphraseNotFound(t *testing.T) {
	m := config.NewManifest()
	if m.ClearProjectDefaultPassphrase("nonexistent") {
		t.Fatal("ClearProjectDefaultPassphrase must return false for nonexistent project")
	}
}

func TestPassphrasePersistenceRoundTrip(t *testing.T) {
	dir := t.TempDir()
	key, err := encrypt.LoadOrCreateKey(dir)
	if err != nil {
		t.Fatalf("key: %v", err)
	}

	m := config.NewManifest()
	p := m.AddProject(config.Project{
		Name:           "test-project",
		ComposeContent: "services: {}",
		Volumes: []config.Volume{
			{Name: "pgdata", ComposeService: "db", ComposeMountPath: "/data"},
		},
	})

	m.SetVolumePassphrase(p.ID, "pgdata", "volume-secret")
	m.SetProjectDefaultPassphrase(p.ID, "default-secret")

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

	if proj.DefaultPassphrase != "default-secret" {
		t.Fatalf("default passphrase: got %q, want %q", proj.DefaultPassphrase, "default-secret")
	}

	if len(proj.Volumes) == 0 {
		t.Fatal("volumes must survive persistence")
	}
	if proj.Volumes[0].Passphrase != "volume-secret" {
		t.Fatalf("volume passphrase: got %q, want %q", proj.Volumes[0].Passphrase, "volume-secret")
	}
}

func TestVolumePriorityOverDefault(t *testing.T) {
	m := config.NewManifest()
	p := m.AddProject(config.Project{
		Name:           "test-project",
		ComposeContent: "services: {}",
		Volumes: []config.Volume{
			{Name: "pgdata", ComposeService: "db", ComposeMountPath: "/data"},
		},
	})

	m.SetProjectDefaultPassphrase(p.ID, "default-secret")
	m.SetVolumePassphrase(p.ID, "pgdata", "volume-secret")

	proj, _ := m.GetProject(p.ID)
	if proj.DefaultPassphrase != "default-secret" {
		t.Fatalf("default passphrase: got %q, want %q", proj.DefaultPassphrase, "default-secret")
	}
	if proj.Volumes[0].Passphrase != "volume-secret" {
		t.Fatalf("volume passphrase: got %q, want %q", proj.Volumes[0].Passphrase, "volume-secret")
	}
	// Both are stored; the API layer chooses volume-specific over default.
}
