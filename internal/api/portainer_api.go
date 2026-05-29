package api

import (
	"encoding/json"
	"net/http"

	"github.com/bocklucas/dvb-made-easy/internal/compose"
	"github.com/bocklucas/dvb-made-easy/internal/config"
	"github.com/bocklucas/dvb-made-easy/internal/portainer"
)

type portainerConnectRequest struct {
	URL           string `json:"url"`
	APIKey        string `json:"api_key"`
	SavedSourceID string `json:"saved_source_id,omitempty"`
}

type portainerConnectResponse struct {
	Endpoints []portainer.Endpoint `json:"endpoints"`
}

type importPortainerRequest struct {
	PortainerURL   string `json:"portainer_url"`
	APIKey         string `json:"api_key"`
	StackID        int    `json:"stack_id"`
	EndpointID     int    `json:"endpoint_id"`
	ProjectName    string `json:"project_name"`
	DeploymentMode string `json:"deployment_mode"`
	SavedSourceID  string `json:"saved_source_id,omitempty"`
}

func (s *Server) handlePortainerConnect(w http.ResponseWriter, r *http.Request) {
	var req portainerConnectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if req.SavedSourceID != "" {
		saved, ok := s.manifest.GetSavedSource(req.SavedSourceID)
		if ok && saved.PortainerConfig != nil {
			if req.URL == "" {
				req.URL = saved.PortainerConfig.PortainerURL
			}
			if req.APIKey == "" {
				req.APIKey = saved.PortainerConfig.APIKey
			}
		}
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
	URL           string `json:"url"`
	APIKey        string `json:"api_key"`
	EndpointID    int    `json:"endpoint_id"`
	SavedSourceID string `json:"saved_source_id,omitempty"`
}

func (s *Server) handlePortainerStacks(w http.ResponseWriter, r *http.Request) {
	var req portainerStacksRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if req.SavedSourceID != "" {
		saved, ok := s.manifest.GetSavedSource(req.SavedSourceID)
		if ok && saved.PortainerConfig != nil {
			if req.URL == "" {
				req.URL = saved.PortainerConfig.PortainerURL
			}
			if req.APIKey == "" {
				req.APIKey = saved.PortainerConfig.APIKey
			}
		}
	}

	if req.URL == "" || req.APIKey == "" {
		http.Error(w, `{"error":"url and api_key are required"}`, http.StatusBadRequest)
		return
	}

	client := portainer.NewClient(req.URL, req.APIKey)

	endpoints, err := client.ListEndpoints(r.Context())
	if err != nil {
		jsonError(w, "failed to list endpoints: "+err.Error(), http.StatusBadGateway)
		return
	}
	endpointMap := make(map[int]string)
	for _, ep := range endpoints {
		endpointMap[ep.ID] = ep.Name
	}

	stacks, err := client.ListStacks(r.Context(), req.EndpointID)
	if err != nil {
		jsonError(w, "failed to list stacks: "+err.Error(), http.StatusBadGateway)
		return
	}

	// Annotate each stack with IsDvbBacked by fetching its compose file.
	for i, stack := range stacks {
		content, err := client.GetStackFile(r.Context(), stack.ID)
		if err == nil {
			stacks[i].IsDvbBacked = portainer.IsDvbBacked(content)
		}
		if name, ok := endpointMap[stack.EndpointID]; ok {
			stacks[i].EndpointName = name
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

	volumes, volResponses := parseComposeVolumes(result)

	ps := &portainer.PortainerSource{
		PortainerURL: req.PortainerURL,
		StackID:      req.StackID,
		EndpointID:   req.EndpointID,
	}

	project := config.Project{
		Name:            req.ProjectName,
		DeploymentMode:  req.DeploymentMode,
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
