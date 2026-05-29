package gitimport

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

// GitSource describes a compose file inside a Git repository.
type GitSource struct {
	RepoURL          string `json:"repo_url"`
	Branch           string `json:"branch"`
	FilePath         string `json:"file_path"`
	LastSyncedCommit string `json:"last_synced_commit,omitempty"`
	AuthToken        string `json:"auth_token,omitempty"`
	SSHPrivateKey    string `json:"ssh_private_key,omitempty"`
	HasAuthToken     bool   `json:"has_auth_token,omitempty"`
	HasSSHKey        bool   `json:"has_ssh_key,omitempty"`
}

// ensureRepo ensures that the repository is cloned locally and updated.
func ensureRepo(cacheDir string, source GitSource) (*git.Repository, error) {
	repoDir := repoPath(cacheDir, source.RepoURL)

	auth, err := authMethod(source)
	if err != nil {
		return nil, fmt.Errorf("auth: %w", err)
	}

	refName := plumbing.NewBranchReferenceName(source.Branch)

	repo, err := git.PlainClone(repoDir, true, &git.CloneOptions{
		URL:           source.RepoURL,
		Auth:          auth,
		ReferenceName: refName,
		SingleBranch:  true,
		NoCheckout:    true,
	})
	if err == git.ErrRepositoryAlreadyExists {
		repo, err = git.PlainOpen(repoDir)
		if err != nil {
			return nil, fmt.Errorf("open: %w", err)
		}
		err = repo.Fetch(&git.FetchOptions{
			Auth: auth,
			RefSpecs: []config.RefSpec{
				config.RefSpec(fmt.Sprintf("+refs/heads/%s:refs/heads/%s", source.Branch, source.Branch)),
			},
			Force: true,
		})
		if err != nil && err != git.NoErrAlreadyUpToDate {
			return nil, fmt.Errorf("fetch: %w", err)
		}
		return repo, nil
	}
	if err != nil {
		return nil, fmt.Errorf("clone: %w", err)
	}

	return repo, nil
}

// Clone clones a repo (bare) and reads the compose file at the given path.
// Returns compose content and commit hash.
func Clone(cacheDir string, source GitSource) (content string, commitHash string, err error) {
	repo, err := ensureRepo(cacheDir, source)
	if err != nil {
		return "", "", err
	}
	refName := plumbing.NewBranchReferenceName(source.Branch)
	return readFile(repo, refName, source.FilePath)
}

// Sync fetches the latest from the repo and reads the compose file.
// Returns compose content and commit hash.
func Sync(cacheDir string, source GitSource) (content string, commitHash string, err error) {
	return Clone(cacheDir, source)
}

// readFile reads a file from the HEAD of the given branch reference.
func readFile(repo *git.Repository, refName plumbing.ReferenceName, filePath string) (string, string, error) {
	ref, err := repo.Reference(refName, true)
	if err != nil {
		return "", "", fmt.Errorf("resolve ref %s: %w", refName, err)
	}

	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return "", "", fmt.Errorf("commit: %w", err)
	}

	tree, err := commit.Tree()
	if err != nil {
		return "", "", fmt.Errorf("tree: %w", err)
	}

	file, err := tree.File(filePath)
	if err != nil {
		if err == object.ErrFileNotFound {
			return "", "", fmt.Errorf("file %q not found in branch", filePath)
		}
		return "", "", fmt.Errorf("file: %w", err)
	}

	reader, err := file.Reader()
	if err != nil {
		return "", "", fmt.Errorf("read file: %w", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return "", "", fmt.Errorf("read all: %w", err)
	}

	return string(data), commit.Hash.String(), nil
}

// repoPath returns a deterministic local path for caching a bare repo.
func repoPath(cacheDir, repoURL string) string {
	h := sha256.Sum256([]byte(repoURL))
	return filepath.Join(cacheDir, "repos", fmt.Sprintf("%x", h))
}

// authMethod builds a transport.AuthMethod from the GitSource fields.
func authMethod(source GitSource) (transport.AuthMethod, error) {
	if source.SSHPrivateKey != "" {
		keys, err := ssh.NewPublicKeys("git", []byte(source.SSHPrivateKey), "")
		if err != nil {
			return nil, fmt.Errorf("ssh key: %w", err)
		}
		return keys, nil
	}
	if source.AuthToken != "" {
		return &http.BasicAuth{
			Username: "x-access-token",
			Password: source.AuthToken,
		}, nil
	}
	return nil, nil
}

// Browse clones or fetches a repo (bare) and lists all potential compose files.
func Browse(cacheDir string, source GitSource) ([]string, error) {
	repo, err := ensureRepo(cacheDir, source)
	if err != nil {
		return nil, err
	}

	refName := plumbing.NewBranchReferenceName(source.Branch)
	ref, err := repo.Reference(refName, true)
	if err != nil {
		return nil, fmt.Errorf("resolve ref %s: %w", refName, err)
	}

	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("tree: %w", err)
	}

	var files []string
	fileIter := tree.Files()
	err = fileIter.ForEach(func(f *object.File) error {
		lower := strings.ToLower(f.Name)
		if strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".yaml") || strings.Contains(filepath.Base(lower), "compose") {
			files = append(files, f.Name)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("iterate files: %w", err)
	}

	return files, nil
}

// TestConnection verifies that the remote repository is reachable with the
// configured credentials by listing remote references.
func TestConnection(source GitSource) error {
	auth, err := authMethod(source)
	if err != nil {
		return fmt.Errorf("auth: %w", err)
	}

	remote := git.NewRemote(nil, &config.RemoteConfig{
		Name: "test",
		URLs: []string{source.RepoURL},
	})

	_, err = remote.List(&git.ListOptions{Auth: auth})
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	return nil
}

// CleanCache removes the cached bare repo for a given URL.
func CleanCache(cacheDir, repoURL string) error {
	return os.RemoveAll(repoPath(cacheDir, repoURL))
}
