# 💾 Offen Made Easy (Offen Restore Manager)

[![Go Version](https://img.shields.io/badge/Go-1.26-blue.svg?style=flat-square&logo=go)](https://golang.org)
[![SvelteKit](https://img.shields.io/badge/SvelteKit-2-FF3E00.svg?style=flat-square&logo=svelte)](https://kit.svelte.dev)
[![Tailwind CSS](https://img.shields.io/badge/Tailwind_CSS-4-38B2AC.svg?style=flat-square&logo=tailwind-css)](https://tailwindcss.com)
[![Docker](https://img.shields.io/badge/Docker-Engine_SDK-2496ED.svg?style=flat-square&logo=docker)](https://www.docker.com)
[![Storage: Local & SMB](https://img.shields.io/badge/Storage-Local%20%7C%20SMB-brightgreen?style=flat-square)](#)
[![Storage: S3 & Azure](https://img.shields.io/badge/Cloud-S3%20%7C%20Azure-blue?style=flat-square)](#)
[![Storage: WebDAV & SFTP](https://img.shields.io/badge/Network-WebDAV%20%7C%20SFTP-orange?style=flat-square)](#)
[![Storage: Dropbox & GDrive](https://img.shields.io/badge/SaaS-Dropbox%20%7C%20GDrive-9cf?style=flat-square)](#)

**Offen Made Easy** (also known as the *Offen Restore Manager*) is a lightweight, self-hosted web application and orchestration engine designed to automate the process of restoring Docker volume backups created by the popular [offen/docker-volume-backup](https://github.com/offen/docker-volume-backup) utility.

> [!NOTE]
> This entire project was co-authored with **Claude** and **Anti-Gravity**.

---

## 🔍 The Problem & The Pain Points

While `offen/docker-volume-backup` is an excellent tool for scheduled backups, GPG encryption, and cloud/local uploads, **it completely lacks a restore interface**. 

To restore a volume backup manually, a developer must go through this complex, error-prone list of steps:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    MANUAL RESTORE WORKFLOW (PAIN POINTS)                │
├─────────────────────────────────────────────────────────────────────────┤
│  1. Stop/Scale Down the consuming containers (prevent database corrupt) │
│  2. Fetch/Download the compressed `.tar.gz` or `.tar.gz.gpg` archive     │
│  3. Decrypt the archive manually using GPG (if encrypted)               │
│  4. Run a temporary helper container mounting the target Docker volume   │
│  5. Extract the archive (carefully calculating --strip-components 2)    │
│  6. Clean up temporary extraction containers and staging files          │
│  7. Scale Up/Restart the original container stack                       │
└─────────────────────────────────────────────────────────────────────────┘
```

**Offen Made Easy** replaces this manual nightmare with a **gorgeous, single-click web GUI** that orchestrates the entire lifecycle safely, quickly, and transparently.

---

## ✨ Features

*   **🔌 Flexible Project Import Methods**
    *   **Direct Paste**: Paste your `docker-compose.yml` file to get going in seconds.
    *   **Git Repository**: Import compose configurations directly from Git (supports HTTPS access tokens and SSH Private Keys). Sync changes in one click.
    *   **Portainer Integration**: Connect to Portainer, list endpoints/stacks, auto-discover Offen-backed applications, and import them directly.
*   **🛡️ Cryptographic Passphrase Persistence**
    *   Securely stores volume passphrases or project-wide default passphrases.
    *   Survives container restarts using an **AES-256-GCM encrypted local manifest storage** (`manifest.enc`).
*   **⚡ Automated Volume Restore**
    *   Create a new volume (e.g. `{volume}_restored_{timestamp}`) and extract the backup directly into it. Extremely safe for inspecting data before replacing production volumes.
*   **🐳 Native Docker Swarm & Compose Support**
    *   Orchestrates scaling down of Docker Swarm services (or stopping Compose containers) to safely perform restores without database corruption.
    *   Cleans and prepares target volumes, performs the restore, and automatically restores/scales up services to their previous configuration when done.
*   **📡 Real-Time Progress Streaming**
    *   Leverages Server-Sent Events (SSE) to broadcast live percentages, logs, container transitions, and extraction status.
*   **💾 Reusable Source & Storage Backends**
    *   Configure Git repositories or Portainer credentials once and reuse them.
    *   Supports a wide array of storage backends: **Local Directory**, **SMB/CIFS**, **AWS S3 / MinIO**, **WebDAV (Nextcloud)**, **SFTP**, **Azure Blob Storage**, **Dropbox**, and **Google Drive**.
*   **🛠️ Sidecar Backup Generator (Configuration Wizard)**
    *   Easily generate and inject the `offen/docker-volume-backup` sidecar service definition into your existing `docker-compose.yml` file.
    *   Configure schedule (cron), retention policies, GPG encryption passphrases, service downtime orchestration (labels to stop containers during backups), and automatic SMB volume mounting.

---

## 🚀 Quick Start

Get up and running with a single command — no cloning required:

```bash
curl -sL https://raw.githubusercontent.com/bocklucas/offen-made-easy/main/docker-compose.prod.yml -o docker-compose.yml && docker compose up -d
```

Or run it directly without a compose file (requires creating the staging volume for temp downloads):

```bash
docker volume create offen-restore-staging

docker run -d -p 7331:7331 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v offen-config:/app/config \
  -v offen-restore-staging:/staging \
  ghcr.io/bocklucas/offen-made-easy:latest
```

Once running, open [http://localhost:7331](http://localhost:7331) in your browser.

> [!TIP]
> If you get a "permission denied" error for the Docker socket, add your Docker group ID:
> ```bash
> docker run -d -p 7331:7331 \
>   -v /var/run/docker.sock:/var/run/docker.sock \
>   -v offen-config:/app/config \
>   -v offen-restore-staging:/staging \
>   --group-add $(getent group docker | cut -d: -f3) \
>   ghcr.io/bocklucas/offen-made-easy:latest
> ```

---

## 🛠️ Local Development (Docker Compose)

To build and run from source using Docker Compose:

```bash
git clone https://github.com/bocklucas/offen-made-easy.git
cd offen-made-easy

# Export your Docker group ID (optional but recommended for permissions)
export DOCKER_GID=$(getent group docker | cut -d: -f3)

docker compose up -d
```

Once the stack is running, navigate to:
*   **Web Console (GUI)**: [http://localhost:7331](http://localhost:7331)
*   **API Documentation / Health**: [http://localhost:7331/api/health](http://localhost:7331/api/health)

---

## 🧑‍💻 Local Development (No Docker)

If you wish to run the backend and frontend services locally outside of Docker containers:

### Prerequisites
*   Go (version 1.26 or newer)
*   Node.js (version 22 or newer)
*   Access to a local Docker daemon (via `/var/run/docker.sock`)

### 1. Start the Go Backend
```bash
# Set custom folders for the encrypted database and local staging
go run cmd/server/main.go --port 7331 --config-dir ./config-dev --staging-dir ./staging-dev
```

### 2. Start the Frontend Dev Server
```bash
cd web
npm install
npm run dev
```
Open [http://localhost:5173](http://localhost:5173) in your browser. The Vite development proxy will forward `/api` requests to the Go backend on port `7331`.

---

## 🔒 Security Model

### 1. AES-GCM Encrypted Manifest
Offen Made Easy keeps your credentials (like SMB credentials, Git auth tokens, and GPG passphrases) secure.
*   Upon the first start, a secure 32-byte key is randomly generated and saved locally as `/app/config/.key` with strict `0600` permissions.
*   Your project settings, connections, and GPG passwords are marshaled to JSON, encrypted using **AES-256-GCM**, and saved as `manifest.enc` (also with `0600` permissions).
*   *Make sure to back up your `.key` file alongside your config folder! Without the key, the configuration database cannot be decrypted.*

### 2. Docker Socket Access
*   The application requires access to the Docker socket `/var/run/docker.sock` to check container states and spin up extraction helper containers.
*   Ensure that only trusted administrators have network access to port `7331` (and `5173` if running the dev server).

---

## ❔ FAQ & Troubleshooting

#### 1. "permission denied" when connecting to `/var/run/docker.sock`
This occurs if the Docker socket mount has different permissions. Ensure that:
*   You set the `DOCKER_GID` environment variable before running `docker compose up`.
*   The user executing the container is added to the correct group, or run the container as `root` (not recommended).

#### 2. How do I restore into a different volume name?
During the restore flow in the UI, select **New Volume Mode**. It will prompt you for a target volume name, defaulting to `{volume_name}_restored_{timestamp}`. You can customize this to whatever name you want, and the app will create and populate it.

#### 3. My backups are GPG-encrypted. How does decryption work?
When importing your volume or setting up the restore, the UI will display a lock icon if it detects `.gpg` files. You can:
1. Store the GPG passphrase in the UI so that it is encrypted and saved in `manifest.enc`.
2. Enter the passphrase dynamically at the time of restore.
During restore, the background Alpine helper container automatically installs `gnupg` and decrypts the stream before piping it to `tar`.
