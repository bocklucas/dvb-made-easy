package compose_test

import (
	"testing"

	"github.com/offen/restore-manager/internal/compose"
)

func TestParseExtractsNamedVolumes(t *testing.T) {
	input := `
services:
  postgres:
    image: postgres:16
    volumes:
      - pgdata:/var/lib/postgresql/data
  redis:
    image: redis:7
    volumes:
      - redis-data:/data

volumes:
  pgdata:
  redis-data:
`

	result, err := compose.Parse(input)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(result.Volumes) != 2 {
		t.Fatalf("volumes: got %d, want 2", len(result.Volumes))
	}

	pg := findVolume(result.Volumes, "pgdata")
	if pg == nil {
		t.Fatal("missing volume pgdata")
	}
	if pg.Service != "postgres" {
		t.Fatalf("pgdata service: got %q, want %q", pg.Service, "postgres")
	}
	if pg.MountPath != "/var/lib/postgresql/data" {
		t.Fatalf("pgdata mount: got %q, want %q", pg.MountPath, "/var/lib/postgresql/data")
	}

	rd := findVolume(result.Volumes, "redis-data")
	if rd == nil {
		t.Fatal("missing volume redis-data")
	}
	if rd.Service != "redis" {
		t.Fatalf("redis-data service: got %q, want %q", rd.Service, "redis")
	}
}

func TestParseIgnoresBindMounts(t *testing.T) {
	input := `
services:
  app:
    image: myapp
    volumes:
      - ./config:/app/config
      - appdata:/app/data

volumes:
  appdata:
`

	result, err := compose.Parse(input)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(result.Volumes) != 1 {
		t.Fatalf("volumes: got %d, want 1 (bind mounts excluded)", len(result.Volumes))
	}

	if result.Volumes[0].Name != "appdata" {
		t.Fatalf("volume name: got %q, want %q", result.Volumes[0].Name, "appdata")
	}
}

func TestParseReturnsErrorForInvalidYAML(t *testing.T) {
	input := `not: [valid: yaml: {{`

	_, err := compose.Parse(input)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestParseHandlesLongFormVolumeSyntax(t *testing.T) {
	input := `
services:
  db:
    image: postgres:16
    volumes:
      - type: volume
        source: dbdata
        target: /var/lib/postgresql/data

volumes:
  dbdata:
`

	result, err := compose.Parse(input)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(result.Volumes) != 1 {
		t.Fatalf("volumes: got %d, want 1", len(result.Volumes))
	}

	if result.Volumes[0].Name != "dbdata" {
		t.Fatalf("name: got %q, want %q", result.Volumes[0].Name, "dbdata")
	}

	if result.Volumes[0].MountPath != "/var/lib/postgresql/data" {
		t.Fatalf("mount: got %q, want %q", result.Volumes[0].MountPath, "/var/lib/postgresql/data")
	}
}

func TestParseExtractsDependsOn(t *testing.T) {
	input := `
services:
  app:
    image: myapp
    depends_on:
      - db
      - redis
  db:
    image: postgres:16
    volumes:
      - pgdata:/var/lib/postgresql/data
  redis:
    image: redis:7

volumes:
  pgdata:
`

	result, err := compose.Parse(input)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	deps, ok := result.DependsOn["app"]
	if !ok {
		t.Fatal("missing depends_on for app")
	}

	if len(deps) != 2 {
		t.Fatalf("app deps: got %d, want 2", len(deps))
	}
}

func TestParseExtractsBackupJobs(t *testing.T) {
	input := `
services:
  server:
    image: docker.io/gitea/gitea:1.23.1-rootless
    volumes:
      - data:/var/lib/gitea
      - config:/etc/gitea

  backup_data:
    image: offen/docker-volume-backup:v2.43.3
    environment:
      BACKUP_FILENAME: "backup-data-%Y-%m-%dT%H-%M-%S.{{ .Extension }}"
    volumes:
      - data:/backup/gitea-data-backup:ro
      - smb_backup:/archive

  backup_config:
    image: offen/docker-volume-backup:v2.43.3
    environment:
      BACKUP_FILENAME: "backup-config-%Y-%m-%dT%H-%M-%S.{{ .Extension }}"
    volumes:
      - config:/backup/gitea-config-backup:ro
      - smb_backup:/archive

volumes:
  data:
  config:
  smb_backup:
`

	result, err := compose.Parse(input)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(result.BackupJobs) != 2 {
		t.Fatalf("backup jobs: got %d, want 2", len(result.BackupJobs))
	}

	dataJob := findBackupJob(result.BackupJobs, "data")
	if dataJob == nil {
		t.Fatal("missing backup job for volume data")
	}
	if dataJob.FilenameFormat != "backup-data-%Y-%m-%dT%H-%M-%S.{{ .Extension }}" {
		t.Fatalf("data filename format: got %q", dataJob.FilenameFormat)
	}

	configJob := findBackupJob(result.BackupJobs, "config")
	if configJob == nil {
		t.Fatal("missing backup job for volume config")
	}
	if configJob.Service != "backup_config" {
		t.Fatalf("config service: got %q, want %q", configJob.Service, "backup_config")
	}

	// Volumes should only come from the non-offen service
	if len(result.Volumes) != 2 {
		t.Fatalf("volumes: got %d, want 2 (offen service volumes excluded)", len(result.Volumes))
	}
	for _, v := range result.Volumes {
		if v.Service != "server" {
			t.Fatalf("volume %q belongs to service %q, want only %q", v.Name, v.Service, "server")
		}
	}
}

func TestParseSkipsNonOfenServices(t *testing.T) {
	input := `
services:
  app:
    image: myapp:latest
    environment:
      BACKUP_FILENAME: "something.tar.gz"
    volumes:
      - appdata:/data:ro

volumes:
  appdata:
`

	result, err := compose.Parse(input)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(result.BackupJobs) != 0 {
		t.Fatalf("backup jobs: got %d, want 0", len(result.BackupJobs))
	}
}

func findBackupJob(jobs []compose.BackupJob, sourceVolume string) *compose.BackupJob {
	for i, j := range jobs {
		if j.SourceVolume == sourceVolume {
			return &jobs[i]
		}
	}
	return nil
}

func findVolume(vols []compose.VolumeMapping, name string) *compose.VolumeMapping {
	for i, v := range vols {
		if v.Name == name {
			return &vols[i]
		}
	}
	return nil
}
