package restore_test

import (
	"context"
	"io"
	"regexp"
	"testing"
	"time"

	"github.com/offen/restore-manager/internal/docker"
	"github.com/offen/restore-manager/internal/restore"
	"github.com/offen/restore-manager/internal/sse"
	"github.com/offen/restore-manager/internal/storage"
)

type mockDockerClient struct {
	createdVolumes  []string
	removedVolumes  []string
	oneshotConfigs  []docker.OneShotConfig
	oneshotExitCode int
	oneshotErr      error
	pulledImages    []string
}

func (m *mockDockerClient) CreateVolume(ctx context.Context, name string) error {
	m.createdVolumes = append(m.createdVolumes, name)
	return nil
}

func (m *mockDockerClient) RemoveVolume(ctx context.Context, name string) error {
	m.removedVolumes = append(m.removedVolumes, name)
	return nil
}

func (m *mockDockerClient) InspectVolume(ctx context.Context, name string) (docker.VolumeInfo, error) {
	return docker.VolumeInfo{Name: name}, nil
}

func (m *mockDockerClient) FindContainers(ctx context.Context, project string, services []string) ([]docker.ContainerInfo, error) {
	return nil, nil
}

func (m *mockDockerClient) StopContainer(ctx context.Context, id string) error {
	return nil
}

func (m *mockDockerClient) StartContainer(ctx context.Context, id string) error {
	return nil
}

func (m *mockDockerClient) RunOneShot(ctx context.Context, cfg docker.OneShotConfig, logWriter io.Writer) (int, error) {
	m.oneshotConfigs = append(m.oneshotConfigs, cfg)
	return m.oneshotExitCode, m.oneshotErr
}

func (m *mockDockerClient) PullImageIfMissing(ctx context.Context, image string) error {
	m.pulledImages = append(m.pulledImages, image)
	return nil
}

type mockBackend struct {
	data []byte
}

func (m *mockBackend) ListBackups(ctx context.Context, pattern *regexp.Regexp) ([]storage.BackupFile, error) {
	return nil, nil
}

func (m *mockBackend) Download(ctx context.Context, key string, w io.Writer) error {
	w.Write(m.data)
	return nil
}

func (m *mockBackend) TestConnection(ctx context.Context) error {
	return nil
}

func TestNewVolumeFlowSuccess(t *testing.T) {
	b := sse.NewBroadcaster()
	dc := &mockDockerClient{}
	backend := &mockBackend{data: []byte("fake-backup-data")}

	orch := restore.NewOrchestrator(dc, b)
	orch.SetStagingDir(t.TempDir())

	req := restore.RestoreRequest{
		Token:      "test-restore-1",
		VolumeName: "pgdata",
		BackupKey:  "backup-2026-05-15T04-00-00.tar.gz",
		Mode:       restore.ModeNewVolume,
		TargetName: "pgdata_restored_20260521",
		BackupSize: int64(len(backend.data)),
	}

	ch := b.Subscribe("test-restore-1")
	defer b.Unsubscribe("test-restore-1", ch)

	err := orch.Run(context.Background(), req, backend)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	if len(dc.createdVolumes) != 1 || dc.createdVolumes[0] != "pgdata_restored_20260521" {
		t.Fatalf("created volumes: %v", dc.createdVolumes)
	}

	if len(dc.pulledImages) != 1 || dc.pulledImages[0] != "alpine:latest" {
		t.Fatalf("pulled images: %v", dc.pulledImages)
	}

	if len(dc.oneshotConfigs) != 1 {
		t.Fatalf("oneshot configs: got %d, want 1", len(dc.oneshotConfigs))
	}

	events := drainEvents(ch)
	assertHasStep(t, events, "downloading")
	assertHasStep(t, events, "creating_volume")
	assertHasStep(t, events, "extracting")
	assertHasStep(t, events, "complete")
}

func TestNewVolumeAutoGeneratesTargetName(t *testing.T) {
	b := sse.NewBroadcaster()
	dc := &mockDockerClient{}
	backend := &mockBackend{data: []byte("data")}

	orch := restore.NewOrchestrator(dc, b)
	orch.SetStagingDir(t.TempDir())

	req := restore.RestoreRequest{
		Token:      "test-restore-2",
		Project:    "gitea",
		VolumeName: "pgdata",
		BackupKey:  "backup-2026-05-15T04-00-00.tar.gz",
		Mode:       restore.ModeNewVolume,
		BackupSize: int64(len(backend.data)),
	}

	err := orch.Run(context.Background(), req, backend)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	if len(dc.createdVolumes) != 1 {
		t.Fatalf("created volumes: got %d, want 1", len(dc.createdVolumes))
	}

	name := dc.createdVolumes[0]
	if name != "gitea_pgdata" {
		t.Fatalf("auto-generated name: got %q, want %q", name, "gitea_pgdata")
	}
}

func drainEvents(ch chan sse.Event) []sse.Event {
	var events []sse.Event
	for {
		select {
		case ev := <-ch:
			events = append(events, ev)
		case <-time.After(100 * time.Millisecond):
			return events
		}
	}
}

func assertHasStep(t *testing.T, events []sse.Event, step string) {
	t.Helper()
	for _, ev := range events {
		if ev.Step == step {
			return
		}
	}
	t.Fatalf("missing step %q in events: %+v", step, events)
}
