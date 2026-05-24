package restore

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/offen/restore-manager/internal/sse"
	"github.com/offen/restore-manager/internal/storage"
)

// ComposeRestoreRequest describes a multi-volume compose restore operation.
type ComposeRestoreRequest struct {
	Token        string
	Project      string
	Services     []string
	DependsOn    map[string][]string
	Mode         RestoreMode
	Volumes      []VolumeRestore
	ContainerIDs []string
}

// VolumeRestore describes a single volume to restore within a compose operation.
type VolumeRestore struct {
	VolumeName string
	BackupKey  string
	Passphrase string
	TargetName string
	BackupSize int64
}

// RunCompose dispatches to runComposeNewVolume or runComposeFullStack.
func (o *Orchestrator) RunCompose(ctx context.Context, req ComposeRestoreRequest, backend storage.Backend) error {
	log.Printf("[restore] starting compose restore token=%s mode=%s volumes=%d", req.Token, req.Mode, len(req.Volumes))

	var err error
	switch req.Mode {
	case ModeNewVolume:
		err = o.runComposeNewVolume(ctx, req, backend)
	case ModeFullStack:
		err = o.runComposeFullStack(ctx, req, backend)
	default:
		o.sendFailed(req.Token, fmt.Sprintf("unknown restore mode: %s", req.Mode))
		return fmt.Errorf("unknown restore mode: %s", req.Mode)
	}

	if err != nil {
		log.Printf("[restore] compose restore failed token=%s: %v", req.Token, err)
	} else {
		log.Printf("[restore] compose restore completed token=%s", req.Token)
	}
	return err
}

// sendVolume sends an SSE event with volume context fields populated.
func (o *Orchestrator) sendVolume(token, step, status, message, volume string, index, total int) {
	log.Printf("[restore] token=%s step=%s volume=%s (%d/%d) msg=%s", token, step, volume, index, total, message)
	o.broadcaster.Send(token, sse.Event{
		Step:        step,
		Status:      status,
		Message:     message,
		Volume:      volume,
		VolumeIndex: index,
		VolumeTotal: total,
	})
}

// runComposeNewVolume restores each volume sequentially in new_volume mode.
func (o *Orchestrator) runComposeNewVolume(ctx context.Context, req ComposeRestoreRequest, backend storage.Backend) error {
	total := len(req.Volumes)

	for i, vol := range req.Volumes {
		targetName := vol.TargetName
		if targetName == "" {
			targetName = fmt.Sprintf("%s_%s", req.Project, vol.VolumeName)
		}

		tmpDir, err := o.createTempDir()
		if err != nil {
			o.sendFailed(req.Token, fmt.Sprintf("create temp dir: %s", err))
			return err
		}

		if err := o.composeDownloadToDir(ctx, req.Token, vol, i, total, backend, tmpDir); err != nil {
			o.removeTempDir(tmpDir)
			return err
		}

		o.sendVolume(req.Token, "creating_volume", "in_progress",
			fmt.Sprintf("Creating volume %s", targetName),
			vol.VolumeName, i+1, total)

		if err := o.docker.CreateVolume(ctx, targetName); err != nil {
			o.removeTempDir(tmpDir)
			o.sendFailed(req.Token, fmt.Sprintf("create volume: %s", err))
			return err
		}

		o.sendVolume(req.Token, "extracting", "in_progress",
			fmt.Sprintf("Extracting backup into volume %s", targetName),
			vol.VolumeName, i+1, total)

		if err := o.docker.PullImageIfMissing(ctx, "alpine:latest"); err != nil {
			o.removeTempDir(tmpDir)
			o.docker.RemoveVolume(ctx, targetName)
			o.sendFailed(req.Token, fmt.Sprintf("pull image: %s", err))
			return err
		}

		cfg := BuildExtractionConfig(filepath.Base(vol.BackupKey), StagingVolume, filepath.Base(tmpDir), targetName, vol.Passphrase)
		exitCode, runErr := o.docker.RunOneShot(ctx, cfg, nil)
		o.removeTempDir(tmpDir)

		if runErr != nil {
			o.docker.RemoveVolume(ctx, targetName)
			o.sendFailed(req.Token, fmt.Sprintf("extraction failed: %s", runErr))
			return runErr
		}
		if exitCode != 0 {
			o.docker.RemoveVolume(ctx, targetName)
			msg := fmt.Sprintf("extraction container exited with code %d", exitCode)
			o.sendFailed(req.Token, msg)
			return fmt.Errorf("%s", msg)
		}
	}

	o.broadcaster.Send(req.Token, sse.Event{
		Step:    "complete",
		Status:  "done",
		Message: "Restore complete",
	})
	o.broadcaster.Complete(req.Token)
	return nil
}

// runComposeFullStack restores each volume sequentially in full_stack mode.
// Containers are stopped before processing and always restarted afterwards,
// even if a volume fails.
func (o *Orchestrator) runComposeFullStack(ctx context.Context, req ComposeRestoreRequest, backend storage.Backend) error {
	// Build a synthetic single RestoreRequest for container discovery methods.
	containerReq := RestoreRequest{
		Token:        req.Token,
		Project:      req.Project,
		Services:     req.Services,
		DependsOn:    req.DependsOn,
		ContainerIDs: req.ContainerIDs,
	}

	containers, err := o.findContainers(ctx, containerReq)
	if err != nil {
		return err
	}

	stopOrder := o.buildStopOrder(containerReq, containers)
	if err := o.stopContainers(ctx, req.Token, stopOrder); err != nil {
		return err
	}

	startOrder := o.buildStartOrder(containerReq, containers)

	total := len(req.Volumes)
	var firstErr error

	for i, vol := range req.Volumes {
		targetName := vol.TargetName
		if targetName == "" {
			targetName = vol.VolumeName
		}

		tmpDir, err := o.createTempDir()
		if err != nil {
			o.sendFailed(req.Token, fmt.Sprintf("create temp dir: %s", err))
			firstErr = err
			break
		}

		if err := o.composeDownloadToDir(ctx, req.Token, vol, i, total, backend, tmpDir); err != nil {
			o.removeTempDir(tmpDir)
			firstErr = err
			break
		}

		o.sendVolume(req.Token, "extracting", "in_progress",
			fmt.Sprintf("Extracting backup into volume %s", targetName),
			vol.VolumeName, i+1, total)

		if err := o.docker.PullImageIfMissing(ctx, "alpine:latest"); err != nil {
			o.removeTempDir(tmpDir)
			o.sendFailed(req.Token, fmt.Sprintf("pull image: %s", err))
			firstErr = err
			break
		}

		cfg := BuildExtractionConfig(filepath.Base(vol.BackupKey), StagingVolume, filepath.Base(tmpDir), targetName, vol.Passphrase)
		exitCode, runErr := o.docker.RunOneShot(ctx, cfg, nil)
		o.removeTempDir(tmpDir)

		if runErr != nil {
			o.sendFailed(req.Token, fmt.Sprintf("extraction failed: %s", runErr))
			firstErr = runErr
			break
		}
		if exitCode != 0 {
			msg := fmt.Sprintf("extraction container exited with code %d", exitCode)
			o.sendFailed(req.Token, msg)
			firstErr = fmt.Errorf("%s", msg)
			break
		}
	}

	// Always restart containers regardless of errors above.
	o.startContainers(ctx, req.Token, startOrder)

	if firstErr != nil {
		return firstErr
	}

	o.broadcaster.Send(req.Token, sse.Event{
		Step:    "complete",
		Status:  "done",
		Message: "Restore complete",
	})
	o.broadcaster.Complete(req.Token)
	return nil
}

// composeDownloadToDir downloads a backup to a temp directory, sending volume-aware SSE events.
func (o *Orchestrator) composeDownloadToDir(ctx context.Context, token string, vol VolumeRestore, index, total int, backend storage.Backend, tmpDir string) error {
	o.sendVolume(token, "downloading", "in_progress",
		fmt.Sprintf("Downloading backup for %s (0%%)", vol.VolumeName),
		vol.VolumeName, index+1, total)

	tmpFile, err := os.Create(filepath.Join(tmpDir, filepath.Base(vol.BackupKey)))
	if err != nil {
		o.sendFailed(token, fmt.Sprintf("create temp file: %s", err))
		return err
	}
	defer tmpFile.Close()

	var dlErr error
	if vol.BackupSize > 0 {
		writer := NewProgressWriter(tmpFile, vol.BackupSize, o.broadcaster, token)
		dlErr = backend.Download(ctx, vol.BackupKey, writer)
	} else {
		dlErr = backend.Download(ctx, vol.BackupKey, tmpFile)
	}

	if dlErr != nil {
		o.sendFailed(token, fmt.Sprintf("download backup: %s", dlErr))
		return dlErr
	}

	return nil
}
