# offen-restore-manager — Design Spec

> **Date:** 2026-05-17
> **Status:** Approved
> **Goal:** A lightweight web GUI tool that makes restoring Docker volume backups trivial — no manual tar commands, no container management, no guesswork.

---

## Problem Statement

Docker volume backup tools (notably `offen/docker-volume-backup`) have excellent backup functionality — scheduled backups, multiple storage backends (S3, WebDAV, SSH, Azure, Dropbox, Google Drive), GPG encryption, retention policies, notifications.

But **restore functionality is non-existent**. Users must manually:
1. Stop consuming containers
2. Download the backup file
3. Extract it manually (`tar -xvf`)
4. Mount a temporary container with the target volume
5. Copy files into the volume
6. Figure out correct `--strip-components` for path layout
7. Restart containers

For database volumes (the most critical use case), the recommended approach — remove the existing volume, create a new one, restore — is even more error-prone.

Existing solutions like [VolumeVault](https://github.com/Darkdragon14/VolumeVault) add a web UI on top of offen but still require manual job creation and don't auto-discover volumes from docker-compose files.

## Solution

A lightweight Docker container with a web GUI that:
- Reads a docker-compose file → auto-discovers volumes, containers, dependencies
- Connects directly to offen's storage backends to browse backups (no manual download)
- Offers one-click restore that handles the entire stack

## Tech Stack

| Layer | Choice | Rationale |
|---|---|---|
| Backend | Go | Performant, single binary, excellent Docker ecosystem, net/http + embed is mature |
| Frontend | Svelte SPA | Minimal framework, compiled to static files, small bundle size |
| Build | Vite (Svelte) + Go embed | Dev: Vite HMR, Prod: embedded in Go binary |
| Deploy | Docker container | Single container, standard Docker socket mount pattern |
| Config Storage | Encrypted JSON on disk | Survives container restarts, user manages via volume mount |

## Architecture

### Container Layout

```
┌───────────────────────────────────────────────────────────────┐
│ offen-restore-manager (Docker Container)                       │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │  Go Binary (Single Process, port 8080)                     │ │
│  │                                                            │ │
│  │  ┌───────────────┐   ┌──────────────────────┐             │ │
│  │  │ REST API      │   │ Embedded SPA          │             │ │
│  │  │ (Chi router)  │   │ (Svelte → static      │             │ │
│  │  │               │   │   files via embed.FS) │             │ │
│  │  └───────┬───────┘   └──────────────────────┘             │ │
│  │          │                                                 │ │
│  │  ┌───────┴─────────────────────────────────────────────┐  │ │
│  │  │ Orchestrator Layer                                   │  │ │
│  │  │ • Docker SDK (stop/start/list/create volumes)        │  │ │
│  │  │ • Shell exec (download, decompress, decrypt)         │  │ │
│  │  │ • Storage backends (S3, WebDAV, SSH)                 │  │ │
│  │  │ • SSE progress broadcaster                           │  │ │
│  │  └─────────────────────────────────────────────────────┘  │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                 │
│  Volumes:                                                       │
│  • /app/config (persistent encrypted config)                    │
│  • /var/run/docker.sock (Docker API — read/write)               │
└───────────────────────────────────────────────────────────────┘
```

### Development Workflow

- **Backend:** `go run cmd/server/main.go` — API server on `:8080`
- **Frontend:** `npm run dev` (Vite) — SPA dev server on `:5173`
- CORS enabled for `localhost:5173` → `localhost:8080`
- **Production:** Single binary, Svelte compiled and embedded

## Components

### Go Backend Packages

| Package | Responsibility |
|---|---|
| `cmd/server` | Entry point, CLI flags, startup |
| `internal/api` | HTTP routes, request/response handlers, middleware |
| `internal/compose` | Parse docker-compose.yml, extract volumes, container dependencies, depends_on ordering |
| `internal/config` | Load/save encrypted manifest from/to disk |
| `internal/docker` | Docker SDK wrapper — container lifecycle, volume operations, compose integration |
| `internal/storage` | Storage backend abstraction with implementations: S3, WebDAV, SSH/SFTP |
| `internal/restore` | Restore orchestrator — download, decrypt, decompress, extract. Launches offen containers as helpers |
| `internal/sse` | Server-sent events stream for real-time progress updates |
| `internal/encrypt` | AES-GCM encryption for config storage |

### Svelte Frontend Pages

| Component | Purpose |
|---|---|
| `SetupWizard` | Multi-step wizard: paste compose → select backend → enter credentials → discover backups |
| `ProjectList` | Sidebar: list of projects, click to expand and see volumes |
| `VolumeDetail` | Main area: volumes for selected project with restore buttons |
| `BackupList` | Available backups for selected volume (fetched live from storage backend) |
| `RestoreFlow` | Restore mode selection → confirm → watch progress via SSE |
| `RestoreProgress` | Real-time progress display with step-by-step updates |

### State Management

Svelte stores (simple reactive state). No Redux or similar.

### API Client

Lightweight `fetch` wrapper in `web/lib/api.ts` with typed responses.

## Data Model

### Encrypted Manifest (on disk)

```json
{
  "installation_id": "uuid",
  "encryption_key_path": ".key",
  "projects": [
    {
      "id": "uuid",
      "name": "home-server",
      "compose_content": "...",
      "compose_hash": "sha256:...",
      "added_at": "2026-05-17T12:00:00Z",
      "last_refreshed": "2026-05-17T12:00:00Z",
      "volumes": [
        {
          "name": "postgres-data",
          "compose_service": "postgres",
          "compose_mount_path": "/var/lib/postgresql/data",
          "credentials": {
            "s3": {
              "bucket": "backups",
              "region": "us-east-1",
              "endpoint": "s3.amazonaws.com",
              "access_key_id_file": "",
              "secret_access_key_file": ""
            }
          },
          "backup_keys": [
            {
              "key": "backup-2026-05-15T04-00-00.tar.gz",
              "last_modified": "2026-05-15T04:00:05Z",
              "size_bytes": 104857600,
              "is_encrypted": false
            }
          ]
        }
      ]
    }
  ]
}
```

**Encryption:** Per-installation AES-GCM key stored in `config/.key`. Config file encrypted with this key. Key is separate from config to allow regeneration.

### API Response Types

```go
// Import result — returned after parsing a compose file
type ImportResult struct {
    ID        string      `json:"id"`
    Volumes   []VolumeInfo `json:"volumes"`
    DriftHash string      `json:"drift_hash,omitempty"`
}

type VolumeInfo struct {
    Name            string   `json:"name"`
    ComposeService  string   `json:"compose_service"`
    ComposeMountPath string  `json:"compose_mount_path"`
    IsEncrypted     bool     `json:"is_encrypted"`
    AvailableBackups []int   `json:"available_backend_indices"`
}

// Backup info — fetched live from storage backend
type BackupInfo struct {
    Key          string    `json:"key"`
    DisplayName  string    `json:"display_name"`
    Size         int64     `json:"size"`
    LastModified time.Time `json:"last_modified"`
    IsEncrypted  bool      `json:"is_encrypted"`
}

// SSE progress event
type ProgressEvent struct {
    Step    string `json:"step"`
    Status  string `json:"status"`
    Message string `json:"message"`
    Percent int    `json:"percent,omitempty"`
}
```

## API Endpoints

### Projects

| Method | Path | Description |
|---|---|---|
| POST | `/api/projects/import` | Parse compose file content, return discovered volumes |
| POST | `/api/projects` | Create a project from imported state |
| GET | `/api/projects` | List all projects |
| GET | `/api/projects/:id` | Get project details (volumes, drift status) |
| DELETE | `/api/projects/:id` | Remove a project |
| PUT | `/api/projects/:id/refresh` | Re-parse compose file, detect drift |

### Volumes

| Method | Path | Description |
|---|---|---|
| GET | `/api/projects/:id/volumes` | List volumes for a project |
| GET | `/api/projects/:id/volumes/:vol/backups` | List backups from configured storage backends |

### Credentials

| Method | Path | Description |
|---|---|---|
| POST | `/api/projects/:id/credentials` | Save credentials for a volume's storage backend |
| PUT | `/api/projects/:id/credentials` | Update credentials |
| DELETE | `/api/projects/:id/credentials` | Remove credentials |

### Restore

| Method | Path | Description |
|---|---|---|
| POST | `/api/projects/:id/volumes/:vol/restore` | Start a restore (`?mode=new_volume\|full_stack&backup_key=...&target_volume_name=...`) |
| GET | `/api/restore/:token/stream` | SSE endpoint for real-time progress |

### Request/Response Examples

**Import compose file:**
```
POST /api/projects/import
Content-Type: application/json

{
  "compose_content": "services:\n  postgres:\n    volumes:\n      - data:/var/lib/postgresql/data\n..."
}

Response 200:
{
  "id": "proj-uuid",
  "volumes": [
    {
      "name": "data",
      "compose_service": "postgres",
      "compose_mount_path": "/var/lib/postgresql/data",
      "is_encrypted": false,
      "available_backend_indices": []
    }
  ],
  "drift_hash": "sha256:abc123"
}
```

**Start restore (new volume):**
```
POST /api/projects/proj-uuid/volumes/data/restore?mode=new_volume&backup_key=backup-2026-05-15.tar.gz&target_volume_name=data_restored_20260517

Response 202:
{
  "restore_token": "rst-uuid",
  "status": "queued"
}
```

**SSE Progress:**
```
GET /api/restore/rst-uuid/stream

data: {"step":"scanning_backends","status":"in_progress","message":"Scanning storage backends..."}
data: {"step":"downloading","status":"in_progress","message":"Downloading backup (35%)","percent":35}
data: {"step":"extracting","status":"in_progress","message":"Extracting archive into volume..."}
data: {"step":"complete","status":"done","message":"Restore complete. Volume: data_restored_20260517"}
```

## Restore Flows

### New Volume Mode

1. User selects a backup and chooses "New Volume"
2. System generates target name: `{original_volume_name}_restored_{timestamp}` (collision-safe)
3. User confirms
4. Go creates a new Docker volume
5. Go launches a temporary offen container to download and extract the backup
6. Progress streamed via SSE
7. On success: user is shown the new volume name and instructions

### Full Stack Mode

1. User selects a backup and chooses "Full Stack Restore"
2. System shows summary: "This will stop 3 containers, restore 2 volumes, restart 3 containers"
3. User confirms
4. Go parses compose file for all containers and their depends_on ordering
5. Go stops all containers (respecting depends_on order, deepest deps first)
6. For each volume that has a backup:
   a. Creates new volume with name `{original}_restored_{timestamp}`
   b. Downloads backup from configured storage backend
   c. Extracts into new volume
   d. Streams progress via SSE
7. On success: user is shown a list of new volume names and instructions for updating their compose file

### Full Stack — Compose File Drift Handling

When a backup was taken with an old compose file and the current compose file differs:
- The restore UI flags the mismatch: "⚠️ Your compose file has changed since some backups were taken"
- The user can choose to proceed (data may not match current schema) or refresh the compose file
- The generated restore instructions clearly list which volumes have been restored to which new names

## Error Handling

| Error Type | Handling |
|---|---|
| Storage backend unreachable | Shown on backup list with retry button; SSE progress shows error |
| Auth failed (S3 key, WebDAV creds) | Shown during credential validation (wizard) and during restore |
| Target volume already exists | Shown during restore; user can choose overwrite (with confirmation) or new name |
| Docker daemon unreachable | Shown on every Docker operation; SSE shows error with suggested fix |
| GPG decryption fails | Shown during restore; user can try different passphrase |
| Compose parse error | Shown on import with line/column info |
| Restore failure mid-way | Partial rollbacks where safe (e.g., created volume cleaned up); SSE shows failure point and suggested fix |
| Container stop fails | Logged; SSE shows which containers failed and continues with remaining |

### SSE Progress Event Schema

```go
type ProgressEvent struct {
    Step    string `json:"step"`       // Human-readable step identifier
    Status  string `json:"status"`     // "in_progress" | "done" | "failed" | "error"
    Message string `json:"message"`    // Human-readable message
    Percent int    `json:"percent,omitempty"` // 0-100 for long-running steps
    Details string `json:"details,omitempty"` // Additional context (e.g., "Stopped container: postgres")
}
```

Step values: `scanning_backends`, `stopping_containers`, `creating_volume`, `downloading`, `extracting`, `starting_containers`, `complete`, `failed`, `error`

## Offen Archive Reverse-Engineering Plan

offen's backup format is straightforward:

1. **Naming pattern:** `backup-YYYY-MM-DDTHH-MM-SS.tar.gz` (customizable via `BACKUP_FILENAME`)
2. **Archive structure:** `tar.gz` containing a top-level directory named after the volume, then the actual data:
   ```
   backup-xxx.tar.gz/
     └── <volume-name>/
         └── <actual files>
   ```
3. **Restore command (via offen container):**
   ```bash
   docker run --rm \
     -v <source-volume-or-archive>:<source-path>:ro \
     -v <target-volume>:<target-path> \
     offen/docker-volume-backup:latest \
     tar -xzf <source-path>/backup.tar.gz -C <target-path> --strip-components 2
   ```
4. **GPG decryption:** Files with `.gpg` extension. Use `gpg --batch --passphrase <passphrase> -d <file.gpg> > <file.tar.gz>`
5. **Storage listing:** S3 `ListObjects`, WebDAV `PROPFIND`, SSH `List` — iterate and filter by offen's naming pattern

## Project File Structure

```
offen-restore-manager/
├── cmd/
│   └── server/
│       └── main.go              # Entry point, flag parsing, startup
├── internal/
│   ├── api/
│   │   ├── router.go            # Chi router setup
│   │   ├── projects.go          # Project CRUD handlers
│   │   ├── volumes.go           # Volume listing handler
│   │   ├── credentials.go       # Credential management
│   │   ├── restore.go           # Restore endpoint handler
│   │   └── sse.go               # SSE stream handler
│   ├── compose/
│   │   └── parser.go            # docker-compose.yml parsing (go-yaml)
│   ├── config/
│   │   ├── manifest.go          # Load/save encrypted manifest
│   │   └── key.go               # Encryption key management
│   ├── docker/
│   │   ├── client.go            # Docker client wrapper (docker/sdk)
│   │   ├── containers.go        # Stop/start/inspect containers
│   │   └── volumes.go           # Volume CRUD
│   ├── storage/
│   │   ├── backend.go           # Storage backend interface
│   │   ├── s3.go                # S3 implementation (minio SDK)
│   │   ├── webdav.go            # WebDAV implementation
│   │   └── ssh.go               # SSH/SFTP implementation
│   ├── restore/
│   │   ├── orchestrator.go      # Main restore orchestrator
│   │   ├── new_volume.go        # New volume restore mode
│   │   └── full_stack.go        # Full stack restore mode
│   └── sse/
│       └── broadcaster.go       # SSE event broadcaster
├── web/
│   ├── src/
│   │   ├── lib/
│   │   │   └── api.ts           # API client
│   │   ├── app.svelte
│   │   └── routes/
│   │       ├── setup/
│   │       ├── projects/
│   │       └── restore/
│   ├── svelte.config.js
│   ├── vite.config.ts
│   └── package.json
├── Dockerfile
├── docker-compose.yml           # For development
├── go.mod
├── Makefile
└── README.md
```

## Key Decisions Summary

| Decision | Choice |
|---|---|
| Language | Go backend, Svelte SPA frontend |
| Framework | Chi (Go), Vite (Svelte) |
| Deploy | Docker container (single binary) |
| Config | Encrypted JSON on disk (volume mount) |
| Compose input | Pasted content (MVP), Git later |
| Restore modes | New volume + Full stack restore |
| Docker access | Docker socket mount |
| Orchestration | Go SDK for Docker ops, shell commands for complex pipelines |
| Multi-project | Yes |
| Progress tracking | SSE (Server-Sent Events) |
| Navigation | Project → volume → backups → restore |
| GPG | Per-backup passphrase prompt |
| Config persistence | Yes, encrypted, survives restarts |
| Drift detection | Timestamps + compose hash mismatch flagging |
