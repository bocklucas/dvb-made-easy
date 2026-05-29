package config

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/bocklucas/dvb-made-easy/internal/encrypt"
	"github.com/bocklucas/dvb-made-easy/internal/gitimport"
	"github.com/bocklucas/dvb-made-easy/internal/portainer"
	"github.com/bocklucas/dvb-made-easy/internal/storage"
)

const manifestFileName = "manifest.enc"

type Manifest struct {
	mu             sync.RWMutex
	InstallationID string         `json:"installation_id"`
	Projects       []Project      `json:"projects"`
	SavedSources   []SavedSource  `json:"saved_sources,omitempty"`
	SavedBackends  []SavedBackend `json:"saved_backends,omitempty"`
}

type SavedBackend struct {
	ID             string              `json:"id"`
	Name           string              `json:"name"`
	Credentials    storage.Credentials `json:"credentials"`
	HasCredentials bool                `json:"has_credentials,omitempty"`
}

type SavedSource struct {
	ID              string                     `json:"id"`
	Name            string                     `json:"name"`
	Type            string                     `json:"type"`
	GitConfig       *gitimport.GitSource       `json:"git_config,omitempty"`
	PortainerConfig *portainer.PortainerSource `json:"portainer_config,omitempty"`
	HasCredentials  bool                       `json:"has_credentials,omitempty"`
}

type Project struct {
	ID                string                `json:"id"`
	Name              string                `json:"name"`
	Source            string                `json:"source,omitempty"`
	DeploymentMode    string                `json:"deployment_mode,omitempty"`
	SwarmName         string                `json:"swarm_name,omitempty"`
	ComposeContent    string                `json:"compose_content"`
	ComposeHash       string                `json:"compose_hash"`
	AddedAt           time.Time             `json:"added_at"`
	LastRefreshed     time.Time             `json:"last_refreshed"`
	Volumes           []Volume              `json:"volumes"`
	Credentials       *storage.Credentials         `json:"credentials,omitempty"`
	DefaultPassphrase string                       `json:"default_passphrase,omitempty"`
	GitSource         *gitimport.GitSource         `json:"git_source,omitempty"`
	PortainerSource   *portainer.PortainerSource   `json:"portainer_source,omitempty"`
}

type Volume struct {
	Name             string `json:"name"`
	ComposeService   string `json:"compose_service"`
	ComposeMountPath string `json:"compose_mount_path"`
	BackupPattern    string `json:"backup_pattern,omitempty"`
	Passphrase       string `json:"passphrase,omitempty"`
}

func NewManifest() *Manifest {
	return &Manifest{
		InstallationID: uuid.New().String(),
	}
}

func Load(configDir string, key []byte) (*Manifest, error) {
	path := filepath.Join(configDir, manifestFileName)
	ciphertext, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return NewManifest(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	plaintext, err := encrypt.Decrypt(key, ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decrypt manifest: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(plaintext, &m); err != nil {
		return nil, fmt.Errorf("unmarshal manifest: %w", err)
	}

	return &m, nil
}

func Save(configDir string, key []byte, m *Manifest) error {
	m.mu.RLock()
	data, err := json.Marshal(m)
	m.mu.RUnlock()
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	ciphertext, err := encrypt.Encrypt(key, data)
	if err != nil {
		return fmt.Errorf("encrypt manifest: %w", err)
	}

	path := filepath.Join(configDir, manifestFileName)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	return os.WriteFile(path, ciphertext, 0600)
}

func (m *Manifest) AddProject(p Project) Project {
	m.mu.Lock()
	defer m.mu.Unlock()

	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	if p.AddedAt.IsZero() {
		p.AddedAt = now
	}
	p.LastRefreshed = now
	p.ComposeHash = composeHash(p.ComposeContent)
	m.Projects = append(m.Projects, p)
	return p
}

func (m *Manifest) ListProjects() []Project {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]Project, len(m.Projects))
	copy(out, m.Projects)
	return out
}

func (m *Manifest) GetProject(id string) (Project, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, p := range m.Projects {
		if p.ID == id {
			return p, true
		}
	}
	return Project{}, false
}

func (m *Manifest) RemoveProject(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, p := range m.Projects {
		if p.ID == id {
			m.Projects = append(m.Projects[:i], m.Projects[i+1:]...)
			return true
		}
	}
	return false
}

func (m *Manifest) UpdateProjectVolumes(id string, volumes []Volume) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, p := range m.Projects {
		if p.ID == id {
			m.Projects[i].Volumes = volumes
			m.Projects[i].LastRefreshed = time.Now().UTC()
			return true
		}
	}
	return false
}

func (m *Manifest) UpdateProjectCompose(id, composeContent string, volumes []Volume) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, p := range m.Projects {
		if p.ID == id {
			m.Projects[i].ComposeContent = composeContent
			m.Projects[i].ComposeHash = composeHash(composeContent)
			m.Projects[i].Volumes = volumes
			m.Projects[i].LastRefreshed = time.Now().UTC()
			return true
		}
	}
	return false
}

func (m *Manifest) SetProjectName(id, name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, p := range m.Projects {
		if p.ID == id {
			m.Projects[i].Name = name
			return true
		}
	}
	return false
}

func (m *Manifest) SetProjectSwarmName(id, swarmName string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, p := range m.Projects {
		if p.ID == id {
			m.Projects[i].SwarmName = swarmName
			return true
		}
	}
	return false
}

func (p Project) StackName() string {
	if p.SwarmName != "" {
		return p.SwarmName
	}
	return p.Name
}

func (m *Manifest) SetProjectDeploymentMode(id, mode string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, p := range m.Projects {
		if p.ID == id {
			m.Projects[i].DeploymentMode = mode
			return true
		}
	}
	return false
}

func (m *Manifest) SetProjectCredentials(id string, creds *storage.Credentials) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, p := range m.Projects {
		if p.ID == id {
			m.Projects[i].Credentials = creds
			return true
		}
	}
	return false
}

func (m *Manifest) ClearProjectCredentials(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, p := range m.Projects {
		if p.ID == id {
			m.Projects[i].Credentials = nil
			return true
		}
	}
	return false
}

func (m *Manifest) SetVolumePassphrase(projectID, volumeName, passphrase string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, p := range m.Projects {
		if p.ID == projectID {
			for j, v := range p.Volumes {
				if v.Name == volumeName {
					m.Projects[i].Volumes[j].Passphrase = passphrase
					return true
				}
			}
			return false
		}
	}
	return false
}

func (m *Manifest) ClearVolumePassphrase(projectID, volumeName string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, p := range m.Projects {
		if p.ID == projectID {
			for j, v := range p.Volumes {
				if v.Name == volumeName {
					m.Projects[i].Volumes[j].Passphrase = ""
					return true
				}
			}
			return false
		}
	}
	return false
}

func (m *Manifest) SetProjectDefaultPassphrase(projectID, passphrase string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, p := range m.Projects {
		if p.ID == projectID {
			m.Projects[i].DefaultPassphrase = passphrase
			return true
		}
	}
	return false
}

func (m *Manifest) ClearProjectDefaultPassphrase(projectID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, p := range m.Projects {
		if p.ID == projectID {
			m.Projects[i].DefaultPassphrase = ""
			return true
		}
	}
	return false
}

func (m *Manifest) SetProjectGitSource(id string, gs *gitimport.GitSource) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, p := range m.Projects {
		if p.ID == id {
			m.Projects[i].GitSource = gs
			return true
		}
	}
	return false
}

func (m *Manifest) SetProjectPortainerSource(id string, ps *portainer.PortainerSource) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, p := range m.Projects {
		if p.ID == id {
			m.Projects[i].PortainerSource = ps
			return true
		}
	}
	return false
}

func (m *Manifest) AddSavedSource(s SavedSource) SavedSource {
	m.mu.Lock()
	defer m.mu.Unlock()

	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	m.SavedSources = append(m.SavedSources, s)
	return s
}

func (m *Manifest) ListSavedSources() []SavedSource {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]SavedSource, len(m.SavedSources))
	copy(out, m.SavedSources)
	return out
}

func (m *Manifest) GetSavedSource(id string) (SavedSource, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, s := range m.SavedSources {
		if s.ID == id {
			return s, true
		}
	}
	return SavedSource{}, false
}

func (m *Manifest) RemoveSavedSource(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, s := range m.SavedSources {
		if s.ID == id {
			m.SavedSources = append(m.SavedSources[:i], m.SavedSources[i+1:]...)
			return true
		}
	}
	return false
}

func (m *Manifest) AddSavedBackend(b SavedBackend) SavedBackend {
	m.mu.Lock()
	defer m.mu.Unlock()

	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	m.SavedBackends = append(m.SavedBackends, b)
	return b
}

func (m *Manifest) ListSavedBackends() []SavedBackend {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]SavedBackend, len(m.SavedBackends))
	copy(out, m.SavedBackends)
	return out
}

func (m *Manifest) GetSavedBackend(id string) (SavedBackend, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, b := range m.SavedBackends {
		if b.ID == id {
			return b, true
		}
	}
	return SavedBackend{}, false
}

func (m *Manifest) RemoveSavedBackend(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, b := range m.SavedBackends {
		if b.ID == id {
			m.SavedBackends = append(m.SavedBackends[:i], m.SavedBackends[i+1:]...)
			return true
		}
	}
	return false
}

func (m *Manifest) UpdateSavedSource(id string, s SavedSource) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, existing := range m.SavedSources {
		if existing.ID == id {
			s.ID = id
			m.SavedSources[i] = s
			return true
		}
	}
	return false
}

func (m *Manifest) UpdateSavedBackend(id string, b SavedBackend) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, existing := range m.SavedBackends {
		if existing.ID == id {
			b.ID = id
			m.SavedBackends[i] = b
			return true
		}
	}
	return false
}

func composeHash(content string) string {
	h := sha256.Sum256([]byte(content))
	return fmt.Sprintf("sha256:%x", h)
}
