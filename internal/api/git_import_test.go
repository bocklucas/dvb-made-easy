package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
)

const testCompose = `services:
  db:
    image: postgres:16
    volumes:
      - pgdata:/var/lib/postgresql/data
volumes:
  pgdata:
`

const testCompose2 = `services:
  db:
    image: postgres:16
    volumes:
      - pgdata:/var/lib/postgresql/data
  cache:
    image: redis:7
    volumes:
      - redis-data:/data
volumes:
  pgdata:
  redis-data:
`

// createBareRepo creates a local bare git repo with a compose file committed,
// and returns the file:// URL to use as repo_url.
func createBareRepo(t *testing.T, composeContent string) string {
	t.Helper()

	// Create a working repo, commit a file, then clone to bare.
	workDir := t.TempDir()
	repo, err := git.PlainInit(workDir, false)
	if err != nil {
		t.Fatalf("init working repo: %v", err)
	}

	// Write compose file.
	composePath := filepath.Join(workDir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		t.Fatalf("write compose: %v", err)
	}

	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}
	if _, err := w.Add("docker-compose.yml"); err != nil {
		t.Fatalf("add: %v", err)
	}

	_, err = w.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@test.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("commit: %v", err)
	}

	// Clone to bare repo for serving.
	bareDir := t.TempDir()
	_, err = git.PlainClone(bareDir, true, &git.CloneOptions{
		URL: workDir,
	})
	if err != nil {
		t.Fatalf("clone to bare: %v", err)
	}

	return bareDir
}

// pushUpdate adds a new commit to the working repo and pushes to the bare repo.
func pushUpdate(t *testing.T, bareDir string, composeContent string) {
	t.Helper()

	// Clone bare to a working copy, update, commit, push.
	workDir := t.TempDir()
	repo, err := git.PlainClone(workDir, false, &git.CloneOptions{
		URL: bareDir,
	})
	if err != nil {
		t.Fatalf("clone from bare: %v", err)
	}

	composePath := filepath.Join(workDir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		t.Fatalf("write compose: %v", err)
	}

	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}
	if _, err := w.Add("docker-compose.yml"); err != nil {
		t.Fatalf("add: %v", err)
	}

	_, err = w.Commit("update compose", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@test.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("commit: %v", err)
	}

	err = repo.Push(&git.PushOptions{
		RemoteName: "origin",
		RefSpecs:   []gitconfig.RefSpec{"refs/heads/master:refs/heads/master"},
	})
	if err != nil {
		t.Fatalf("push: %v", err)
	}
}

func TestImportGit(t *testing.T) {
	bareDir := createBareRepo(t, testCompose)
	srv, _, _ := setupTestServer(t)

	body, _ := json.Marshal(map[string]string{
		"repo_url":     bareDir,
		"branch":       "master",
		"file_path":    "docker-compose.yml",
		"project_name": "git-test",
	})

	resp, err := http.Post(srv.URL+"/api/projects/import-git", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]string
		json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("status: got %d, want 200, error: %v", resp.StatusCode, errBody)
	}

	var result struct {
		ID      string `json:"id"`
		Volumes []struct {
			Name             string `json:"name"`
			ComposeService   string `json:"compose_service"`
			ComposeMountPath string `json:"compose_mount_path"`
		} `json:"volumes"`
		DriftHash string `json:"drift_hash"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if result.ID == "" {
		t.Fatal("import must return an ID")
	}
	if len(result.Volumes) != 1 {
		t.Fatalf("volumes: got %d, want 1", len(result.Volumes))
	}
	if result.Volumes[0].Name != "pgdata" {
		t.Fatalf("volume name: got %q, want %q", result.Volumes[0].Name, "pgdata")
	}
	if result.DriftHash == "" {
		t.Fatal("drift hash must be set")
	}

	// Verify the project was created with Source == "git" and GitSource set.
	getResp, err := http.Get(srv.URL + "/api/projects/" + result.ID)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	defer getResp.Body.Close()

	var project struct {
		Source    string `json:"source"`
		Name     string `json:"name"`
		GitSource *struct {
			RepoURL          string `json:"repo_url"`
			Branch           string `json:"branch"`
			FilePath         string `json:"file_path"`
			LastSyncedCommit string `json:"last_synced_commit"`
		} `json:"git_source"`
	}
	if err := json.NewDecoder(getResp.Body).Decode(&project); err != nil {
		t.Fatalf("decode project: %v", err)
	}
	if project.Source != "git" {
		t.Fatalf("source: got %q, want %q", project.Source, "git")
	}
	if project.Name != "git-test" {
		t.Fatalf("name: got %q, want %q", project.Name, "git-test")
	}
	if project.GitSource == nil {
		t.Fatal("git_source must be set")
	}
	if project.GitSource.LastSyncedCommit == "" {
		t.Fatal("last_synced_commit must be set after import")
	}
}

func TestGitSync_NoChanges(t *testing.T) {
	bareDir := createBareRepo(t, testCompose)
	srv, _, _ := setupTestServer(t)

	// Import first.
	body, _ := json.Marshal(map[string]string{
		"repo_url":     bareDir,
		"branch":       "master",
		"file_path":    "docker-compose.yml",
		"project_name": "sync-test",
	})
	importResp, err := http.Post(srv.URL+"/api/projects/import-git", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	var importResult struct {
		ID string `json:"id"`
	}
	json.NewDecoder(importResp.Body).Decode(&importResult)
	importResp.Body.Close()

	// Sync — no changes expected.
	syncResp, err := http.Post(srv.URL+"/api/projects/"+importResult.ID+"/git-sync", "application/json", bytes.NewReader([]byte("{}")))
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	defer syncResp.Body.Close()

	if syncResp.StatusCode != http.StatusOK {
		var errBody map[string]string
		json.NewDecoder(syncResp.Body).Decode(&errBody)
		t.Fatalf("status: got %d, want 200, error: %v", syncResp.StatusCode, errBody)
	}

	var syncResult struct {
		Changed bool   `json:"changed"`
		Commit  string `json:"commit"`
	}
	json.NewDecoder(syncResp.Body).Decode(&syncResult)

	if syncResult.Changed {
		t.Fatal("sync should report no changes")
	}
	if syncResult.Commit == "" {
		t.Fatal("commit must be set")
	}
}

func TestGitSync_WithChanges(t *testing.T) {
	bareDir := createBareRepo(t, testCompose)
	srv, _, _ := setupTestServer(t)

	// Import.
	body, _ := json.Marshal(map[string]string{
		"repo_url":     bareDir,
		"branch":       "master",
		"file_path":    "docker-compose.yml",
		"project_name": "sync-changes-test",
	})
	importResp, err := http.Post(srv.URL+"/api/projects/import-git", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	var importResult struct {
		ID string `json:"id"`
	}
	json.NewDecoder(importResp.Body).Decode(&importResult)
	importResp.Body.Close()

	// Push a new commit with a changed compose file.
	pushUpdate(t, bareDir, testCompose2)

	// Sync — changes expected.
	syncResp, err := http.Post(srv.URL+"/api/projects/"+importResult.ID+"/git-sync", "application/json", bytes.NewReader([]byte("{}")))
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	defer syncResp.Body.Close()

	if syncResp.StatusCode != http.StatusOK {
		var errBody map[string]string
		json.NewDecoder(syncResp.Body).Decode(&errBody)
		t.Fatalf("status: got %d, want 200, error: %v", syncResp.StatusCode, errBody)
	}

	var syncResult struct {
		Changed bool `json:"changed"`
		Diff    *struct {
			Added []struct {
				Name string `json:"name"`
			} `json:"added"`
			Removed []struct {
				Name string `json:"name"`
			} `json:"removed"`
			Unchanged []struct {
				Name string `json:"name"`
			} `json:"unchanged"`
		} `json:"diff"`
		Commit string `json:"commit"`
	}
	json.NewDecoder(syncResp.Body).Decode(&syncResult)

	if !syncResult.Changed {
		t.Fatal("sync should report changes")
	}
	if syncResult.Diff == nil {
		t.Fatal("diff must be set when changed")
	}
	if len(syncResult.Diff.Added) != 1 {
		t.Fatalf("added: got %d, want 1", len(syncResult.Diff.Added))
	}
	if syncResult.Diff.Added[0].Name != "redis-data" {
		t.Fatalf("added volume: got %q, want %q", syncResult.Diff.Added[0].Name, "redis-data")
	}
	if len(syncResult.Diff.Unchanged) != 1 {
		t.Fatalf("unchanged: got %d, want 1", len(syncResult.Diff.Unchanged))
	}
	if syncResult.Commit == "" {
		t.Fatal("commit must be set")
	}
}

func TestGitSync_NotGitSource(t *testing.T) {
	srv, m, _ := setupTestServer(t)

	// Create a paste-based project.
	body, _ := json.Marshal(map[string]string{
		"compose_content": testCompose,
	})
	importResp, err := http.Post(srv.URL+"/api/projects/import", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	var importResult struct {
		ID string `json:"id"`
	}
	json.NewDecoder(importResp.Body).Decode(&importResult)
	importResp.Body.Close()

	_ = m // manifest used for setup only

	// Try to sync a non-git project.
	syncResp, err := http.Post(srv.URL+"/api/projects/"+importResult.ID+"/git-sync", "application/json", bytes.NewReader([]byte("{}")))
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	defer syncResp.Body.Close()

	if syncResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", syncResp.StatusCode)
	}
}

func TestImportGit_MissingRepoURL(t *testing.T) {
	srv, _, _ := setupTestServer(t)

	body, _ := json.Marshal(map[string]string{
		"branch":       "main",
		"file_path":    "docker-compose.yml",
		"project_name": "test",
	})
	resp, err := http.Post(srv.URL+"/api/projects/import-git", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", resp.StatusCode)
	}
}

func TestImportGit_MissingProjectName(t *testing.T) {
	srv, _, _ := setupTestServer(t)

	body, _ := json.Marshal(map[string]string{
		"repo_url":  "file:///some/path",
		"branch":    "main",
		"file_path": "docker-compose.yml",
	})
	resp, err := http.Post(srv.URL+"/api/projects/import-git", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", resp.StatusCode)
	}
}
