package compose_test

import (
	"testing"

	"github.com/offen/restore-manager/internal/compose"
)

const composeWithBackupJob = `
services:
  postgres:
    image: postgres:16
    volumes:
      - pgdata:/var/lib/postgresql/data

  backup:
    image: offen/docker-volume-backup:v2.43.3
    environment:
      BACKUP_FILENAME: "backup-pgdata-%Y-%m-%dT%H-%M-%S.tar.gz"
    volumes:
      - pgdata:/backup/data:ro
      - /mnt/backups:/archive

volumes:
  pgdata:
`

func TestInferVolumesWithBackupJob(t *testing.T) {
	result, err := compose.Infer(composeWithBackupJob)
	if err != nil {
		t.Fatalf("infer: %v", err)
	}

	if len(result.Volumes) != 1 {
		t.Fatalf("volumes: got %d, want 1", len(result.Volumes))
	}

	vol := result.Volumes[0]
	if vol.Name != "pgdata" {
		t.Errorf("name: got %q, want %q", vol.Name, "pgdata")
	}
	if vol.TargetVolumeName != "pgdata" {
		t.Errorf("target_volume_name: got %q, want %q", vol.TargetVolumeName, "pgdata")
	}
	if vol.BackupPattern != "backup-pgdata-%Y-%m-%dT%H-%M-%S.tar.gz" {
		t.Errorf("backup_pattern: got %q", vol.BackupPattern)
	}
	if vol.IsEncrypted {
		t.Error("is_encrypted: got true, want false")
	}
	if vol.ComposeService != "postgres" {
		t.Errorf("compose_service: got %q, want %q", vol.ComposeService, "postgres")
	}
	if vol.ComposeMountPath != "/var/lib/postgresql/data" {
		t.Errorf("compose_mount_path: got %q", vol.ComposeMountPath)
	}
}

func TestInferLocalStorageFromBackupArchiveEnv(t *testing.T) {
	input := `
services:
  app:
    image: myapp
    volumes:
      - appdata:/data

  backup:
    image: offen/docker-volume-backup:v2.43.3
    environment:
      BACKUP_FILENAME: "backup-%Y-%m-%dT%H-%M-%S.tar.gz"
      BACKUP_ARCHIVE: "/mnt/nas/backups"
    volumes:
      - appdata:/backup/data:ro

volumes:
  appdata:
`

	result, err := compose.Infer(input)
	if err != nil {
		t.Fatalf("infer: %v", err)
	}

	if result.Storage == nil {
		t.Fatal("storage: got nil, want suggestion")
	}
	if result.Storage.Type != "local" {
		t.Errorf("storage type: got %q, want %q", result.Storage.Type, "local")
	}
	if result.Storage.LocalPath != "/mnt/nas/backups" {
		t.Errorf("local_path: got %q, want %q", result.Storage.LocalPath, "/mnt/nas/backups")
	}
	if result.Storage.Confidence != "high" {
		t.Errorf("confidence: got %q, want %q", result.Storage.Confidence, "high")
	}
}

func TestInferNoBackupJobStillListsVolumes(t *testing.T) {
	input := `
services:
  redis:
    image: redis:7
    volumes:
      - redis-data:/data

volumes:
  redis-data:
`

	result, err := compose.Infer(input)
	if err != nil {
		t.Fatalf("infer: %v", err)
	}

	if len(result.Volumes) != 1 {
		t.Fatalf("volumes: got %d, want 1", len(result.Volumes))
	}

	vol := result.Volumes[0]
	if vol.Name != "redis-data" {
		t.Errorf("name: got %q, want %q", vol.Name, "redis-data")
	}
	if vol.BackupPattern != "" {
		t.Errorf("backup_pattern: got %q, want empty", vol.BackupPattern)
	}
	// Target name should still default to the volume name
	if vol.TargetVolumeName != "redis-data" {
		t.Errorf("target_volume_name: got %q, want %q", vol.TargetVolumeName, "redis-data")
	}
	if result.Storage != nil {
		t.Errorf("storage: got %+v, want nil", result.Storage)
	}
}

func TestInferDetectsEncryptedPattern(t *testing.T) {
	input := `
services:
  db:
    image: postgres:16
    volumes:
      - dbdata:/var/lib/postgresql/data

  backup:
    image: offen/docker-volume-backup:v2.43.3
    environment:
      BACKUP_FILENAME: "backup-db-%Y-%m-%dT%H-%M-%S.tar.gz.gpg"
    volumes:
      - dbdata:/backup/db:ro

volumes:
  dbdata:
`

	result, err := compose.Infer(input)
	if err != nil {
		t.Fatalf("infer: %v", err)
	}

	if len(result.Volumes) != 1 {
		t.Fatalf("volumes: got %d, want 1", len(result.Volumes))
	}

	if !result.Volumes[0].IsEncrypted {
		t.Error("is_encrypted: got false, want true for .gpg suffix")
	}
}

func TestInferSMBStorageFromEnvVars(t *testing.T) {
	input := `
services:
  app:
    image: myapp
    volumes:
      - appdata:/data

  backup:
    image: offen/docker-volume-backup:v2.43.3
    environment:
      BACKUP_FILENAME: "backup-%Y-%m-%dT%H-%M-%S.tar.gz"
      SMB_HOST: "192.168.1.50"
      SMB_SHARE: "backups"
      SMB_USERNAME: "admin"
    volumes:
      - appdata:/backup/data:ro

volumes:
  appdata:
`

	result, err := compose.Infer(input)
	if err != nil {
		t.Fatalf("infer: %v", err)
	}

	if result.Storage == nil {
		t.Fatal("storage: got nil, want suggestion")
	}
	if result.Storage.Type != "smb" {
		t.Errorf("storage type: got %q, want %q", result.Storage.Type, "smb")
	}
	if result.Storage.SMBHost != "192.168.1.50" {
		t.Errorf("smb_host: got %q, want %q", result.Storage.SMBHost, "192.168.1.50")
	}
	if result.Storage.SMBShare != "backups" {
		t.Errorf("smb_share: got %q, want %q", result.Storage.SMBShare, "backups")
	}
	if result.Storage.SMBUser != "admin" {
		t.Errorf("smb_username: got %q, want %q", result.Storage.SMBUser, "admin")
	}
	if result.Storage.Confidence != "high" {
		t.Errorf("confidence: got %q, want %q", result.Storage.Confidence, "high")
	}
}

func TestInferLocalStorageFromBindMount(t *testing.T) {
	input := `
services:
  app:
    image: myapp
    volumes:
      - appdata:/data

  backup:
    image: offen/docker-volume-backup:v2.43.3
    environment:
      BACKUP_FILENAME: "backup-%Y-%m-%dT%H-%M-%S.tar.gz"
    volumes:
      - appdata:/backup/data:ro
      - /mnt/backups:/backup/archive

volumes:
  appdata:
`

	result, err := compose.Infer(input)
	if err != nil {
		t.Fatalf("infer: %v", err)
	}

	if result.Storage == nil {
		t.Fatal("storage: got nil, want suggestion")
	}
	if result.Storage.Type != "local" {
		t.Errorf("storage type: got %q, want %q", result.Storage.Type, "local")
	}
	if result.Storage.LocalPath != "/mnt/backups" {
		t.Errorf("local_path: got %q, want %q", result.Storage.LocalPath, "/mnt/backups")
	}
}

func TestInferSMBStorageFromCIFSVolume(t *testing.T) {
	input := `
services:
  app:
    image: myapp
    volumes:
      - appdata:/data

  backup:
    image: offen/docker-volume-backup:v2.43.3
    environment:
      BACKUP_FILENAME: "backup-%Y-%m-%dT%H-%M-%S.tar.gz"
    volumes:
      - appdata:/backup/data:ro
      - smb_backup:/archive

volumes:
  appdata:
  smb_backup:
    driver: local
    driver_opts:
      type: cifs
      device: "//192.168.1.100/docker-backup/gitea"
      o: "username=backup_user,password=secret,vers=3.0"
`

	result, err := compose.Infer(input)
	if err != nil {
		t.Fatalf("infer: %v", err)
	}

	if result.Storage == nil {
		t.Fatal("storage: got nil, want suggestion")
	}
	if result.Storage.Type != "smb" {
		t.Errorf("storage type: got %q, want %q", result.Storage.Type, "smb")
	}
	if result.Storage.SMBHost != "192.168.1.100" {
		t.Errorf("smb_host: got %q, want %q", result.Storage.SMBHost, "192.168.1.100")
	}
	if result.Storage.SMBShare != "docker-backup" {
		t.Errorf("smb_share: got %q, want %q", result.Storage.SMBShare, "docker-backup")
	}
	if result.Storage.SMBUser != "backup_user" {
		t.Errorf("smb_username: got %q, want %q", result.Storage.SMBUser, "backup_user")
	}
	if result.Storage.Confidence != "high" {
		t.Errorf("confidence: got %q, want %q", result.Storage.Confidence, "high")
	}
}
