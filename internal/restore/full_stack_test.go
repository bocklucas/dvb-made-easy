package restore_test

import (
	"context"
	"errors"
	"testing"

	"github.com/offen/restore-manager/internal/docker"
	"github.com/offen/restore-manager/internal/restore"
	"github.com/offen/restore-manager/internal/sse"
)

type fullStackMockDocker struct {
	mockDockerClient
	containers []docker.ContainerInfo
	stoppedIDs []string
	startedIDs []string
	startErr   map[string]error
}

func (m *fullStackMockDocker) FindContainers(ctx context.Context, project string, services []string) ([]docker.ContainerInfo, error) {
	return m.containers, nil
}

func (m *fullStackMockDocker) StopContainer(ctx context.Context, id string) error {
	m.stoppedIDs = append(m.stoppedIDs, id)
	return nil
}

func (m *fullStackMockDocker) StartContainer(ctx context.Context, id string) error {
	m.startedIDs = append(m.startedIDs, id)
	if m.startErr != nil {
		if err, ok := m.startErr[id]; ok {
			return err
		}
	}
	return nil
}

func TestFullStackFlowSuccess(t *testing.T) {
	b := sse.NewBroadcaster()
	dc := &fullStackMockDocker{
		containers: []docker.ContainerInfo{
			{ID: "c1", Name: "proj-postgres-1", Service: "postgres", State: "running"},
			{ID: "c2", Name: "proj-app-1", Service: "app", State: "running"},
		},
	}
	backend := &mockBackend{data: []byte("backup-data")}

	orch := restore.NewOrchestrator(dc, b)
	orch.SetStagingDir(t.TempDir())

	ch := b.Subscribe("fs-test-1")
	defer b.Unsubscribe("fs-test-1", ch)

	req := restore.RestoreRequest{
		Token:      "fs-test-1",
		Project:    "proj",
		Services:   []string{"postgres", "app"},
		DependsOn:  map[string][]string{"app": {"postgres"}},
		VolumeName: "pgdata",
		BackupKey:  "backup-2026-05-15T04-00-00.tar.gz",
		Mode:       restore.ModeFullStack,
		TargetName: "pgdata_restored",
		BackupSize: int64(len(backend.data)),
	}

	err := orch.Run(context.Background(), req, backend)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	// Stop order: reverse topo — app before postgres
	if len(dc.stoppedIDs) != 2 {
		t.Fatalf("stopped: got %v, want 2 containers", dc.stoppedIDs)
	}
	if dc.stoppedIDs[0] != "c2" {
		t.Fatalf("stop order: first stopped %q, want c2 (app)", dc.stoppedIDs[0])
	}

	// Start order: forward topo — postgres before app
	if len(dc.startedIDs) != 2 {
		t.Fatalf("started: got %v, want 2 containers", dc.startedIDs)
	}
	if dc.startedIDs[0] != "c1" {
		t.Fatalf("start order: first started %q, want c1 (postgres)", dc.startedIDs[0])
	}

	events := drainEvents(ch)
	assertHasStep(t, events, "finding_containers")
	assertHasStep(t, events, "stopping_containers")
	assertHasStep(t, events, "starting_containers")
	assertHasStep(t, events, "complete")
}

func TestFullStackNoContainersNoManualIDs(t *testing.T) {
	b := sse.NewBroadcaster()
	dc := &fullStackMockDocker{
		containers: nil,
	}
	backend := &mockBackend{data: []byte("data")}

	orch := restore.NewOrchestrator(dc, b)
	orch.SetStagingDir(t.TempDir())

	ch := b.Subscribe("fs-test-2")
	defer b.Unsubscribe("fs-test-2", ch)

	req := restore.RestoreRequest{
		Token:      "fs-test-2",
		Project:    "proj",
		Services:   []string{"postgres"},
		VolumeName: "pgdata",
		BackupKey:  "backup.tar.gz",
		Mode:       restore.ModeFullStack,
		TargetName: "pgdata_restored",
		BackupSize: 4,
	}

	err := orch.Run(context.Background(), req, backend)
	if err == nil {
		t.Fatal("expected error when no containers found and no manual IDs")
	}

	events := drainEvents(ch)
	assertHasStep(t, events, "failed")
}

func TestFullStackManualContainerIDs(t *testing.T) {
	b := sse.NewBroadcaster()
	dc := &fullStackMockDocker{
		containers: nil,
	}
	backend := &mockBackend{data: []byte("data")}

	orch := restore.NewOrchestrator(dc, b)
	orch.SetStagingDir(t.TempDir())

	req := restore.RestoreRequest{
		Token:        "fs-test-3",
		Project:      "proj",
		Services:     []string{"postgres"},
		VolumeName:   "pgdata",
		BackupKey:    "backup.tar.gz",
		Mode:         restore.ModeFullStack,
		TargetName:   "pgdata_restored",
		ContainerIDs: []string{"manual-c1", "manual-c2"},
		BackupSize:   4,
	}

	err := orch.Run(context.Background(), req, backend)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	if len(dc.stoppedIDs) != 2 {
		t.Fatalf("stopped: got %v, want 2", dc.stoppedIDs)
	}
}

func TestFullStackContinuesOnStartFailure(t *testing.T) {
	b := sse.NewBroadcaster()
	dc := &fullStackMockDocker{
		containers: []docker.ContainerInfo{
			{ID: "c1", Service: "postgres", State: "running"},
			{ID: "c2", Service: "app", State: "running"},
		},
		startErr: map[string]error{"c2": errors.New("start failed")},
	}
	backend := &mockBackend{data: []byte("data")}

	orch := restore.NewOrchestrator(dc, b)
	orch.SetStagingDir(t.TempDir())

	ch := b.Subscribe("fs-test-4")
	defer b.Unsubscribe("fs-test-4", ch)

	req := restore.RestoreRequest{
		Token:      "fs-test-4",
		Project:    "proj",
		Services:   []string{"postgres", "app"},
		VolumeName: "pgdata",
		BackupKey:  "backup.tar.gz",
		Mode:       restore.ModeFullStack,
		TargetName: "pgdata_restored",
		BackupSize: 4,
	}

	err := orch.Run(context.Background(), req, backend)
	if err != nil {
		t.Fatalf("run should succeed even with partial start failure: %v", err)
	}

	events := drainEvents(ch)
	hasError := false
	for _, ev := range events {
		if ev.Status == "error" {
			hasError = true
		}
	}
	if !hasError {
		t.Fatal("expected an error event for failed container start")
	}
	assertHasStep(t, events, "complete")
}

