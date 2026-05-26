package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
)

func (c *Client) FindServices(ctx context.Context, stack string, services []string) ([]ServiceInfo, error) {
	args := filters.NewArgs()
	args.Add("label", "com.docker.stack.namespace="+stack)

	list, err := c.cli.ServiceList(ctx, types.ServiceListOptions{
		Filters: args,
	})
	if err != nil {
		return nil, fmt.Errorf("list swarm services: %w", err)
	}

	serviceSet := make(map[string]bool, len(services))
	for _, s := range services {
		serviceSet[s] = true
	}

	var result []ServiceInfo
	for _, svc := range list {
		// Extract compose service name if possible
		compSvc := svc.Spec.Name
		if strings.HasPrefix(compSvc, stack+"_") {
			compSvc = strings.TrimPrefix(compSvc, stack+"_")
		}

		// Filter by services if specified
		if len(services) > 0 {
			if !serviceSet[compSvc] {
				continue
			}
		}

		var replicas uint64
		isReplicated := false
		if svc.Spec.Mode.Replicated != nil && svc.Spec.Mode.Replicated.Replicas != nil {
			replicas = *svc.Spec.Mode.Replicated.Replicas
			isReplicated = true
		}

		result = append(result, ServiceInfo{
			ID:           svc.ID,
			Name:         svc.Spec.Name,
			Stack:        stack,
			Service:      compSvc,
			Replicas:     replicas,
			IsReplicated: isReplicated,
			Version:      svc.Meta.Version.Index,
		})
	}
	return result, nil
}

func (c *Client) ScaleService(ctx context.Context, id string, version uint64, replicas uint64) error {
	svc, _, err := c.cli.ServiceInspectWithRaw(ctx, id, types.ServiceInspectOptions{})
	if err != nil {
		return fmt.Errorf("inspect service %s: %w", id, err)
	}

	if svc.Spec.Mode.Replicated == nil {
		return fmt.Errorf("service %s is not replicated", id)
	}

	svc.Spec.Mode.Replicated.Replicas = &replicas

	_, err = c.cli.ServiceUpdate(ctx, id, svc.Meta.Version, svc.Spec, types.ServiceUpdateOptions{})
	if err != nil {
		return fmt.Errorf("scale service %s: %w", id, err)
	}
	return nil
}
