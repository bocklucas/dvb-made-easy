package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/offen/restore-manager/internal/compose"
	"github.com/offen/restore-manager/internal/config"
)

type importRequest struct {
	ComposeContent string `json:"compose_content"`
	DeploymentMode string `json:"deployment_mode"`
}

type importResponse struct {
	ID        string           `json:"id"`
	Volumes   []volumeResponse `json:"volumes"`
	DriftHash string           `json:"drift_hash"`
}

type volumeResponse struct {
	Name             string `json:"name"`
	ComposeService   string `json:"compose_service"`
	ComposeMountPath string `json:"compose_mount_path"`
	BackupPattern    string `json:"backup_pattern,omitempty"`
}

type createRequest struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	DeploymentMode string `json:"deployment_mode"`
}

func (s *Server) handleImportCompose(w http.ResponseWriter, r *http.Request) {
	var req importRequest
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

	volumes, volResponses := parseComposeVolumes(result)

	project := config.Project{
		ComposeContent: req.ComposeContent,
		Volumes:        volumes,
		Source:         "paste",
		DeploymentMode: req.DeploymentMode,
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

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	_, ok := s.manifest.GetProject(req.ID)
	if !ok {
		http.Error(w, `{"error":"project not found — import first"}`, http.StatusNotFound)
		return
	}

	s.manifest.SetProjectName(req.ID, req.Name)
	if req.DeploymentMode != "" {
		s.manifest.SetProjectDeploymentMode(req.ID, req.DeploymentMode)
	}
	s.persistConfig()

	project, _ := s.manifest.GetProject(req.ID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(project)
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	projects := s.manifest.ListProjects()
	sanitized := make([]config.Project, len(projects))
	for i, p := range projects {
		sanitized[i] = sanitizeProject(p)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sanitized)
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	project, ok := s.manifest.GetProject(id)
	if !ok {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sanitizeProject(project))
}

func sanitizeProject(p config.Project) config.Project {
	if p.Credentials != nil {
		safe := p.Credentials.Sanitize()
		p.Credentials = &safe
	}
	if p.GitSource != nil {
		safe := *p.GitSource
		safe.AuthToken = ""
		safe.SSHPrivateKey = ""
		p.GitSource = &safe
	}
	if p.PortainerSource != nil {
		safe := *p.PortainerSource
		safe.APIKey = ""
		p.PortainerSource = &safe
	}
	return p
}

type updateProjectRequest struct {
	Name           string  `json:"name"`
	DeploymentMode string  `json:"deployment_mode"`
	SwarmName      *string `json:"swarm_name"`
}

func (s *Server) handleUpdateProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	_, ok := s.manifest.GetProject(id)
	if !ok {
		http.Error(w, `{"error":"project not found"}`, http.StatusNotFound)
		return
	}

	if req.Name != "" {
		s.manifest.SetProjectName(id, req.Name)
	}
	if req.DeploymentMode != "" {
		s.manifest.SetProjectDeploymentMode(id, req.DeploymentMode)
	}
	if req.SwarmName != nil {
		s.manifest.SetProjectSwarmName(id, *req.SwarmName)
	}
	s.persistConfig()

	project, _ := s.manifest.GetProject(id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sanitizeProject(project))
}

func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if !s.manifest.RemoveProject(id) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	s.persistConfig()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) persistConfig() {
	config.Save(s.configDir, s.key, s.manifest)
}
