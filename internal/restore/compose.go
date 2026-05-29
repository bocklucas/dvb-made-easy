package restore

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/bocklucas/dvb-made-easy/internal/sse"
	"github.com/bocklucas/dvb-made-easy/internal/storage"
)

type ComposeRestoreRequest struct {
	Token          string
	Project        string
	StackName      string
	Services       []string
	DependsOn      map[string][]string
	Mode           RestoreMode
	Volumes        []VolumeRestore
	ContainerIDs   []string
	DeploymentMode string
}

// VolumeRestore describes a single volume to restore within a compose operation.
type VolumeRestore struct {
	VolumeName string
	BackupKey  string
	Passphrase string
	TargetName string
	BackupSize int64
}

// RunCompose runs the compose restore operation.
func (o *Orchestrator) RunCompose(ctx context.Context, req ComposeRestoreRequest, backend storage.Backend) error {
	log.Printf("[restore] starting compose restore token=%s mode=%s volumes=%d", req.Token, req.Mode, len(req.Volumes))

	if req.Mode != ModeNewVolume {
		o.sendFailed(req.Token, fmt.Sprintf("unsupported restore mode: %s", req.Mode))
		return fmt.Errorf("unsupported restore mode: %s", req.Mode)
	}

	err := o.runComposeNewVolume(ctx, req, backend)
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

	prefix := req.StackName
	if prefix == "" {
		prefix = req.Project
	}

	for i, vol := range req.Volumes {
		targetName := vol.TargetName
		if targetName == "" {
			targetName = fmt.Sprintf("%s_%s", prefix, vol.VolumeName)
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

		// Wipe volume if it exists
		_, inspectErr := o.docker.InspectVolume(ctx, targetName)
		if inspectErr == nil {
			o.sendVolume(req.Token, "wiping_volume", "in_progress",
				fmt.Sprintf("Wiping existing volume %s", targetName),
				vol.VolumeName, i+1, total)
			if err := o.docker.RemoveVolume(ctx, targetName); err != nil {
				o.removeTempDir(tmpDir)
				o.sendFailed(req.Token, fmt.Sprintf("wipe volume (remove): %s", err))
				return err
			}
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

		cfg, cfgErr := BuildExtractionConfig(filepath.Base(vol.BackupKey), StagingVolume, filepath.Base(tmpDir), targetName, vol.Passphrase)
		if cfgErr != nil {
			o.removeTempDir(tmpDir)
			o.docker.RemoveVolume(ctx, targetName)
			o.sendFailed(req.Token, fmt.Sprintf("invalid backup filename: %s", cfgErr))
			return cfgErr
		}
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

// composeDownloadToDir downloads a backup to a temp directory, sending volume-aware SSE events.
func (o *Orchestrator) composeDownloadToDir(ctx context.Context, token string, vol VolumeRestore, index, total int, backend storage.Backend, tmpDir string) error {
	o.sendVolume(token, "downloading", "in_progress",
		fmt.Sprintf("Downloading backup for %s (0%%)", vol.VolumeName),
		vol.VolumeName, index+1, total)

	backupName := filepath.Base(vol.BackupKey)
	if backupName == "." || backupName == string(os.PathSeparator) {
		o.sendFailed(token, "invalid backup key")
		return fmt.Errorf("invalid backup key: %s", vol.BackupKey)
	}
	tmpFile, err := os.Create(filepath.Join(tmpDir, backupName))
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

	o.sendVolume(token, "downloading", "done",
		fmt.Sprintf("Downloaded backup for %s", vol.VolumeName),
		vol.VolumeName, index+1, total)

	return nil
}
