package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/volume"
)

func (c *Client) CreateVolume(ctx context.Context, name string) error {
	_, err := c.cli.VolumeCreate(ctx, volume.CreateOptions{
		Name: name,
	})
	if err != nil {
		return fmt.Errorf("create volume %s: %w", name, err)
	}
	return nil
}

func (c *Client) RemoveVolume(ctx context.Context, name string) error {
	if err := c.cli.VolumeRemove(ctx, name, false); err != nil {
		return fmt.Errorf("remove volume %s: %w", name, err)
	}
	return nil
}

func (c *Client) InspectVolume(ctx context.Context, name string) (VolumeInfo, error) {
	vol, err := c.cli.VolumeInspect(ctx, name)
	if err != nil {
		return VolumeInfo{}, fmt.Errorf("inspect volume %s: %w", name, err)
	}
	return VolumeInfo{
		Name:       vol.Name,
		Driver:     vol.Driver,
		MountPoint: vol.Mountpoint,
	}, nil
}
