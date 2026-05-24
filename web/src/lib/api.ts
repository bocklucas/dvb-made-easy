import type {
  BackupFile,
  ComposeDiff,
  ComposeRestoreRequest,
  CredentialResponse,
  Credentials,
  GitImportRequest,
  GitSource,
  GitSyncResponse,
  ImportResponse,
  InferenceResult,
  PortainerConnectResponse,
  PortainerSource,
  PortainerStack,
  Project,
  RestoreRequest,
  RestoreResponse,
  SavedBackend,
  SavedSource,
  TimestampGroup
} from './types';

export class ApiError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
  }
}

async function extractError(res: Response): Promise<string> {
  try {
    const body = await res.json();
    return body.error || res.statusText;
  } catch {
    return res.statusText;
  }
}

async function apiGet<T>(path: string): Promise<T> {
  const res = await fetch(path);
  if (!res.ok) {
    throw new ApiError(res.status, await extractError(res));
  }
  return res.json() as Promise<T>;
}

async function apiPost<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body)
  });
  if (!res.ok) {
    throw new ApiError(res.status, await extractError(res));
  }
  return res.json() as Promise<T>;
}

async function apiDelete(path: string): Promise<void> {
  const res = await fetch(path, { method: 'DELETE' });
  if (!res.ok) {
    throw new ApiError(res.status, await extractError(res));
  }
}

async function apiPut<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(path, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body)
  });
  if (!res.ok) {
    throw new ApiError(res.status, await extractError(res));
  }
  return res.json() as Promise<T>;
}

// Projects

// Portainer

export function portainerConnect(url: string, apiKey: string): Promise<PortainerConnectResponse> {
  return apiPost<PortainerConnectResponse>('/api/portainer/connect', { url, api_key: apiKey });
}

export function portainerStacks(url: string, apiKey: string, endpointId: number): Promise<PortainerStack[]> {
  return apiPost<PortainerStack[]>('/api/portainer/stacks', {
    url,
    api_key: apiKey,
    endpoint_id: endpointId
  });
}

export function importPortainer(req: {
  portainer_url: string;
  api_key: string;
  stack_id: number;
  endpoint_id: number;
  project_name: string;
}): Promise<ImportResponse> {
  return apiPost<ImportResponse>('/api/projects/import-portainer', req);
}

// Projects

export function importCompose(content: string, deploymentMode?: string): Promise<ImportResponse> {
  return apiPost<ImportResponse>('/api/projects/import', { compose_content: content, deployment_mode: deploymentMode });
}

export function inferFromCompose(content: string): Promise<InferenceResult> {
  return apiPost<InferenceResult>('/api/projects/infer', { compose_content: content });
}

export function importGit(req: GitImportRequest): Promise<ImportResponse> {
  return apiPost<ImportResponse>('/api/projects/import-git', req);
}

export function syncGit(projectId: string): Promise<GitSyncResponse> {
  return apiPost<GitSyncResponse>(`/api/projects/${projectId}/git-sync`, {});
}

export function createProject(id: string, name: string): Promise<Project> {
  return apiPost<Project>('/api/projects', { id, name });
}

export function listProjects(): Promise<Project[]> {
  return apiGet<Project[]>('/api/projects');
}

export function getProject(id: string): Promise<Project> {
  return apiGet<Project>(`/api/projects/${id}`);
}

export function deleteProject(id: string): Promise<void> {
  return apiDelete(`/api/projects/${id}`);
}

export function updateCompose(projectId: string, composeContent: string): Promise<ComposeDiff> {
  return apiPut<ComposeDiff>(`/api/projects/${projectId}/compose`, {
    compose_content: composeContent
  });
}

export async function confirmVolumeRemoval(projectId: string, volumes: string[]): Promise<void> {
  const res = await fetch(`/api/projects/${projectId}/compose/confirm-removal`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ volumes })
  });
  if (!res.ok) {
    throw new ApiError(res.status, await extractError(res));
  }
}

// Credentials

export async function saveCredentials(projectId: string, creds: Credentials): Promise<void> {
  await apiPost<unknown>(`/api/projects/${projectId}/credentials`, creds);
}

export async function updateCredentials(projectId: string, creds: Credentials): Promise<void> {
  await apiPut<unknown>(`/api/projects/${projectId}/credentials`, creds);
}

export function getCredentials(projectId: string): Promise<CredentialResponse> {
  return apiGet<CredentialResponse>(`/api/projects/${projectId}/credentials`);
}

export function deleteCredentials(projectId: string): Promise<void> {
  return apiDelete(`/api/projects/${projectId}/credentials`);
}

// Backups

export function listBackups(projectId: string, volume: string): Promise<BackupFile[]> {
  return apiGet<BackupFile[]>(`/api/projects/${projectId}/volumes/${volume}/backups`);
}

// Restore

export function startRestore(
  projectId: string,
  volume: string,
  req: RestoreRequest
): Promise<RestoreResponse> {
  return apiPost<RestoreResponse>(`/api/projects/${projectId}/volumes/${volume}/restore`, req);
}

// Passphrases

export async function setVolumePassphrase(
  projectId: string,
  volume: string,
  passphrase: string
): Promise<void> {
  await apiPut<unknown>(`/api/projects/${projectId}/volumes/${volume}/passphrase`, { passphrase });
}

export function deleteVolumePassphrase(projectId: string, volume: string): Promise<void> {
  return apiDelete(`/api/projects/${projectId}/volumes/${volume}/passphrase`);
}

export async function setProjectPassphrase(projectId: string, passphrase: string): Promise<void> {
  await apiPut<unknown>(`/api/projects/${projectId}/passphrase`, { passphrase });
}

export function deleteProjectPassphrase(projectId: string): Promise<void> {
  return apiDelete(`/api/projects/${projectId}/passphrase`);
}

// Compose Restore

export function getBackupTimestamps(projectId: string): Promise<TimestampGroup[]> {
  return apiGet<TimestampGroup[]>(`/api/projects/${projectId}/backup-timestamps`);
}

export function startComposeRestore(
  projectId: string,
  req: ComposeRestoreRequest
): Promise<RestoreResponse> {
  return apiPost<RestoreResponse>(`/api/projects/${projectId}/compose-restore`, req);
}

export function subscribeRestore(
  token: string,
  onEvent: (event: import('./types').ProgressEvent) => void,
  onError: (err: Event) => void
): () => void {
  const source = new EventSource(`/api/restore/${token}/stream`);

  source.onmessage = (e: MessageEvent) => {
    try {
      const data = JSON.parse(e.data as string) as import('./types').ProgressEvent;
      onEvent(data);
    } catch {
      // ignore malformed messages
    }
  };

  source.onerror = (err) => {
    source.close();
    onError(err);
  };

  return () => source.close();
}

// Saved Sources

export function listSavedSources(): Promise<SavedSource[]> {
  return apiGet<SavedSource[]>('/api/sources');
}

export function createSavedSource(source: {
  name: string;
  type: string;
  git_config?: GitSource;
  portainer_config?: PortainerSource;
}): Promise<SavedSource> {
  return apiPost<SavedSource>('/api/sources', source);
}

export function deleteSavedSource(id: string): Promise<void> {
  return apiDelete(`/api/sources/${id}`);
}

// Saved Backends

export function listSavedBackends(): Promise<SavedBackend[]> {
  return apiGet<SavedBackend[]>('/api/backends');
}

export function createSavedBackend(backend: { name: string; credentials: Credentials }): Promise<SavedBackend> {
  return apiPost<SavedBackend>('/api/backends', backend);
}

export function deleteSavedBackend(id: string): Promise<void> {
  return apiDelete(`/api/backends/${id}`);
}
