package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/offen/restore-manager/internal/config"
	"github.com/offen/restore-manager/internal/restore"
	"github.com/offen/restore-manager/internal/sse"
)

type Server struct {
	manifest     *config.Manifest
	key          []byte
	configDir    string
	broadcaster  *sse.Broadcaster
	dockerClient restore.DockerClient
}

func (s *Server) SetDockerClient(dc restore.DockerClient) {
	s.dockerClient = dc
}

func NewRouter(manifest *config.Manifest, key []byte, configDir string, dockerClient restore.DockerClient) http.Handler {
	s := &Server{
		manifest:     manifest,
		key:          key,
		configDir:    configDir,
		broadcaster:  sse.NewBroadcaster(),
		dockerClient: dockerClient,
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)

	r.Get("/api/health", s.handleHealth)

	r.Post("/api/projects/import", s.handleImportCompose)
	r.Post("/api/projects/import-git", s.handleImportGit)
	r.Post("/api/projects/import-portainer", s.handleImportPortainer)
	r.Post("/api/projects/infer", s.handleInfer)
	r.Post("/api/projects", s.handleCreateProject)
	r.Get("/api/projects", s.handleListProjects)
	r.Get("/api/projects/{id}", s.handleGetProject)
	r.Delete("/api/projects/{id}", s.handleDeleteProject)

	r.Post("/api/projects/{id}/credentials", s.handleSaveCredentials)
	r.Put("/api/projects/{id}/credentials", s.handleUpdateCredentials)
	r.Get("/api/projects/{id}/credentials", s.handleGetCredentials)
	r.Delete("/api/projects/{id}/credentials", s.handleDeleteCredentials)

	r.Put("/api/projects/{id}/compose", s.handleUpdateCompose)
	r.Post("/api/projects/{id}/compose/confirm-removal", s.handleConfirmVolumeRemoval)
	r.Post("/api/projects/{id}/git-sync", s.handleGitSync)

	r.Get("/api/projects/{id}/backup-timestamps", s.handleBackupTimestamps)
	r.Post("/api/projects/{id}/compose-restore", s.handleComposeRestore)

	r.Get("/api/projects/{id}/volumes/{vol}/backups", s.handleListBackups)

	r.Post("/api/projects/{id}/volumes/{vol}/restore", s.handleRestore)

	r.Put("/api/projects/{id}/volumes/{vol}/passphrase", s.handleSetVolumePassphrase)
	r.Delete("/api/projects/{id}/volumes/{vol}/passphrase", s.handleDeleteVolumePassphrase)

	r.Put("/api/projects/{id}/passphrase", s.handleSetProjectPassphrase)
	r.Delete("/api/projects/{id}/passphrase", s.handleDeleteProjectPassphrase)

	r.Get("/api/restore/{token}/stream", s.handleRestoreStream)

	r.Post("/api/portainer/connect", s.handlePortainerConnect)
	r.Post("/api/portainer/stacks", s.handlePortainerStacks)

	r.Post("/api/sources", s.handleCreateSavedSource)
	r.Get("/api/sources", s.handleListSavedSources)
	r.Delete("/api/sources/{id}", s.handleDeleteSavedSource)

	r.Post("/api/backends", s.handleCreateSavedBackend)
	r.Get("/api/backends", s.handleListSavedBackends)
	r.Delete("/api/backends/{id}", s.handleDeleteSavedBackend)

	return r
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
