package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/offen/restore-manager/internal/compose"
	"github.com/offen/restore-manager/internal/config"
	"github.com/offen/restore-manager/internal/gitimport"
)

type gitImportRequest struct {
	RepoURL        string `json:"repo_url"`
	Branch         string `json:"branch"`
	FilePath       string `json:"file_path"`
	AuthToken      string `json:"auth_token,omitempty"`
	SSHPrivateKey  string `json:"ssh_private_key,omitempty"`
	ProjectName    string `json:"project_name"`
	DeploymentMode string `json:"deployment_mode"`
	SavedSourceID  string `json:"saved_source_id,omitempty"`
}

type gitSyncResponse struct {
	Changed bool                `json:"changed"`
	Diff    *composeDiffResponse `json:"diff,omitempty"`
	Commit  string              `json:"commit"`
}

func (s *Server) handleImportGit(w http.ResponseWriter, r *http.Request) {
	var req gitImportRequest
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
	if req.ProjectName == "" {
		http.Error(w, `{"error":"project_name is required"}`, http.StatusBadRequest)
		return
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

	content, commitHash, err := gitimport.Clone(s.configDir, gs)
	if err != nil {
		jsonError(w, "git clone failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	result, err := compose.Parse(content)
	if err != nil {
		jsonError(w, "failed to parse compose file: "+err.Error(), http.StatusBadRequest)
		return
	}

	volumes, volResponses := parseComposeVolumes(result)

	gs.LastSyncedCommit = commitHash

	project := config.Project{
		Name:           req.ProjectName,
		DeploymentMode: req.DeploymentMode,
		ComposeContent: content,
		Volumes:        volumes,
		Source:         "git",
		GitSource:      &gs,
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

func (s *Server) handleGitSync(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	project, ok := s.manifest.GetProject(id)
	if !ok {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	if project.Source != "git" || project.GitSource == nil {
		http.Error(w, `{"error":"project is not a git source"}`, http.StatusBadRequest)
		return
	}

	content, commitHash, err := gitimport.Sync(s.configDir, *project.GitSource)
	if err != nil {
		jsonError(w, "git sync failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if compose content actually changed.
	if content == project.ComposeContent {
		// No changes — just update the commit hash.
		gs := *project.GitSource
		gs.LastSyncedCommit = commitHash
		s.manifest.SetProjectGitSource(id, &gs)
		s.persistConfig()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(gitSyncResponse{
			Changed: false,
			Commit:  commitHash,
		})
		return
	}

	// Compose changed — run diff logic (same as handleUpdateCompose).
	result, err := compose.Parse(content)
	if err != nil {
		jsonError(w, "failed to parse updated compose file: "+err.Error(), http.StatusBadRequest)
		return
	}

	finalVolumes, diff := diffComposeVolumes(project.Volumes, result)

	s.manifest.UpdateProjectCompose(id, content, finalVolumes)

	gs := *project.GitSource
	gs.LastSyncedCommit = commitHash
	s.manifest.SetProjectGitSource(id, &gs)
	s.persistConfig()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gitSyncResponse{
		Changed: true,
		Diff:    &diff,
		Commit:  commitHash,
	})
}

type gitBrowseRequest struct {
	RepoURL       string `json:"repo_url"`
	Branch        string `json:"branch"`
	AuthToken     string `json:"auth_token,omitempty"`
	SSHPrivateKey string `json:"ssh_private_key,omitempty"`
	SavedSourceID string `json:"saved_source_id,omitempty"`
}

type gitBrowseResponse struct {
	Files []string `json:"files"`
}

func (s *Server) handleGitBrowse(w http.ResponseWriter, r *http.Request) {
	var req gitBrowseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if req.RepoURL == "" && req.SavedSourceID == "" {
		http.Error(w, `{"error":"repo_url is required"}`, http.StatusBadRequest)
		return
	}

	gs := gitimport.GitSource{
		RepoURL:       req.RepoURL,
		Branch:        req.Branch,
		AuthToken:     req.AuthToken,
		SSHPrivateKey: req.SSHPrivateKey,
	}

	if req.SavedSourceID != "" {
		saved, ok := s.manifest.GetSavedSource(req.SavedSourceID)
		if ok && saved.GitConfig != nil {
			if gs.RepoURL == "" {
				gs.RepoURL = saved.GitConfig.RepoURL
			}
			if gs.Branch == "" {
				gs.Branch = saved.GitConfig.Branch
			}
			mergeGitConfigs(&gs, saved.GitConfig)
		}
	}

	if gs.RepoURL == "" {
		http.Error(w, `{"error":"repo_url is required"}`, http.StatusBadRequest)
		return
	}
	if gs.Branch == "" {
		gs.Branch = "main"
	}

	files, err := gitimport.Browse(s.configDir, gs)
	if err != nil {
		jsonError(w, "git browse failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	if files == nil {
		files = []string{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gitBrowseResponse{
		Files: files,
	})
}

