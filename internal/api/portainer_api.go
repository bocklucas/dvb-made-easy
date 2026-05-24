package api

import (
	"encoding/json"
	"net/http"

	"github.com/offen/restore-manager/internal/compose"
	"github.com/offen/restore-manager/internal/config"
	"github.com/offen/restore-manager/internal/portainer"
)

type portainerConnectRequest struct {
	URL    string `json:"url"`
	APIKey string `json:"api_key"`
}

type portainerConnectResponse struct {
	Endpoints []portainer.Endpoint `json:"endpoints"`
}

type importPortainerRequest struct {
	PortainerURL string `json:"portainer_url"`
	APIKey       string `json:"api_key"`
	StackID      int    `json:"stack_id"`
	EndpointID   int    `json:"endpoint_id"`
	ProjectName  string `json:"project_name"`
}

func (s *Server) handlePortainerConnect(w http.ResponseWriter, r *http.Request) {
	var req portainerConnectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, `{"error":"url is required"}`, http.StatusBadRequest)
		return
	}
	if req.APIKey == "" {
		http.Error(w, `{"error":"api_key is required"}`, http.StatusBadRequest)
		return
	}

	client := portainer.NewClient(req.URL, req.APIKey)

	if err := client.TestConnection(r.Context()); err != nil {
		jsonError(w, "connection failed: "+err.Error(), http.StatusBadGateway)
		return
	}

	endpoints, err := client.ListEndpoints(r.Context())
	if err != nil {
		jsonError(w, "failed to list endpoints: "+err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(portainerConnectResponse{Endpoints: endpoints})
}

type portainerStacksRequest struct {
	URL        string `json:"url"`
	APIKey     string `json:"api_key"`
	EndpointID int    `json:"endpoint_id"`
}

func (s *Server) handlePortainerStacks(w http.ResponseWriter, r *http.Request) {
	var req portainerStacksRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if req.URL == "" || req.APIKey == "" || req.EndpointID == 0 {
		http.Error(w, `{"error":"url, api_key, and endpoint_id are required"}`, http.StatusBadRequest)
		return
	}

	client := portainer.NewClient(req.URL, req.APIKey)

	stacks, err := client.ListStacks(r.Context(), req.EndpointID)
	if err != nil {
		jsonError(w, "failed to list stacks: "+err.Error(), http.StatusBadGateway)
		return
	}

	// Annotate each stack with IsOfenBacked by fetching its compose file.
	for i, stack := range stacks {
		content, err := client.GetStackFile(r.Context(), stack.ID)
		if err == nil {
			stacks[i].IsOfenBacked = portainer.IsOfenBacked(content)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stacks)
}

func (s *Server) handleImportPortainer(w http.ResponseWriter, r *http.Request) {
	var req importPortainerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if req.PortainerURL == "" {
		http.Error(w, `{"error":"portainer_url is required"}`, http.StatusBadRequest)
		return
	}
	if req.APIKey == "" {
		http.Error(w, `{"error":"api_key is required"}`, http.StatusBadRequest)
		return
	}
	if req.StackID == 0 {
		http.Error(w, `{"error":"stack_id is required"}`, http.StatusBadRequest)
		return
	}
	if req.EndpointID == 0 {
		http.Error(w, `{"error":"endpoint_id is required"}`, http.StatusBadRequest)
		return
	}
	if req.ProjectName == "" {
		http.Error(w, `{"error":"project_name is required"}`, http.StatusBadRequest)
		return
	}

	client := portainer.NewClient(req.PortainerURL, req.APIKey)

	composeContent, err := client.GetStackFile(r.Context(), req.StackID)
	if err != nil {
		jsonError(w, "failed to get stack file: "+err.Error(), http.StatusBadGateway)
		return
	}

	result, err := compose.Parse(composeContent)
	if err != nil {
		jsonError(w, "failed to parse compose file: "+err.Error(), http.StatusBadRequest)
		return
	}

	patternMap := make(map[string]string)
	for _, job := range result.BackupJobs {
		patternMap[job.SourceVolume] = job.FilenameFormat
	}

	seen := make(map[string]bool)
	var volumes []config.Volume
	var volResponses []volumeResponse
	for _, v := range result.Volumes {
		if seen[v.Name] {
			continue
		}
		seen[v.Name] = true
		vol := config.Volume{
			Name:             v.Name,
			ComposeService:   v.Service,
			ComposeMountPath: v.MountPath,
			BackupPattern:    patternMap[v.Name],
		}
		volumes = append(volumes, vol)
		volResponses = append(volResponses, volumeResponse{
			Name:             v.Name,
			ComposeService:   v.Service,
			ComposeMountPath: v.MountPath,
			BackupPattern:    vol.BackupPattern,
		})
	}

	ps := &portainer.PortainerSource{
		PortainerURL: req.PortainerURL,
		StackID:      req.StackID,
		EndpointID:   req.EndpointID,
	}

	project := config.Project{
		Name:            req.ProjectName,
		ComposeContent:  composeContent,
		Volumes:         volumes,
		Source:          "portainer",
		PortainerSource: ps,
	}
	staged := s.manifest.AddProject(project)

	s.persistConfig()

	resp := importResponse{
		ID:        staged.ID,
		Volumes:   volResponses,
		DriftHash: staged.ComposeHash,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
