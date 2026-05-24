package restore_test

import (
	"testing"

	"github.com/offen/restore-manager/internal/docker"
	"github.com/offen/restore-manager/internal/restore"
)

func TestBuildExtractionConfigUnencrypted(t *testing.T) {
	cfg := restore.BuildExtractionConfig("backup-2026-05-15T04-00-00.tar.gz", "offen-restore-staging", "restore123", "target-volume", "")

	if cfg.Image != "alpine:latest" {
		t.Fatalf("image: got %q, want %q", cfg.Image, "alpine:latest")
	}

	if len(cfg.Cmd) != 3 {
		t.Fatalf("cmd len: got %d, want 3", len(cfg.Cmd))
	}
	if cfg.Cmd[0] != "sh" || cfg.Cmd[1] != "-c" {
		t.Fatalf("cmd prefix: got %v, want [sh -c ...]", cfg.Cmd[:2])
	}

	wantCmd := "tar -xzf /staging/restore123/backup-2026-05-15T04-00-00.tar.gz -C /target --strip-components 2"
	if cfg.Cmd[2] != wantCmd {
		t.Fatalf("cmd:\ngot  %q\nwant %q", cfg.Cmd[2], wantCmd)
	}

	if len(cfg.Env) != 0 {
		t.Fatalf("env: got %v, want empty", cfg.Env)
	}

	assertMounts(t, cfg.Mounts, "offen-restore-staging", "target-volume")
}

func TestBuildExtractionConfigEncrypted(t *testing.T) {
	cfg := restore.BuildExtractionConfig("backup-2026-05-15T04-00-00.tar.gz.gpg", "offen-restore-staging", "restore456", "target-volume", "my-secret-pass")

	wantCmd := `apk add --no-cache gnupg && gpg --batch --passphrase "$GPG_PASSPHRASE" -d /staging/restore456/backup-2026-05-15T04-00-00.tar.gz.gpg | tar -xz -C /target --strip-components 2`
	if cfg.Cmd[2] != wantCmd {
		t.Fatalf("cmd:\ngot  %q\nwant %q", cfg.Cmd[2], wantCmd)
	}

	if len(cfg.Env) != 1 || cfg.Env[0] != "GPG_PASSPHRASE=my-secret-pass" {
		t.Fatalf("env: got %v, want [GPG_PASSPHRASE=my-secret-pass]", cfg.Env)
	}
}

func assertMounts(t *testing.T, mounts []docker.Mount, expectStagingVolume, expectTargetVolume string) {
	t.Helper()
	if len(mounts) != 2 {
		t.Fatalf("mounts: got %d, want 2", len(mounts))
	}

	staging := mounts[0]
	if staging.Source != expectStagingVolume || staging.Target != "/staging" || !staging.ReadOnly {
		t.Fatalf("staging mount: got %+v", staging)
	}

	vol := mounts[1]
	if vol.Source != expectTargetVolume || vol.Target != "/target" || vol.ReadOnly {
		t.Fatalf("volume mount: got %+v", vol)
	}
}
