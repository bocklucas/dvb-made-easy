package docker

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
)

func (c *Client) RunOneShot(ctx context.Context, cfg OneShotConfig, logWriter io.Writer) (int, error) {
	var mounts []mount.Mount
	for _, m := range cfg.Mounts {
		mounts = append(mounts, mount.Mount{
			Type:     mount.TypeVolume,
			Source:   m.Source,
			Target:   m.Target,
			ReadOnly: m.ReadOnly,
		})
	}

	resp, err := c.cli.ContainerCreate(ctx,
		&container.Config{
			Image: cfg.Image,
			Cmd:   cfg.Cmd,
			Env:   cfg.Env,
		},
		&container.HostConfig{
			Mounts:     mounts,
			AutoRemove: cfg.AutoRemove,
		},
		&network.NetworkingConfig{},
		nil,
		"",
	)
	if err != nil {
		return -1, fmt.Errorf("create oneshot container: %w", err)
	}
	containerID := resp.ID

	defer func() {
		if !cfg.AutoRemove {
			c.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})
		}
	}()

	if err := c.cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return -1, fmt.Errorf("start oneshot container: %w", err)
	}

	waitCh, errCh := c.cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)

	if logWriter != nil {
		logs, err := c.cli.ContainerLogs(ctx, containerID, container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     true,
		})
		if err == nil {
			io.Copy(logWriter, logs)
			logs.Close()
		}
	}

	select {
	case result := <-waitCh:
		if result.Error != nil {
			return int(result.StatusCode), fmt.Errorf("oneshot container error: %s", result.Error.Message)
		}
		return int(result.StatusCode), nil
	case err := <-errCh:
		return -1, fmt.Errorf("wait for oneshot container: %w", err)
	case <-ctx.Done():
		return -1, ctx.Err()
	}
}

func (c *Client) PullImageIfMissing(ctx context.Context, img string) error {
	_, _, err := c.cli.ImageInspectWithRaw(ctx, img)
	if err == nil {
		return nil
	}

	reader, err := c.cli.ImagePull(ctx, img, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("pull image %s: %w", img, err)
	}
	defer reader.Close()
	io.Copy(io.Discard, reader)

	return nil
}
