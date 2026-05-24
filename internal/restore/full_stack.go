package restore

import (
	"context"
	"fmt"
	"log"

	"github.com/offen/restore-manager/internal/docker"
	"github.com/offen/restore-manager/internal/sse"
	"github.com/offen/restore-manager/internal/storage"
)

func (o *Orchestrator) runFullStack(ctx context.Context, req RestoreRequest, backend storage.Backend) error {
	tmpDir, err := o.createTempDir()
	if err != nil {
		o.sendFailed(req.Token, fmt.Sprintf("create temp dir: %s", err))
		return err
	}
	defer o.removeTempDir(tmpDir)

	if err := o.downloadBackup(ctx, req, backend, tmpDir); err != nil {
		return err
	}

	containers, err := o.findContainers(ctx, req)
	if err != nil {
		return err
	}

	stopOrder := o.buildStopOrder(req, containers)
	if err := o.stopContainers(ctx, req.Token, stopOrder); err != nil {
		return err
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

	startOrder := o.buildStartOrder(req, containers)
	o.startContainers(ctx, req.Token, startOrder)

	o.send(req.Token, "complete", "done", "Restore complete")
	o.broadcaster.Complete(req.Token)
	return nil
}

func (o *Orchestrator) findContainers(ctx context.Context, req RestoreRequest) ([]docker.ContainerInfo, error) {
	o.send(req.Token, "finding_containers", "in_progress", "Discovering containers")

	if len(req.ContainerIDs) > 0 {
		log.Printf("[restore] token=%s using %d provided container IDs", req.Token, len(req.ContainerIDs))
		var containers []docker.ContainerInfo
		for _, id := range req.ContainerIDs {
			containers = append(containers, docker.ContainerInfo{ID: id})
		}
		return containers, nil
	}

	containers, err := o.docker.FindContainers(ctx, req.Project, req.Services)
	if err != nil {
		o.sendFailed(req.Token, fmt.Sprintf("find containers: %s", err))
		return nil, err
	}

	if len(containers) == 0 {
		msg := "no containers found — provide container IDs manually"
		o.sendFailed(req.Token, msg)
		return nil, fmt.Errorf("%s", msg)
	}

	log.Printf("[restore] token=%s found %d containers", req.Token, len(containers))
	return containers, nil
}

func (o *Orchestrator) buildStopOrder(req RestoreRequest, containers []docker.ContainerInfo) []docker.ContainerInfo {
	if len(req.DependsOn) == 0 {
		return containers
	}

	services := make([]string, 0, len(containers))
	serviceMap := make(map[string]docker.ContainerInfo)
	for _, c := range containers {
		if c.Service != "" {
			services = append(services, c.Service)
			serviceMap[c.Service] = c
		}
	}

	forward := TopoSort(services, req.DependsOn)
	reversed := ReverseOrder(forward)

	ordered := make([]docker.ContainerInfo, 0, len(reversed))
	for _, svc := range reversed {
		if c, ok := serviceMap[svc]; ok {
			ordered = append(ordered, c)
		}
	}

	for _, c := range containers {
		if c.Service == "" {
			ordered = append(ordered, c)
		}
	}

	return ordered
}

func (o *Orchestrator) buildStartOrder(req RestoreRequest, containers []docker.ContainerInfo) []docker.ContainerInfo {
	if len(req.DependsOn) == 0 {
		return containers
	}

	services := make([]string, 0, len(containers))
	serviceMap := make(map[string]docker.ContainerInfo)
	for _, c := range containers {
		if c.Service != "" {
			services = append(services, c.Service)
			serviceMap[c.Service] = c
		}
	}

	forward := TopoSort(services, req.DependsOn)

	ordered := make([]docker.ContainerInfo, 0, len(forward))
	for _, svc := range forward {
		if c, ok := serviceMap[svc]; ok {
			ordered = append(ordered, c)
		}
	}

	for _, c := range containers {
		if c.Service == "" {
			ordered = append(ordered, c)
		}
	}

	return ordered
}

func (o *Orchestrator) stopContainers(ctx context.Context, token string, containers []docker.ContainerInfo) error {
	o.send(token, "stopping_containers", "in_progress", fmt.Sprintf("Stopping %d containers", len(containers)))

	for _, c := range containers {
		name := c.Name
		if name == "" {
			name = c.ID
		}
		o.send(token, "stopping_containers", "in_progress", fmt.Sprintf("Stopping %s", name))
		if err := o.docker.StopContainer(ctx, c.ID); err != nil {
			o.sendFailed(token, fmt.Sprintf("stop container %s: %s", name, err))
			return err
		}
	}
	return nil
}

func (o *Orchestrator) startContainers(ctx context.Context, token string, containers []docker.ContainerInfo) {
	o.send(token, "starting_containers", "in_progress", fmt.Sprintf("Starting %d containers", len(containers)))

	for _, c := range containers {
		name := c.Name
		if name == "" {
			name = c.ID
		}
		o.send(token, "starting_containers", "in_progress", fmt.Sprintf("Starting %s", name))
		if err := o.docker.StartContainer(ctx, c.ID); err != nil {
			o.broadcaster.Send(token, sse.Event{
				Step:    "starting_containers",
				Status:  "error",
				Message: fmt.Sprintf("Failed to start %s: %s", name, err),
			})
		}
	}
}
