package restore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bocklucas/dvb-made-easy/internal/storage"
)

func (o *Orchestrator) runNewVolume(ctx context.Context, req RestoreRequest, backend storage.Backend) error {
	tmpDir, err := o.createTempDir()
	if err != nil {
		o.sendFailed(req.Token, fmt.Sprintf("create temp dir: %s", err))
		return err
	}
	defer o.removeTempDir(tmpDir)

	if err := o.downloadBackup(ctx, req, backend, tmpDir); err != nil {
		return err
	}

	// Wipe volume if it exists
	_, inspectErr := o.docker.InspectVolume(ctx, req.TargetName)
	if inspectErr == nil {
		o.send(req.Token, "wiping_volume", "in_progress", fmt.Sprintf("Wiping existing volume %s", req.TargetName))
		if err := o.docker.RemoveVolume(ctx, req.TargetName); err != nil {
			o.sendFailed(req.Token, fmt.Sprintf("wipe volume (remove): %s", err))
			return err
		}
	}

	o.send(req.Token, "creating_volume", "in_progress", fmt.Sprintf("Creating volume %s", req.TargetName))
	if err := o.docker.CreateVolume(ctx, req.TargetName); err != nil {
		o.sendFailed(req.Token, fmt.Sprintf("create volume: %s", err))
		return err
	}

	if err := o.extractBackup(ctx, req, tmpDir); err != nil {
		o.docker.RemoveVolume(ctx, req.TargetName)
		return err
	}

	o.send(req.Token, "complete", "done", "Restore complete")
	o.broadcaster.Complete(req.Token)
	return nil
}

func (o *Orchestrator) downloadBackup(ctx context.Context, req RestoreRequest, backend storage.Backend, tmpDir string) error {
	o.send(req.Token, "downloading", "in_progress", "Downloading backup (0%)")

	backupName := filepath.Base(req.BackupKey)
	if backupName == "." || backupName == string(os.PathSeparator) {
		o.sendFailed(req.Token, "invalid backup key")
		return fmt.Errorf("invalid backup key: %s", req.BackupKey)
	}
	targetPath := filepath.Join(tmpDir, backupName)
	if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(tmpDir)+string(os.PathSeparator)) {
		o.sendFailed(req.Token, "invalid backup key")
		return fmt.Errorf("path traversal in backup key: %s", req.BackupKey)
	}
	tmpFile, err := os.Create(targetPath)
	if err != nil {
		o.sendFailed(req.Token, fmt.Sprintf("create temp file: %s", err))
		return err
	}
	defer tmpFile.Close()

	var writer *ProgressWriter
	if req.BackupSize > 0 {
		writer = NewProgressWriter(tmpFile, req.BackupSize, o.broadcaster, req.Token)
	}

	if writer != nil {
		err = backend.Download(ctx, req.BackupKey, writer)
	} else {
		err = backend.Download(ctx, req.BackupKey, tmpFile)
	}
	if err != nil {
		o.sendFailed(req.Token, fmt.Sprintf("download backup: %s", err))
		return err
	}

	o.send(req.Token, "downloading", "done", "Download complete")
	return nil
}

func (o *Orchestrator) extractBackup(ctx context.Context, req RestoreRequest, tmpDir string) error {
	o.send(req.Token, "extracting", "in_progress", "Extracting backup into volume")

	if err := o.docker.PullImageIfMissing(ctx, "alpine:latest"); err != nil {
		o.sendFailed(req.Token, fmt.Sprintf("pull image: %s", err))
		return err
	}

	cfg, err := BuildExtractionConfig(filepath.Base(req.BackupKey), StagingVolume, filepath.Base(tmpDir), req.TargetName, req.Passphrase)
	if err != nil {
		o.sendFailed(req.Token, fmt.Sprintf("invalid backup filename: %s", err))
		return err
	}

	exitCode, err := o.docker.RunOneShot(ctx, cfg, nil)
	if err != nil {
		o.sendFailed(req.Token, fmt.Sprintf("extraction failed: %s", err))
		return err
	}
	if exitCode != 0 {
		msg := fmt.Sprintf("extraction container exited with code %d", exitCode)
		o.sendFailed(req.Token, msg)
		return fmt.Errorf("%s", msg)
	}

	return nil
}
