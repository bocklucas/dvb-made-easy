package restore

import (
	"context"
	"fmt"
	"log"

	"github.com/bocklucas/dvb-made-easy/internal/docker"
	"github.com/bocklucas/dvb-made-easy/internal/sse"
	"github.com/bocklucas/dvb-made-easy/internal/storage"
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

	var services []docker.ServiceInfo
	var containers []docker.ContainerInfo

	if req.DeploymentMode == "swarm" {
		services, err = o.findServices(ctx, req)
		if err != nil {
			return err
		}
		if err := o.stopServices(ctx, req.Token, services); err != nil {
			return err
		}
	} else {
		containers, err = o.findContainers(ctx, req)
		if err != nil {
			return err
		}
		stopOrder := o.buildStopOrder(req, containers)
		if err := o.stopContainers(ctx, req.Token, stopOrder); err != nil {
			return err
		}
	}

	// Wipe volume if it exists
	_, inspectErr := o.docker.InspectVolume(ctx, req.TargetName)
	if inspectErr == nil {
		o.send(req.Token, "wiping_volume", "in_progress", fmt.Sprintf("Wiping existing volume %s", req.TargetName))
		if err := o.docker.RemoveVolume(ctx, req.TargetName); err != nil {
			if req.DeploymentMode == "swarm" {
				o.startServices(ctx, req.Token, services)
			} else {
				startOrder := o.buildStartOrder(req, containers)
				o.startContainers(ctx, req.Token, startOrder)
			}
			o.sendFailed(req.Token, fmt.Sprintf("wipe volume (remove): %s", err))
			return err
		}
	}

	o.send(req.Token, "creating_volume", "in_progress", fmt.Sprintf("Creating volume %s", req.TargetName))
	if err := o.docker.CreateVolume(ctx, req.TargetName); err != nil {
		if req.DeploymentMode == "swarm" {
			o.startServices(ctx, req.Token, services)
		} else {
			startOrder := o.buildStartOrder(req, containers)
			o.startContainers(ctx, req.Token, startOrder)
		}
		o.sendFailed(req.Token, fmt.Sprintf("create volume: %s", err))
		return err
	}

	if err := o.extractBackup(ctx, req, tmpDir); err != nil {
		o.docker.RemoveVolume(ctx, req.TargetName)
		if req.DeploymentMode == "swarm" {
			o.startServices(ctx, req.Token, services)
		} else {
			startOrder := o.buildStartOrder(req, containers)
			o.startContainers(ctx, req.Token, startOrder)
		}
		return err
	}

	if req.DeploymentMode == "swarm" {
		o.startServices(ctx, req.Token, services)
	} else {
		startOrder := o.buildStartOrder(req, containers)
		o.startContainers(ctx, req.Token, startOrder)
	}

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

func (o *Orchestrator) findServices(ctx context.Context, req RestoreRequest) ([]docker.ServiceInfo, error) {
	o.send(req.Token, "finding_services", "in_progress", "Discovering Swarm services")

	stack := req.StackName
	if stack == "" {
		stack = req.Project
	}
	services, err := o.docker.FindServices(ctx, stack, req.Services)
	if err != nil {
		o.sendFailed(req.Token, fmt.Sprintf("find services: %s", err))
		return nil, err
	}

	if len(services) == 0 {
		msg := "no Swarm services found — make sure stack name is correct"
		o.sendFailed(req.Token, msg)
		return nil, fmt.Errorf("%s", msg)
	}

	log.Printf("[restore] token=%s found %d services", req.Token, len(services))
	return services, nil
}

func (o *Orchestrator) stopServices(ctx context.Context, token string, services []docker.ServiceInfo) error {
	o.send(token, "stopping_services", "in_progress", fmt.Sprintf("Scaling down %d Swarm services", len(services)))

	for _, svc := range services {
		if !svc.IsReplicated {
			log.Printf("[restore] token=%s skipping non-replicated service %s", token, svc.Name)
			continue
		}
		o.send(token, "stopping_services", "in_progress", fmt.Sprintf("Scaling down %s to 0 replicas", svc.Name))
		if err := o.docker.ScaleService(ctx, svc.ID, svc.Version, 0); err != nil {
			o.sendFailed(token, fmt.Sprintf("scale down service %s: %s", svc.Name, err))
			return err
		}
	}
	return nil
}

func (o *Orchestrator) startServices(ctx context.Context, token string, services []docker.ServiceInfo) {
	o.send(token, "starting_services", "in_progress", fmt.Sprintf("Scaling up %d Swarm services", len(services)))

	for _, svc := range services {
		if !svc.IsReplicated {
			continue
		}
		o.send(token, "starting_services", "in_progress", fmt.Sprintf("Scaling up %s to %d replicas", svc.Name, svc.Replicas))
		if err := o.docker.ScaleService(ctx, svc.ID, svc.Version, svc.Replicas); err != nil {
			o.broadcaster.Send(token, sse.Event{
				Step:    "starting_services",
				Status:  "error",
				Message: fmt.Sprintf("Failed to scale up service %s: %s", svc.Name, err),
			})
		}
	}
}

