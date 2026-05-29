package restore_test

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/bocklucas/dvb-made-easy/internal/restore"
	"github.com/bocklucas/dvb-made-easy/internal/sse"
)

// failingBackend wraps mockBackend but fails on a specific key.
type failingBackend struct {
	mockBackend
	failOnKey string
}

func (m *failingBackend) Download(ctx context.Context, key string, w io.Writer) error {
	if key == m.failOnKey {
		return fmt.Errorf("simulated download failure for %s", key)
	}
	w.Write([]byte("data"))
	return nil
}

func TestComposeNewVolumeSuccess(t *testing.T) {
	b := sse.NewBroadcaster()
	dc := &mockDockerClient{}
	backend := &mockBackend{data: []byte("backup-data")}

	orch := restore.NewOrchestrator(dc, b)
	orch.SetStagingDir(t.TempDir())

	ch := b.Subscribe("compose-nv-1")
	defer b.Unsubscribe("compose-nv-1", ch)

	req := restore.ComposeRestoreRequest{
		Token:   "compose-nv-1",
		Project: "myproject",
		Mode:    restore.ModeNewVolume,
		Volumes: []restore.VolumeRestore{
			{
				VolumeName: "pgdata",
				BackupKey:  "backup-pgdata.tar.gz",
				TargetName: "pgdata_restored",
				BackupSize: int64(len(backend.data)),
			},
			{
				VolumeName: "redisdata",
				BackupKey:  "backup-redisdata.tar.gz",
				TargetName: "redisdata_restored",
				BackupSize: int64(len(backend.data)),
			},
		},
	}

	err := orch.RunCompose(context.Background(), req, backend)
	if err != nil {
		t.Fatalf("RunCompose: %v", err)
	}

	// Both volumes should be created.
	if len(dc.createdVolumes) != 2 {
		t.Fatalf("created volumes: got %v, want 2", dc.createdVolumes)
	}
	if dc.createdVolumes[0] != "pgdata_restored" {
		t.Fatalf("first volume: got %q, want %q", dc.createdVolumes[0], "pgdata_restored")
	}
	if dc.createdVolumes[1] != "redisdata_restored" {
		t.Fatalf("second volume: got %q, want %q", dc.createdVolumes[1], "redisdata_restored")
	}

	events := drainEvents(ch)

	// Events should have volume_total=2.
	foundVolumeTotal := false
	for _, ev := range events {
		if ev.VolumeTotal == 2 {
			foundVolumeTotal = true
			break
		}
	}
	if !foundVolumeTotal {
		t.Fatalf("no event with volume_total=2 found: %+v", events)
	}

	// "complete" step should be emitted.
	assertHasStep(t, events, "complete")
}
