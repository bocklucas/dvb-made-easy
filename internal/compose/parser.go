package compose

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type ParseResult struct {
	Name                 string
	Volumes              []VolumeMapping
	Services             []string
	DependsOn            map[string][]string
	BackupJobs           []BackupJob
	BackupServiceEnv     map[string]string // env vars from the first offen backup service found
	BackupServiceVolumes []string          // raw volume mount strings from the backup service
	VolumeDefs           map[string]VolumeDef
	BackupImage          string            // image of the first backup service found
	StopServices         []string          // list of services that have the stop label
}

type VolumeDef struct {
	Driver     string
	DriverOpts map[string]string
}

type VolumeMapping struct {
	Name      string
	Service   string
	MountPath string
}

type BackupJob struct {
	Service        string
	SourceVolume   string
	FilenameFormat string
}

type composeFile struct {
	Name     string                `yaml:"name"`
	Services map[string]serviceDef `yaml:"services"`
	Volumes  map[string]volumeDef  `yaml:"volumes"`
}

type volumeDef struct {
	Driver     string            `yaml:"driver"`
	DriverOpts map[string]string `yaml:"driver_opts"`
}

type serviceDef struct {
	Image       string       `yaml:"image"`
	Volumes     []any        `yaml:"volumes"`
	DependsOn   dependsOnDef `yaml:"depends_on"`
	Environment envDef       `yaml:"environment"`
	Labels      labelsDef    `yaml:"labels"`
}

type labelsDef struct {
	Labels map[string]string
}

func (l *labelsDef) UnmarshalYAML(node *yaml.Node) error {
	l.Labels = make(map[string]string)
	switch node.Kind {
	case yaml.MappingNode:
		return node.Decode(&l.Labels)
	case yaml.SequenceNode:
		var items []string
		if err := node.Decode(&items); err != nil {
			return err
		}
		for _, item := range items {
			k, v, _ := strings.Cut(item, "=")
			l.Labels[k] = v
		}
		return nil
	default:
		return nil
	}
}

type envDef struct {
	Vars map[string]string
}

func (e *envDef) UnmarshalYAML(node *yaml.Node) error {
	e.Vars = make(map[string]string)
	switch node.Kind {
	case yaml.MappingNode:
		return node.Decode(&e.Vars)
	case yaml.SequenceNode:
		var items []string
		if err := node.Decode(&items); err != nil {
			return err
		}
		for _, item := range items {
			k, v, _ := strings.Cut(item, "=")
			e.Vars[k] = v
		}
		return nil
	default:
		return nil
	}
}

type dependsOnDef struct {
	Services []string
}

func (d *dependsOnDef) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.SequenceNode:
		return node.Decode(&d.Services)
	case yaml.MappingNode:
		var m map[string]any
		if err := node.Decode(&m); err != nil {
			return err
		}
		for k := range m {
			d.Services = append(d.Services, k)
		}
		return nil
	default:
		return fmt.Errorf("unexpected depends_on type: %v", node.Kind)
	}
}

func Parse(content string) (*ParseResult, error) {
	var cf composeFile
	if err := yaml.Unmarshal([]byte(content), &cf); err != nil {
		return nil, fmt.Errorf("parse compose file: %w", err)
	}

	topLevelVolumes := make(map[string]bool)
	for name := range cf.Volumes {
		topLevelVolumes[name] = true
	}

	result := &ParseResult{
		Name:       cf.Name,
		DependsOn:  make(map[string][]string),
		VolumeDefs: make(map[string]VolumeDef),
	}

	for name, v := range cf.Volumes {
		result.VolumeDefs[name] = VolumeDef{
			Driver:     v.Driver,
			DriverOpts: v.DriverOpts,
		}
	}

	for svcName, svc := range cf.Services {
		result.Services = append(result.Services, svcName)

		if len(svc.DependsOn.Services) > 0 {
			result.DependsOn[svcName] = svc.DependsOn.Services
		}

		isBackupService := strings.HasPrefix(svc.Image, ofenImagePrefix)

		if !isBackupService {
			for _, vol := range svc.Volumes {
				vm, ok := parseVolumeEntry(vol, svcName, topLevelVolumes)
				if ok {
					result.Volumes = append(result.Volumes, vm)
				}
			}
			if val, ok := svc.Labels.Labels["docker-volume-backup.stop-during-backup"]; ok && val == "true" {
				result.StopServices = append(result.StopServices, svcName)
			}
		} else {
			if result.BackupImage == "" {
				result.BackupImage = svc.Image
			}
			if result.BackupServiceEnv == nil {
				// Capture env and volume mounts from the first backup service encountered.
				result.BackupServiceEnv = svc.Environment.Vars
				for _, vol := range svc.Volumes {
					if s, ok := vol.(string); ok {
						result.BackupServiceVolumes = append(result.BackupServiceVolumes, s)
					}
				}
			}
		}

		if job, ok := parseBackupJob(svcName, svc, topLevelVolumes); ok {
			result.BackupJobs = append(result.BackupJobs, job)
		}
	}

	// Ensure all top-level named volumes appear in result.Volumes, even if they
	// are only mounted in backup services. Scan all services to find a mount.
	captured := make(map[string]bool, len(result.Volumes))
	for _, v := range result.Volumes {
		captured[v.Name] = true
	}
	for volName := range topLevelVolumes {
		if captured[volName] {
			continue
		}
		if vm, ok := findVolumeMountInServices(volName, cf.Services, topLevelVolumes); ok {
			result.Volumes = append(result.Volumes, vm)
		}
	}

	return result, nil
}

const ofenImagePrefix = "offen/docker-volume-backup"

// findVolumeMountInServices scans all services to find where a named volume is
// mounted. It prefers non-backup services; if none mount the volume it falls
// back to the first backup service that does.
func findVolumeMountInServices(volName string, services map[string]serviceDef, topLevel map[string]bool) (VolumeMapping, bool) {
	var fallback *VolumeMapping
	for svcName, svc := range services {
		isBackup := strings.HasPrefix(svc.Image, ofenImagePrefix)
		for _, vol := range svc.Volumes {
			vm, ok := parseVolumeEntry(vol, svcName, topLevel)
			if !ok || vm.Name != volName {
				continue
			}
			if !isBackup {
				return vm, true
			}
			if fallback == nil {
				cp := vm
				fallback = &cp
			}
		}
	}
	if fallback != nil {
		return *fallback, true
	}
	return VolumeMapping{}, false
}

func parseBackupJob(svcName string, svc serviceDef, topLevel map[string]bool) (BackupJob, bool) {
	if !strings.HasPrefix(svc.Image, ofenImagePrefix) {
		return BackupJob{}, false
	}

	filenameFormat := svc.Environment.Vars["BACKUP_FILENAME"]
	if filenameFormat == "" {
		return BackupJob{}, false
	}

	sourceVolume := ""
	for _, vol := range svc.Volumes {
		s, ok := vol.(string)
		if !ok {
			continue
		}
		parts := strings.SplitN(s, ":", 3)
		if len(parts) < 2 {
			continue
		}
		isReadOnly := len(parts) == 3 && strings.Contains(parts[2], "ro")
		if isReadOnly && topLevel[parts[0]] {
			sourceVolume = parts[0]
			break
		}
	}

	if sourceVolume == "" {
		return BackupJob{}, false
	}

	return BackupJob{
		Service:        svcName,
		SourceVolume:   sourceVolume,
		FilenameFormat: filenameFormat,
	}, true
}

func parseVolumeEntry(entry any, serviceName string, topLevel map[string]bool) (VolumeMapping, bool) {
	switch v := entry.(type) {
	case string:
		return parseShortVolume(v, serviceName, topLevel)
	case map[string]any:
		return parseLongVolume(v, serviceName, topLevel)
	default:
		return VolumeMapping{}, false
	}
}

func parseShortVolume(s string, serviceName string, topLevel map[string]bool) (VolumeMapping, bool) {
	parts := strings.SplitN(s, ":", 3)
	if len(parts) < 2 {
		return VolumeMapping{}, false
	}

	source := parts[0]
	target := parts[1]

	if strings.HasPrefix(source, ".") || strings.HasPrefix(source, "/") || strings.HasPrefix(source, "~") {
		return VolumeMapping{}, false
	}

	if !topLevel[source] {
		return VolumeMapping{}, false
	}

	return VolumeMapping{
		Name:      source,
		Service:   serviceName,
		MountPath: target,
	}, true
}

func parseLongVolume(m map[string]any, serviceName string, topLevel map[string]bool) (VolumeMapping, bool) {
	volType, _ := m["type"].(string)
	if volType != "volume" {
		return VolumeMapping{}, false
	}

	source, _ := m["source"].(string)
	target, _ := m["target"].(string)

	if source == "" || target == "" {
		return VolumeMapping{}, false
	}

	if !topLevel[source] {
		return VolumeMapping{}, false
	}

	return VolumeMapping{
		Name:      source,
		Service:   serviceName,
		MountPath: target,
	}, true
}
