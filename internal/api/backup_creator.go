package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/bocklucas/dvb-made-easy/internal/compose"
	"github.com/bocklucas/dvb-made-easy/internal/gitimport"
	"github.com/bocklucas/dvb-made-easy/internal/portainer"
)

type bcParseRequest struct {
	ComposeContent string `json:"compose_content"`
}

type bcParseResponse struct {
	ProjectName    string                   `json:"project_name,omitempty"`
	Volumes        []volumeResponse         `json:"volumes"`
	Services       []string                 `json:"services"`
	CronExpression string                   `json:"cron_expression,omitempty"`
	GpgPassphrase  string                   `json:"gpg_passphrase,omitempty"`
	RetentionDays  int                      `json:"retention_days,omitempty"`
	BackupImage    string                   `json:"backup_image,omitempty"`
	StopServices   []string                 `json:"stop_services,omitempty"`
	EnvVars        map[string]string        `json:"env_vars,omitempty"`
	BackupVolumes  []string                 `json:"backup_volumes,omitempty"`
	SMBConfig      *compose.SMBVolumeConfig `json:"smb_config,omitempty"`
}

type bcGitFetchRequest struct {
	RepoURL       string `json:"repo_url"`
	Branch        string `json:"branch"`
	FilePath      string `json:"file_path"`
	AuthToken     string `json:"auth_token,omitempty"`
	SSHPrivateKey string `json:"ssh_private_key,omitempty"`
	SavedSourceID string `json:"saved_source_id,omitempty"`
}

type bcGitFetchResponse struct {
	ComposeContent string `json:"compose_content"`
}

type bcPortainerFetchRequest struct {
	PortainerURL  string `json:"portainer_url"`
	APIKey        string `json:"api_key"`
	StackID       int    `json:"stack_id"`
	SavedSourceID string `json:"saved_source_id,omitempty"`
}

type bcPortainerFetchResponse struct {
	ComposeContent string `json:"compose_content"`
}

type bcGenerateRequest struct {
	ComposeContent  string                    `json:"compose_content"`
	ServiceName     string                    `json:"service_name"`
	Image           string                    `json:"image"`
	CronExpression  string                    `json:"cron_expression"`
	FilenameFormat  string                    `json:"filename_format"`
	GpgPassphrase   string                    `json:"gpg_passphrase"`
	RetentionDays   int                       `json:"retention_days"`
	SelectedVolumes []string                  `json:"selected_volumes"`
	StopServices    []string                  `json:"stop_services"`
	EnvVars         map[string]string         `json:"env_vars"`
	Volumes         []string                  `json:"volumes"`
	SMBConfig       *compose.SMBVolumeConfig  `json:"smb_config,omitempty"`
}

type bcGenerateResponse struct {
	ComposeContent string `json:"compose_content"`
}

func (s *Server) handleBCParse(w http.ResponseWriter, r *http.Request) {
	var req bcParseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if req.ComposeContent == "" {
		http.Error(w, `{"error":"compose_content is required"}`, http.StatusBadRequest)
		return
	}

	result, err := compose.Parse(req.ComposeContent)
	if err != nil {
		jsonError(w, "failed to parse compose file: "+err.Error(), http.StatusBadRequest)
		return
	}

	_, vols := parseComposeVolumes(result)

	var cronExpr string
	var gpgPass string
	var retDays int
	envVars := make(map[string]string)

	if result.BackupServiceEnv != nil {
		cronExpr = result.BackupServiceEnv["BACKUP_CRON_EXPRESSION"]
		gpgPass = result.BackupServiceEnv["GPG_PASSPHRASE"]
		if retStr, ok := result.BackupServiceEnv["BACKUP_RETENTION_DAYS"]; ok {
			fmt.Sscanf(retStr, "%d", &retDays)
		}

		for k, v := range result.BackupServiceEnv {
			if k != "BACKUP_CRON_EXPRESSION" && k != "GPG_PASSPHRASE" && k != "BACKUP_RETENTION_DAYS" && k != "BACKUP_FILENAME" {
				envVars[k] = v
			}
		}
	}

	var smbConfig *compose.SMBVolumeConfig
	if smbDef, ok := result.VolumeDefs["smb_backup"]; ok {
		if smbDef.Driver == "local" && smbDef.DriverOpts["type"] == "cifs" {
			device := smbDef.DriverOpts["device"]
			o := smbDef.DriverOpts["o"]

			host, share, path := compose.ParseCIFSDevice(device)
			username := compose.ParseCIFSOption(o, "username")
			password := compose.ParseCIFSOption(o, "password")

			var port int
			if portStr := compose.ParseCIFSOption(o, "port"); portStr != "" {
				fmt.Sscanf(portStr, "%d", &port)
			}

			useEnvVars := false
			if compose.ParseCIFSOption(o, "addr") == "${SMB_BACKUP_ADDR}" {
				useEnvVars = true
			}
			if strings.Contains(o, "addr=${SMB_BACKUP_ADDR}") {
				useEnvVars = true
			}

			smbConfig = &compose.SMBVolumeConfig{
				Host:       host,
				Share:      share,
				Path:       path,
				Username:   username,
				Password:   password,
				Port:       port,
				UseEnvVars: useEnvVars,
			}
		}
	}

	resp := bcParseResponse{
		ProjectName:    result.Name,
		Volumes:        vols,
		Services:       result.Services,
		CronExpression: cronExpr,
		GpgPassphrase:  gpgPass,
		RetentionDays:  retDays,
		BackupImage:    result.BackupImage,
		StopServices:   result.StopServices,
		EnvVars:        envVars,
		BackupVolumes:  result.BackupServiceVolumes,
		SMBConfig:      smbConfig,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleBCGitFetch(w http.ResponseWriter, r *http.Request) {
	var req bcGitFetchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if req.RepoURL == "" {
		http.Error(w, `{"error":"repo_url is required"}`, http.StatusBadRequest)
		return
	}
	if req.Branch == "" {
		req.Branch = "main"
	}
	if req.FilePath == "" {
		req.FilePath = "docker-compose.yml"
	}

	gs := gitimport.GitSource{
		RepoURL:       req.RepoURL,
		Branch:        req.Branch,
		FilePath:      req.FilePath,
		AuthToken:     req.AuthToken,
		SSHPrivateKey: req.SSHPrivateKey,
	}

	if req.SavedSourceID != "" {
		saved, ok := s.manifest.GetSavedSource(req.SavedSourceID)
		if ok && saved.GitConfig != nil {
			mergeGitConfigs(&gs, saved.GitConfig)
		}
	}

	content, _, err := gitimport.Clone(s.configDir, gs)
	if err != nil {
		jsonError(w, "git clone failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	resp := bcGitFetchResponse{
		ComposeContent: content,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleBCGenerate(w http.ResponseWriter, r *http.Request) {
	var req bcGenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if req.ComposeContent == "" {
		http.Error(w, `{"error":"compose_content is required"}`, http.StatusBadRequest)
		return
	}

	opts := compose.GenerateOptions{
		ServiceName:     req.ServiceName,
		Image:           req.Image,
		CronExpression:  req.CronExpression,
		FilenameFormat:  req.FilenameFormat,
		GpgPassphrase:   req.GpgPassphrase,
		RetentionDays:   req.RetentionDays,
		SelectedVolumes: req.SelectedVolumes,
		StopServices:    req.StopServices,
		EnvVars:         req.EnvVars,
		Volumes:         req.Volumes,
		SMBConfig:       req.SMBConfig,
	}

	generated, err := compose.Generate(req.ComposeContent, opts)
	if err != nil {
		jsonError(w, "failed to generate compose file: "+err.Error(), http.StatusBadRequest)
		return
	}

	resp := bcGenerateResponse{
		ComposeContent: generated,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleBCPortainerFetch(w http.ResponseWriter, r *http.Request) {
	var req bcPortainerFetchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if req.SavedSourceID != "" {
		saved, ok := s.manifest.GetSavedSource(req.SavedSourceID)
		if ok && saved.PortainerConfig != nil {
			if req.PortainerURL == "" {
				req.PortainerURL = saved.PortainerConfig.PortainerURL
			}
			if req.APIKey == "" {
				req.APIKey = saved.PortainerConfig.APIKey
			}
		}
	}

	if req.PortainerURL == "" || req.APIKey == "" || req.StackID == 0 {
		http.Error(w, `{"error":"portainer_url, api_key, and stack_id are required"}`, http.StatusBadRequest)
		return
	}

	client := portainer.NewClient(req.PortainerURL, req.APIKey)
	composeContent, err := client.GetStackFile(r.Context(), req.StackID)
	if err != nil {
		jsonError(w, "failed to get stack file from portainer: "+err.Error(), http.StatusBadGateway)
		return
	}

	resp := bcPortainerFetchResponse{
		ComposeContent: composeContent,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
