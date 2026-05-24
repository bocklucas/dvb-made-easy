package restore

import (
	"context"
	"io"

	"github.com/offen/restore-manager/internal/docker"
)

type DockerClient interface {
	CreateVolume(ctx context.Context, name string) error
	RemoveVolume(ctx context.Context, name string) error
	InspectVolume(ctx context.Context, name string) (docker.VolumeInfo, error)
	FindContainers(ctx context.Context, project string, services []string) ([]docker.ContainerInfo, error)
	StopContainer(ctx context.Context, id string) error
	StartContainer(ctx context.Context, id string) error
	RunOneShot(ctx context.Context, cfg docker.OneShotConfig, logWriter io.Writer) (int, error)
	PullImageIfMissing(ctx context.Context, image string) error
}
