export type ProjectSource = 'paste' | 'git' | 'portainer';

export interface PortainerSource {
  portainer_url: string;
  stack_id: number;
  endpoint_id: number;
  api_key?: string;
}

export interface PortainerEndpoint {
  Id: number;
  Name: string;
}

export interface PortainerStack {
  Id: number;
  Name: string;
  EndpointId: number;
  Status: number;
  is_offen_backed: boolean;
}

export interface PortainerConnectResponse {
  endpoints: PortainerEndpoint[];
}

export interface GitSource {
  repo_url: string;
  branch: string;
  file_path: string;
  last_synced_commit?: string;
  auth_token?: string;
  ssh_private_key?: string;
}

export interface GitImportRequest {
  repo_url: string;
  branch: string;
  file_path: string;
  auth_token?: string;
  ssh_private_key?: string;
  project_name: string;
}

export interface GitSyncResponse {
  changed: boolean;
  diff?: ComposeDiff;
  commit: string;
}

export interface Project {
  id: string;
  name: string;
  source?: ProjectSource;
  deployment_mode?: string;
  compose_content: string;
  compose_hash: string;
  added_at: string;
  last_refreshed: string;
  volumes: Volume[];
  credentials?: Credentials;
  default_passphrase?: string;
  git_source?: GitSource;
  portainer_source?: PortainerSource;
}

export interface Volume {
  name: string;
  compose_service: string;
  compose_mount_path: string;
  backup_pattern?: string;
  passphrase?: string;
}

export interface ImportResponse {
  id: string;
  volumes: VolumeInfo[];
  drift_hash: string;
}

export interface VolumeInfo {
  name: string;
  compose_service: string;
  compose_mount_path: string;
  backup_pattern?: string;
}

export interface BackupFile {
  key: string;
  size: number;
  last_modified: string;
  is_encrypted: boolean;
}

export interface Credentials {
  type: 'local' | 'smb';
  local?: { path: string };
  smb?: {
    host: string;
    share: string;
    path: string;
    username: string;
    password: string;
    port: number;
  };
}

export interface CredentialResponse {
  type: 'local' | 'smb';
  local?: { path: string };
  smb?: { host: string; share: string; path: string; username: string; port: number };
}

export interface RestoreRequest {
  mode: 'new_volume' | 'full_stack';
  backup_key: string;
  target_volume_name?: string;
  passphrase?: string;
  container_ids?: string[];
}

export interface RestoreResponse {
  restore_token: string;
  status: string;
}

export interface ProgressEvent {
  step: string;
  status: 'in_progress' | 'done' | 'failed' | 'error';
  message: string;
  percent?: number;
  details?: string;
  volume?: string;
  volume_index?: number;
  volume_total?: number;
}

export interface TimestampGroup {
  timestamp: string;
  backups: TimestampBackup[];
  complete: boolean;
}

export interface TimestampBackup {
  volume_name: string;
  key: string;
  size: number;
  is_encrypted: boolean;
}

export interface ComposeRestoreRequest {
  mode: 'full_stack' | 'new_volume';
  timestamp: string;
  volumes: ComposeVolumeRestore[];
  container_ids?: string[];
}

export interface ComposeVolumeRestore {
  volume_name: string;
  backup_key: string;
  passphrase: string;
  target_volume_name: string;
}

export interface InferredVolume {
  name: string;
  compose_service: string;
  compose_mount_path: string;
  backup_pattern?: string;
  target_volume_name: string;
  is_encrypted: boolean;
}

export interface StorageSuggestion {
  type: string;
  confidence: string;
  source: string;
  local_path?: string;
  smb_host?: string;
  smb_share?: string;
  smb_username?: string;
}

export interface InferenceResult {
  volumes: InferredVolume[];
  storage?: StorageSuggestion;
}

export interface ComposeDiffVolume {
  name: string;
  compose_service: string;
  compose_mount_path: string;
  backup_pattern?: string;
}

export interface ComposeDiff {
  added: ComposeDiffVolume[];
  removed: ComposeDiffVolume[];
  unchanged: ComposeDiffVolume[];
}

export interface SavedSource {
  id: string;
  name: string;
  type: 'git' | 'portainer';
  git_config?: GitSource;
  portainer_config?: PortainerSource;
}

export interface SavedBackend {
  id: string;
  name: string;
  credentials: Credentials;
}
