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
	RepoURL       string `json:"repo_url"`
	Branch        string `json:"branch"`
	FilePath      string `json:"file_path"`
	AuthToken     string `json:"auth_token,omitempty"`
	SSHPrivateKey string `json:"ssh_private_key,omitempty"`
	ProjectName   string `json:"project_name"`
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

	gs.LastSyncedCommit = commitHash

	project := config.Project{
		Name:           req.ProjectName,
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

	existing := make(map[string]config.Volume, len(project.Volumes))
	for _, v := range project.Volumes {
		existing[v.Name] = v
	}

	patternMap := make(map[string]string)
	for _, job := range result.BackupJobs {
		patternMap[job.SourceVolume] = job.FilenameFormat
	}

	seen := make(map[string]bool)
	var newVolumeNames []string
	newVolumeMeta := make(map[string]compose.VolumeMapping)
	for _, vm := range result.Volumes {
		if seen[vm.Name] {
			continue
		}
		seen[vm.Name] = true
		newVolumeNames = append(newVolumeNames, vm.Name)
		newVolumeMeta[vm.Name] = vm
	}

	newSet := make(map[string]bool, len(newVolumeNames))
	for _, n := range newVolumeNames {
		newSet[n] = true
	}

	var added, unchanged []volumeResponse
	var finalVolumes []config.Volume

	for _, name := range newVolumeNames {
		vm := newVolumeMeta[name]
		if prev, exists := existing[name]; exists {
			unchanged = append(unchanged, volumeResponse{
				Name:             name,
				ComposeService:   vm.Service,
				ComposeMountPath: vm.MountPath,
				BackupPattern:    prev.BackupPattern,
			})
			finalVolumes = append(finalVolumes, config.Volume{
				Name:             name,
				ComposeService:   vm.Service,
				ComposeMountPath: vm.MountPath,
				BackupPattern:    prev.BackupPattern,
				Passphrase:       prev.Passphrase,
			})
		} else {
			pattern := patternMap[name]
			added = append(added, volumeResponse{
				Name:             name,
				ComposeService:   vm.Service,
				ComposeMountPath: vm.MountPath,
				BackupPattern:    pattern,
			})
			finalVolumes = append(finalVolumes, config.Volume{
				Name:             name,
				ComposeService:   vm.Service,
				ComposeMountPath: vm.MountPath,
				BackupPattern:    pattern,
			})
		}
	}

	var removed []volumeResponse
	for _, prev := range project.Volumes {
		if !newSet[prev.Name] {
			removed = append(removed, volumeResponse{
				Name:             prev.Name,
				ComposeService:   prev.ComposeService,
				ComposeMountPath: prev.ComposeMountPath,
				BackupPattern:    prev.BackupPattern,
			})
			finalVolumes = append(finalVolumes, prev)
		}
	}

	s.manifest.UpdateProjectCompose(id, content, finalVolumes)

	gs := *project.GitSource
	gs.LastSyncedCommit = commitHash
	s.manifest.SetProjectGitSource(id, &gs)
	s.persistConfig()

	diff := composeDiffResponse{
		Added:     added,
		Removed:   removed,
		Unchanged: unchanged,
	}
	if diff.Added == nil {
		diff.Added = []volumeResponse{}
	}
	if diff.Removed == nil {
		diff.Removed = []volumeResponse{}
	}
	if diff.Unchanged == nil {
		diff.Unchanged = []volumeResponse{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gitSyncResponse{
		Changed: true,
		Diff:    &diff,
		Commit:  commitHash,
	})
}
