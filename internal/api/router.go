package api

import (
	"embed"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/bocklucas/dvb-made-easy/internal/config"
	"github.com/bocklucas/dvb-made-easy/internal/restore"
	"github.com/bocklucas/dvb-made-easy/internal/sse"
)

type Server struct {
	manifest     *config.Manifest
	key          []byte
	configDir    string
	stagingDir   string
	broadcaster  *sse.Broadcaster
	dockerClient restore.DockerClient
}

func (s *Server) SetDockerClient(dc restore.DockerClient) {
	s.dockerClient = dc
}

//go:embed all:build
var frontendFS embed.FS

func NewRouter(manifest *config.Manifest, key []byte, configDir string, stagingDir string, dockerClient restore.DockerClient) http.Handler {
	s := &Server{
		manifest:     manifest,
		key:          key,
		configDir:    configDir,
		stagingDir:   stagingDir,
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
	r.Post("/api/git/browse", s.handleGitBrowse)
	r.Post("/api/projects/import-portainer", s.handleImportPortainer)
	r.Post("/api/projects/infer", s.handleInfer)
	r.Post("/api/backup-creator/parse", s.handleBCParse)
	r.Post("/api/backup-creator/git-fetch", s.handleBCGitFetch)
	r.Post("/api/backup-creator/portainer-fetch", s.handleBCPortainerFetch)
	r.Post("/api/backup-creator/generate", s.handleBCGenerate)
	r.Post("/api/projects", s.handleCreateProject)
	r.Get("/api/projects", s.handleListProjects)
	r.Get("/api/projects/{id}", s.handleGetProject)
	r.Delete("/api/projects/{id}", s.handleDeleteProject)
	r.Put("/api/projects/{id}", s.handleUpdateProject)

	r.Post("/api/projects/{id}/credentials", s.handleSaveCredentials)
	r.Put("/api/projects/{id}/credentials", s.handleUpdateCredentials)
	r.Get("/api/projects/{id}/credentials", s.handleGetCredentials)
	r.Delete("/api/projects/{id}/credentials", s.handleDeleteCredentials)

	r.Put("/api/projects/{id}/compose", s.handleUpdateCompose)
	r.Post("/api/projects/{id}/compose/confirm-removal", s.handleConfirmVolumeRemoval)
	r.Post("/api/projects/{id}/git-sync", s.handleGitSync)

	r.Get("/api/projects/{id}/backup-timestamps", s.handleBackupTimestamps)
	r.Post("/api/projects/{id}/compose-restore", s.handleComposeRestore)

	r.Get("/api/projects/{id}/volumes/check-exists", s.handleCheckVolumesExist)
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
	r.Get("/api/sources/{id}", s.handleGetSavedSource)
	r.Put("/api/sources/{id}", s.handleUpdateSavedSource)
	r.Delete("/api/sources/{id}", s.handleDeleteSavedSource)
	r.Post("/api/sources/{id}/test", s.handleTestSavedSource)

	r.Post("/api/backends", s.handleCreateSavedBackend)
	r.Get("/api/backends", s.handleListSavedBackends)
	r.Get("/api/backends/{id}", s.handleGetSavedBackend)
	r.Put("/api/backends/{id}", s.handleUpdateSavedBackend)
	r.Delete("/api/backends/{id}", s.handleDeleteSavedBackend)
	r.Post("/api/backends/{id}/test", s.handleTestSavedBackend)

	// Serve static files from embedded FS
	subFS, err := fs.Sub(frontendFS, "build")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(subFS))

	r.Handle("/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if strings.HasPrefix(path, "/api") {
			jsonError(w, "Not Found", http.StatusNotFound)
			return
		}

		cleaned := filepath.Clean(path)
		fsPath := strings.TrimPrefix(cleaned, "/")
		if fsPath == "" {
			fsPath = "index.html"
		}

		// Check if file exists in our subFS
		_, err := subFS.Open(fsPath)
		if err != nil {
			// Serve index.html for SPA routing fallback
			indexFile, err := subFS.Open("index.html")
			if err != nil {
				http.Error(w, "Index not found", http.StatusInternalServerError)
				return
			}
			defer indexFile.Close()

			content, err := io.ReadAll(indexFile)
			if err != nil {
				http.Error(w, "Read error", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(content)
			return
		}

		fileServer.ServeHTTP(w, r)
	}))

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
