package compose

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type ParseResult struct {
	Volumes              []VolumeMapping
	Services             []string
	DependsOn            map[string][]string
	BackupJobs           []BackupJob
	BackupServiceEnv     map[string]string // env vars from the first offen backup service found
	BackupServiceVolumes []string          // raw volume mount strings from the backup service
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
	Services map[string]serviceDef `yaml:"services"`
	Volumes  map[string]any        `yaml:"volumes"`
}

type serviceDef struct {
	Image       string       `yaml:"image"`
	Volumes     []any        `yaml:"volumes"`
	DependsOn   dependsOnDef `yaml:"depends_on"`
	Environment envDef       `yaml:"environment"`
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
		DependsOn: make(map[string][]string),
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
		} else if result.BackupServiceEnv == nil {
			// Capture env and volume mounts from the first backup service encountered.
			result.BackupServiceEnv = svc.Environment.Vars
			for _, vol := range svc.Volumes {
				if s, ok := vol.(string); ok {
					result.BackupServiceVolumes = append(result.BackupServiceVolumes, s)
				}
			}
		}

		if job, ok := parseBackupJob(svcName, svc, topLevelVolumes); ok {
			result.BackupJobs = append(result.BackupJobs, job)
		}
	}

	return result, nil
}

const ofenImagePrefix = "offen/docker-volume-backup"

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
