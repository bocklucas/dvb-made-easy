package compose_test

import (
	"strings"
	"testing"

	"github.com/offen/restore-manager/internal/compose"
)

func TestGenerateBackupService(t *testing.T) {
	input := `services:
  web:
    image: nginx:latest
  db:
    image: postgres:16
    volumes:
      - db_data:/var/lib/postgresql/data

volumes:
  db_data:
`

	opts := compose.GenerateOptions{
		ServiceName:     "my-backup",
		Image:           "offen/docker-volume-backup:v2.43.3",
		CronExpression:  "0 3 * * *",
		FilenameFormat:  "backup-%Y-%m-%d.tar.gz",
		GpgPassphrase:   "supersecret",
		RetentionDays:   14,
		SelectedVolumes: []string{"db_data"},
		StopServices:    []string{"db"},
		EnvVars: map[string]string{
			"AWS_S3_BUCKET_NAME": "my-cool-bucket",
		},
		Volumes: []string{"./host-backups:/archive"},
	}

	output, err := compose.Generate(input, opts)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify backup service was added
	if !strings.Contains(output, "my-backup:") {
		t.Error("expected output to contain 'my-backup:' service definition")
	}

	// Verify image is set
	if !strings.Contains(output, "image: offen/docker-volume-backup:v2.43.3") {
		t.Error("expected output to contain backup image")
	}

	// Verify environment variables are present
	if !strings.Contains(output, "BACKUP_CRON_EXPRESSION: 0 3 * * *") {
		t.Error("expected cron expression env var")
	}
	if !strings.Contains(output, "BACKUP_FILENAME: backup-%Y-%m-%d.tar.gz") {
		t.Error("expected filename format env var")
	}
	if !strings.Contains(output, "GPG_PASSPHRASE: supersecret") {
		t.Error("expected gpg passphrase env var")
	}
	if !strings.Contains(output, "BACKUP_RETENTION_DAYS: \"14\"") {
		t.Error("expected retention days env var")
	}
	if !strings.Contains(output, "AWS_S3_BUCKET_NAME: my-cool-bucket") {
		t.Error("expected S3 bucket env var")
	}

	// Verify volumes are mounted
	if !strings.Contains(output, "db_data:/backup/db_data:ro") {
		t.Error("expected database volume mounted to backup container")
	}
	if !strings.Contains(output, "/var/run/docker.sock:/var/run/docker.sock:ro") {
		t.Error("expected docker socket mounted to backup container")
	}
	if !strings.Contains(output, "./host-backups:/archive") {
		t.Error("expected host backup volume mounted to backup container")
	}

	// Verify database service has stop label
	if !strings.Contains(output, "docker-volume-backup.stop-during-backup: \"true\"") &&
		!strings.Contains(output, "docker-volume-backup.stop-during-backup=true") {
		t.Error("expected db service to have stop label")
	}
}

func TestGenerateOverwritesExistingBackupService(t *testing.T) {
	input := `services:
  web:
    image: nginx:latest
  backup:
    image: offen/docker-volume-backup:v2
    environment:
      BACKUP_CRON_EXPRESSION: "0 0 * * *"
`

	opts := compose.GenerateOptions{
		ServiceName:    "backup",
		Image:          "offen/docker-volume-backup:v2",
		CronExpression: "0 5 * * *",
	}

	output, err := compose.Generate(input, opts)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if strings.Contains(output, "0 0 * * *") {
		t.Error("expected old cron expression to be overwritten")
	}

	if !strings.Contains(output, "0 5 * * *") {
		t.Error("expected new cron expression to be present")
	}
}

func TestGenerateBackupServiceWithSMB(t *testing.T) {
	input := `services:
  web:
    image: nginx:latest
`

	opts := compose.GenerateOptions{
		ServiceName: "backup",
		Image:       "offen/docker-volume-backup:v2",
		SMBConfig: &compose.SMBVolumeConfig{
			Host:     "192.168.1.100",
			Share:    "backup-share",
			Path:     "subpath",
			Username: "smb_user",
			Password: "password123",
		},
		Volumes: []string{"smb_backup:/archive"},
	}

	output, err := compose.Generate(input, opts)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(output, "smb_backup:/archive") {
		t.Error("expected smb_backup mounted to archive")
	}

	if !strings.Contains(output, "smb_backup:") {
		t.Error("expected volumes block to contain smb_backup definition")
	}

	if !strings.Contains(output, "driver: local") {
		t.Error("expected volume driver local")
	}

	if !strings.Contains(output, "type: cifs") {
		t.Error("expected volume driver_opts type cifs")
	}

	if !strings.Contains(output, "device: //192.168.1.100/backup-share/subpath") {
		t.Error("expected correct CIFS device path")
	}

	if !strings.Contains(output, "o: vers=3.0,addr=192.168.1.100,username=smb_user,password=password123") {
		t.Error("expected correct CIFS options list")
	}
}

func TestGenerateBackupServiceWithSMBEnvVars(t *testing.T) {
	input := `services:
  web:
    image: nginx:latest
`

	opts := compose.GenerateOptions{
		ServiceName: "backup",
		Image:       "offen/docker-volume-backup:v2",
		SMBConfig: &compose.SMBVolumeConfig{
			Host:       "192.168.1.100",
			Share:      "backup-share",
			Path:       "subpath",
			Username:   "smb_user",
			Password:   "password123",
			Port:       1445,
			UseEnvVars: true,
		},
		Volumes: []string{"smb_backup:/archive"},
	}

	output, err := compose.Generate(input, opts)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(output, "device: //${SMB_BACKUP_ADDR}/${SMB_BACKUP_SHARE}/${SMB_BACKUP_PATH}") {
		t.Error("expected environment variables in CIFS device path")
	}

	if !strings.Contains(output, "o: vers=3.0,addr=${SMB_BACKUP_ADDR},username=${SMB_BACKUP_USERNAME},password=${SMB_BACKUP_PASSWORD},port=${SMB_BACKUP_PORT}") {
		t.Error("expected environment variables in CIFS options list")
	}
}
