package restore

import (
	"context"
	"io"

	"github.com/bocklucas/dvb-made-easy/internal/docker"
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
	FindServices(ctx context.Context, stack string, services []string) ([]docker.ServiceInfo, error)
	ScaleService(ctx context.Context, id string, version uint64, replicas uint64) error
}
