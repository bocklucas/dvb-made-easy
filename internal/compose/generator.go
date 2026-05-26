package compose

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type SMBVolumeConfig struct {
	Host       string `json:"host"`
	Share      string `json:"share"`
	Path       string `json:"path,omitempty"`
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
	Port       int    `json:"port,omitempty"`
	UseEnvVars bool   `json:"use_env_vars,omitempty"`
}

// GenerateOptions represents all parameters to inject the backup service.
type GenerateOptions struct {
	ServiceName     string            `json:"service_name"`
	Image           string            `json:"image"`
	CronExpression  string            `json:"cron_expression"`
	FilenameFormat  string            `json:"filename_format"`
	GpgPassphrase   string            `json:"gpg_passphrase"`
	RetentionDays   int               `json:"retention_days"`
	SelectedVolumes []string          `json:"selected_volumes"`
	StopServices    []string          `json:"stop_services"`
	EnvVars         map[string]string `json:"env_vars"`
	Volumes         []string          `json:"volumes"` // Extra volumes to mount
	SMBConfig       *SMBVolumeConfig  `json:"smb_config,omitempty"`
}

// Generate inserts or updates the offen/docker-volume-backup sidecar service in the docker-compose YAML.
func Generate(composeContent string, opts GenerateOptions) (string, error) {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(composeContent), &root); err != nil {
		return "", fmt.Errorf("failed to parse compose file: %w", err)
	}

	if len(root.Content) == 0 {
		return "", fmt.Errorf("empty compose file")
	}

	docNode := root.Content[0]
	if docNode.Kind != yaml.MappingNode {
		return "", fmt.Errorf("root of compose file must be a map")
	}

	// 1. Find or create the "services" mapping node.
	var servicesNode *yaml.Node
	for i := 0; i < len(docNode.Content); i += 2 {
		if docNode.Content[i].Value == "services" {
			servicesNode = docNode.Content[i+1]
			break
		}
	}

	if servicesNode == nil {
		servicesKey := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!str",
			Value: "services",
		}
		servicesVal := &yaml.Node{
			Kind: yaml.MappingNode,
			Tag:  "!!map",
		}
		docNode.Content = append(docNode.Content, servicesKey, servicesVal)
		servicesNode = servicesVal
	}

	if servicesNode.Kind != yaml.MappingNode {
		return "", fmt.Errorf("'services' block must be a map")
	}

	// 2. Add/update labels on services that need to stop during backup.
	for _, stopSvc := range opts.StopServices {
		var svcNode *yaml.Node
		for i := 0; i < len(servicesNode.Content); i += 2 {
			if servicesNode.Content[i].Value == stopSvc {
				svcNode = servicesNode.Content[i+1]
				break
			}
		}
		if svcNode != nil && svcNode.Kind == yaml.MappingNode {
			var labelsNode *yaml.Node
			for i := 0; i < len(svcNode.Content); i += 2 {
				if svcNode.Content[i].Value == "labels" {
					labelsNode = svcNode.Content[i+1]
					break
				}
			}

			labelKey := "docker-volume-backup.stop-during-backup"
			labelVal := "true"

			if labelsNode == nil {
				labelsKeyNode := &yaml.Node{
					Kind:  yaml.ScalarNode,
					Tag:   "!!str",
					Value: "labels",
				}
				labelsValNode := &yaml.Node{
					Kind: yaml.MappingNode,
					Tag:  "!!map",
				}
				labelsValNode.Content = append(labelsValNode.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: labelKey},
					&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: labelVal},
				)
				svcNode.Content = append(svcNode.Content, labelsKeyNode, labelsValNode)
			} else if labelsNode.Kind == yaml.MappingNode {
				found := false
				for j := 0; j < len(labelsNode.Content); j += 2 {
					if labelsNode.Content[j].Value == labelKey {
						labelsNode.Content[j+1].Value = labelVal
						found = true
						break
					}
				}
				if !found {
					labelsNode.Content = append(labelsNode.Content,
						&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: labelKey},
						&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: labelVal},
					)
				}
			} else if labelsNode.Kind == yaml.SequenceNode {
				found := false
				for _, itemNode := range labelsNode.Content {
					if itemNode.Kind == yaml.ScalarNode {
						if itemNode.Value == labelKey || itemNode.Value == labelKey+"="+labelVal {
							itemNode.Value = labelKey + "=" + labelVal
							found = true
							break
						}
					}
				}
				if !found {
					labelsNode.Content = append(labelsNode.Content, &yaml.Node{
						Kind:  yaml.ScalarNode,
						Tag:   "!!str",
						Value: labelKey + "=" + labelVal,
					})
				}
			}
		}
	}

	// 3. Build the backup service node.
	if opts.Image == "" {
		opts.Image = "offen/docker-volume-backup:latest"
	}
	if opts.ServiceName == "" {
		opts.ServiceName = "backup"
	}

	envMap := make(map[string]string)
	for k, v := range opts.EnvVars {
		envMap[k] = v
	}
	if opts.CronExpression != "" {
		envMap["BACKUP_CRON_EXPRESSION"] = opts.CronExpression
	}
	if opts.FilenameFormat != "" {
		envMap["BACKUP_FILENAME"] = opts.FilenameFormat
	}
	if opts.GpgPassphrase != "" {
		envMap["GPG_PASSPHRASE"] = opts.GpgPassphrase
	}
	if opts.RetentionDays > 0 {
		envMap["BACKUP_RETENTION_DAYS"] = fmt.Sprintf("%d", opts.RetentionDays)
	}

	var volumeList []string
	for _, vol := range opts.SelectedVolumes {
		volumeList = append(volumeList, fmt.Sprintf("%s:/backup/%s:ro", vol, vol))
	}
	if len(opts.StopServices) > 0 {
		volumeList = append(volumeList, "/var/run/docker.sock:/var/run/docker.sock:ro")
	}
	volumeList = append(volumeList, opts.Volumes...)

	backupSvcIdx := -1
	for i := 0; i < len(servicesNode.Content); i += 2 {
		if servicesNode.Content[i].Value == opts.ServiceName {
			backupSvcIdx = i
			break
		}
	}

	type tempSvc struct {
		Image       string            `yaml:"image"`
		Restart     string            `yaml:"restart,omitempty"`
		Environment map[string]string `yaml:"environment,omitempty"`
		Volumes     []string          `yaml:"volumes,omitempty"`
	}
	backupDef := tempSvc{
		Image:       opts.Image,
		Restart:     "always",
		Environment: envMap,
		Volumes:     volumeList,
	}

	var backupValNode yaml.Node
	if err := backupValNode.Encode(backupDef); err != nil {
		return "", fmt.Errorf("failed to encode backup service node: %w", err)
	}

	actualBackupValNode := &backupValNode
	if backupValNode.Kind == yaml.DocumentNode && len(backupValNode.Content) > 0 {
		actualBackupValNode = backupValNode.Content[0]
	}

	backupKeyNode := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: opts.ServiceName,
	}

	if backupSvcIdx >= 0 {
		servicesNode.Content[backupSvcIdx+1] = actualBackupValNode
	} else {
		servicesNode.Content = append(servicesNode.Content, backupKeyNode, actualBackupValNode)
	}

	// 4. If SMBConfig is provided, add/update the top-level volume.
	if opts.SMBConfig != nil {
		var volumesNode *yaml.Node
		for i := 0; i < len(docNode.Content); i += 2 {
			if docNode.Content[i].Value == "volumes" {
				volumesNode = docNode.Content[i+1]
				break
			}
		}

		if volumesNode == nil {
			volumesKey := &yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
				Value: "volumes",
			}
			volumesVal := &yaml.Node{
				Kind: yaml.MappingNode,
				Tag:  "!!map",
			}
			docNode.Content = append(docNode.Content, volumesKey, volumesVal)
			volumesNode = volumesVal
		}

		if volumesNode.Kind == yaml.MappingNode {
			backupVolIdx := -1
			for i := 0; i < len(volumesNode.Content); i += 2 {
				if volumesNode.Content[i].Value == "smb_backup" {
					backupVolIdx = i
					break
				}
			}

			type volumeOpts struct {
				Type   string `yaml:"type"`
				Device string `yaml:"device"`
				O      string `yaml:"o"`
			}
			type volumeDef struct {
				Driver     string     `yaml:"driver"`
				DriverOpts volumeOpts `yaml:"driver_opts"`
			}

			optsList := []string{"vers=3.0"}
			deviceHost := opts.SMBConfig.Host
			deviceShare := opts.SMBConfig.Share
			devicePath := opts.SMBConfig.Path

			if opts.SMBConfig.UseEnvVars {
				deviceHost = "${SMB_BACKUP_ADDR}"
				deviceShare = "${SMB_BACKUP_SHARE}"
				if opts.SMBConfig.Path != "" {
					devicePath = "${SMB_BACKUP_PATH}"
				}
				optsList = append(optsList, "addr=${SMB_BACKUP_ADDR}")
				if opts.SMBConfig.Username != "" {
					optsList = append(optsList, "username=${SMB_BACKUP_USERNAME}")
				}
				if opts.SMBConfig.Password != "" {
					optsList = append(optsList, "password=${SMB_BACKUP_PASSWORD}")
				}
				if opts.SMBConfig.Port != 0 {
					optsList = append(optsList, "port=${SMB_BACKUP_PORT}")
				}
			} else {
				if opts.SMBConfig.Host != "" {
					optsList = append(optsList, "addr="+opts.SMBConfig.Host)
				}
				if opts.SMBConfig.Username != "" {
					optsList = append(optsList, "username="+opts.SMBConfig.Username)
				}
				if opts.SMBConfig.Password != "" {
					optsList = append(optsList, "password="+opts.SMBConfig.Password)
				}
				if opts.SMBConfig.Port != 0 && opts.SMBConfig.Port != 445 {
					optsList = append(optsList, fmt.Sprintf("port=%d", opts.SMBConfig.Port))
				}
			}

			device := fmt.Sprintf("//%s/%s", deviceHost, deviceShare)
			p := devicePath
			if len(p) > 0 && p[0] == '/' {
				p = p[1:]
			}
			if p != "" {
				device = fmt.Sprintf("//%s/%s/%s", deviceHost, deviceShare, p)
			}

			volDef := volumeDef{
				Driver: "local",
				DriverOpts: volumeOpts{
					Type:   "cifs",
					Device: device,
					O:      strings.Join(optsList, ","),
				},
			}

			var newVolNode yaml.Node
			if err := newVolNode.Encode(volDef); err != nil {
				return "", fmt.Errorf("failed to encode backup volume node: %w", err)
			}

			actualVolNode := &newVolNode
			if newVolNode.Kind == yaml.DocumentNode && len(newVolNode.Content) > 0 {
				actualVolNode = newVolNode.Content[0]
			}

			volKeyNode := &yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
				Value: "smb_backup",
			}

			if backupVolIdx >= 0 {
				volumesNode.Content[backupVolIdx+1] = actualVolNode
			} else {
				volumesNode.Content = append(volumesNode.Content, volKeyNode, actualVolNode)
			}
		}
	}

	out, err := yaml.Marshal(&root)
	if err != nil {
		return "", fmt.Errorf("failed to marshal generated compose: %w", err)
	}

	return string(out), nil
}
