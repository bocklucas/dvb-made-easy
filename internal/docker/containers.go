package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
)

func (c *Client) FindContainers(ctx context.Context, project string, services []string) ([]ContainerInfo, error) {
	containers, err := c.findByLabels(ctx, project)
	if err != nil {
		return nil, err
	}
	if len(containers) > 0 {
		return containers, nil
	}

	containers, err = c.findByNamePattern(ctx, project, services)
	if err != nil {
		return nil, err
	}
	return containers, nil
}

func (c *Client) findByLabels(ctx context.Context, project string) ([]ContainerInfo, error) {
	args := filters.NewArgs()
	args.Add("label", "com.docker.compose.project="+project)

	list, err := c.cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: args,
	})
	if err != nil {
		return nil, fmt.Errorf("list containers by label: %w", err)
	}

	var result []ContainerInfo
	for _, ct := range list {
		name := ""
		if len(ct.Names) > 0 {
			name = strings.TrimPrefix(ct.Names[0], "/")
		}
		result = append(result, ContainerInfo{
			ID:      ct.ID,
			Name:    name,
			Service: ct.Labels["com.docker.compose.service"],
			State:   ct.State,
			Labels:  ct.Labels,
		})
	}
	return result, nil
}

func (c *Client) findByNamePattern(ctx context.Context, project string, services []string) ([]ContainerInfo, error) {
	list, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("list all containers: %w", err)
	}

	serviceSet := make(map[string]bool, len(services))
	for _, s := range services {
		serviceSet[s] = true
	}

	var result []ContainerInfo
	for _, ct := range list {
		name := ""
		if len(ct.Names) > 0 {
			name = strings.TrimPrefix(ct.Names[0], "/")
		}
		for svc := range serviceSet {
			prefix := project + "_" + svc + "_"
			prefixDash := project + "-" + svc + "-"
			if strings.HasPrefix(name, prefix) || strings.HasPrefix(name, prefixDash) {
				result = append(result, ContainerInfo{
					ID:      ct.ID,
					Name:    name,
					Service: svc,
					State:   ct.State,
					Labels:  ct.Labels,
				})
				break
			}
		}
	}
	return result, nil
}

func (c *Client) StopContainer(ctx context.Context, id string) error {
	timeout := 30
	if err := c.cli.ContainerStop(ctx, id, container.StopOptions{Timeout: &timeout}); err != nil {
		return fmt.Errorf("stop container %s: %w", id, err)
	}
	return nil
}

func (c *Client) StartContainer(ctx context.Context, id string) error {
	if err := c.cli.ContainerStart(ctx, id, container.StartOptions{}); err != nil {
		return fmt.Errorf("start container %s: %w", id, err)
	}
	return nil
}
