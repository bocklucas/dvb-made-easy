package restore

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/offen/restore-manager/internal/sse"
	"github.com/offen/restore-manager/internal/storage"
)

type RestoreMode string

const (
	ModeNewVolume RestoreMode = "new_volume"
	ModeFullStack RestoreMode = "full_stack"
)

type RestoreRequest struct {
	Token          string
	Project        string
	StackName      string
	Services       []string
	DependsOn      map[string][]string
	VolumeName     string
	BackupKey      string
	Mode           RestoreMode
	TargetName     string
	Passphrase     string
	ContainerIDs   []string
	BackupSize     int64
	DeploymentMode string
}

type Orchestrator struct {
	docker      DockerClient
	broadcaster *sse.Broadcaster
	stagingDir  string
}

func NewOrchestrator(docker DockerClient, broadcaster *sse.Broadcaster) *Orchestrator {
	return &Orchestrator{
		docker:      docker,
		broadcaster: broadcaster,
		stagingDir:  "/staging",
	}
}

func (o *Orchestrator) Run(ctx context.Context, req RestoreRequest, backend storage.Backend) error {
	log.Printf("[restore] starting restore token=%s mode=%s volume=%s backup=%s", req.Token, req.Mode, req.VolumeName, req.BackupKey)

	if req.TargetName == "" {
		prefix := req.StackName
		if prefix == "" {
			prefix = req.Project
		}
		req.TargetName = fmt.Sprintf("%s_%s", prefix, req.VolumeName)
	}

	var err error
	switch req.Mode {
	case ModeNewVolume:
		err = o.runNewVolume(ctx, req, backend)
	case ModeFullStack:
		err = o.runFullStack(ctx, req, backend)
	default:
		o.sendFailed(req.Token, fmt.Sprintf("unknown restore mode: %s", req.Mode))
		return fmt.Errorf("unknown restore mode: %s", req.Mode)
	}

	if err != nil {
		log.Printf("[restore] failed token=%s: %v", req.Token, err)
	} else {
		log.Printf("[restore] completed token=%s", req.Token)
	}
	return err
}

func (o *Orchestrator) send(token string, step, status, message string) {
	log.Printf("[restore] token=%s step=%s status=%s msg=%s", token, step, status, message)
	o.broadcaster.Send(token, sse.Event{
		Step:    step,
		Status:  status,
		Message: message,
	})
}

func (o *Orchestrator) sendFailed(token, message string) {
	log.Printf("[restore] token=%s FAILED: %s", token, message)
	o.broadcaster.Send(token, sse.Event{
		Step:    "failed",
		Status:  "failed",
		Message: message,
	})
	o.broadcaster.Complete(token)
}

const StagingVolume = "offen-restore-staging"

func (o *Orchestrator) SetStagingDir(dir string) {
	o.stagingDir = dir
}

func (o *Orchestrator) createTempDir() (string, error) {
	return os.MkdirTemp(o.stagingDir, "restore-*")
}

func (o *Orchestrator) removeTempDir(dir string) {
	os.RemoveAll(dir)
}

