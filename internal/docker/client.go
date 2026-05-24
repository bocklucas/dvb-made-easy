package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/client"
)

type Client struct {
	cli *client.Client
}

type ContainerInfo struct {
	ID      string
	Name    string
	Service string
	State   string
	Labels  map[string]string
}

type OneShotConfig struct {
	Image      string
	Cmd        []string
	Env        []string
	Mounts     []Mount
	AutoRemove bool
}

type Mount struct {
	Source   string
	Target   string
	ReadOnly bool
}

type VolumeInfo struct {
	Name       string
	Driver     string
	MountPoint string
}

func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("create docker client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := cli.Ping(ctx); err != nil {
		cli.Close()
		return nil, fmt.Errorf("ping docker daemon: %w", err)
	}

	return &Client{cli: cli}, nil
}

func (c *Client) Close() {
	c.cli.Close()
}
