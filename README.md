# 💾 Offen Made Easy (Offen Restore Manager)

[![Go Version](https://img.shields.io/badge/Go-1.26-blue.svg?style=flat-square&logo=go)](https://golang.org)
[![SvelteKit](https://img.shields.io/badge/SvelteKit-2-FF3E00.svg?style=flat-square&logo=svelte)](https://kit.svelte.dev)
[![Tailwind CSS](https://img.shields.io/badge/Tailwind_CSS-4-38B2AC.svg?style=flat-square&logo=tailwind-css)](https://tailwindcss.com)
[![Docker](https://img.shields.io/badge/Docker-Engine_SDK-2496ED.svg?style=flat-square&logo=docker)](https://www.docker.com)
[![SMB/CIFS Supported](https://img.shields.io/badge/Storage-Local%20%7C%20SMB-brightgreen.svg?style=flat-square)](#)

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
*   **📡 Real-Time Progress Streaming**
    *   Leverages Server-Sent Events (SSE) to broadcast live percentages, logs, container transitions, and extraction status.
*   **💾 Reusable Source & Storage Backends**
    *   Configure your Git repositories, Portainer credentials, or SMB/Local storage options once and share them across multiple backup/restore projects.

---

## 🚀 Getting Started

The easiest way to get Offen Made Easy running is to clone this repository and start the pre-configured Docker Compose stack.

### 1. Clone the Repository
Clone the repository to your host machine and navigate into the project directory:

```bash
# Clone the repository
git clone https://github.com/bocklucas/offen-made-easy.git

# CD into the directory
cd offen-made-easy
```

### 2. Start the Stack
Ensure you have Docker and Docker Compose installed. Since the backend needs access to the Docker socket `/var/run/docker.sock`, export your local Docker group ID (`DOCKER_GID`) to ensure the container has the correct socket permissions, then start the stack:

```bash
# Export your Docker group ID (optional but recommended for permissions)
export DOCKER_GID=$(getent group docker | cut -d: -f3)

# Start the stack
docker compose up -d
```

Once the stack is running, navigate to:
*   **Web Console (GUI)**: [http://localhost:5173](http://localhost:5173)
*   **API Documentation / Health**: [http://localhost:8080/api/health](http://localhost:8080/api/health)

---

## 🛠️ Local Development (No Docker)

If you wish to run the backend and frontend services locally outside of Docker containers:

### Prerequisites
*   Go (version 1.26 or newer)
*   Node.js (version 22 or newer)
*   Access to a local Docker daemon (via `/var/run/docker.sock`)

### 1. Start the Go Backend
```bash
# Set a custom folder for the encrypted database
go run cmd/server/main.go --port 8080 --config-dir ./config-dev
```

### 2. Start the Frontend Dev Server
```bash
cd web
npm install
npm run dev
```
Open [http://localhost:5173](http://localhost:5173) in your browser. The Vite development proxy will forward `/api` requests to the Go backend on port `8080`.

---

## 🔒 Security Model

### 1. AES-GCM Encrypted Manifest
Offen Made Easy keeps your credentials (like SMB credentials, Git auth tokens, and GPG passphrases) secure.
*   Upon the first start, a secure 32-byte key is randomly generated and saved locally as `/app/config/.key` with strict `0600` permissions.
*   Your project settings, connections, and GPG passwords are marshaled to JSON, encrypted using **AES-256-GCM**, and saved as `manifest.enc` (also with `0600` permissions).
*   *Make sure to back up your `.key` file alongside your config folder! Without the key, the configuration database cannot be decrypted.*

### 2. Docker Socket Access
*   The application requires access to the Docker socket `/var/run/docker.sock` to check container states and spin up extraction helper containers.
*   Ensure that only trusted administrators have network access to port `5173` and `8080`.

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
