<script lang="ts">
  import type { CredentialResponse, Credentials, SavedBackend } from '$lib/types';
  import { updateCredentials, listSavedBackends, createSavedBackend } from '$lib/api';
  import { onMount } from 'svelte';
  import CredentialFields from '$lib/components/CredentialFields.svelte';

  interface Props {
    projectId: string;
    existing: CredentialResponse;
    onClose: () => void;
    onSaved: () => void;
  }

  let { projectId, existing, onClose, onSaved }: Props = $props();

  function getInitialCredentials(): Credentials {
    return {
      type: existing.type,
      local: existing.local ? { ...existing.local } : undefined,
      smb: existing.smb ? { ...existing.smb, password: '' } : undefined,
      s3: existing.s3 ? { ...existing.s3, secret_key: '' } : undefined,
      webdav: existing.webdav ? { ...existing.webdav, password: '' } : undefined,
      azure: existing.azure ? { ...existing.azure, connection_string: '' } : undefined,
      dropbox: existing.dropbox ? { ...existing.dropbox, access_token: '', app_secret: '' } : undefined,
      gdrive: existing.gdrive ? { ...existing.gdrive, credentials: '' } : undefined,
      sftp: existing.sftp ? { ...existing.sftp, password: '', private_key: '' } : undefined
    };
  }

  let credentials = $state<Credentials>(getInitialCredentials());

  // Saved backends
  let savedBackends = $state<SavedBackend[]>([]);
  let selectedSavedBackendId = $state('');
  let saveAsBackend = $state(false);
  let savedBackendName = $state('');

  let error = $state('');
  let testSuccess = $state(false);
  let testing = $state(false);
  let saving = $state(false);

  onMount(async () => {
    try {
      savedBackends = await listSavedBackends();
    } catch {
      // ignore — saved backends are optional
    }
  });

  function selectType(type: Credentials['type']) {
    credentials.type = type;
    error = '';
    testSuccess = false;
    selectedSavedBackendId = '';
  }

  function applySavedBackend() {
    const backend = savedBackends.find((b) => b.id === selectedSavedBackendId);
    if (!backend) return;

    error = '';
    testSuccess = false;
    saveAsBackend = false;

    credentials = {
      type: backend.credentials.type,
      local: backend.credentials.local ? { ...backend.credentials.local } : undefined,
      smb: backend.credentials.smb ? { ...backend.credentials.smb } : undefined,
      s3: backend.credentials.s3 ? { ...backend.credentials.s3 } : undefined,
      webdav: backend.credentials.webdav ? { ...backend.credentials.webdav } : undefined,
      azure: backend.credentials.azure ? { ...backend.credentials.azure } : undefined,
      dropbox: backend.credentials.dropbox ? { ...backend.credentials.dropbox } : undefined,
      gdrive: backend.credentials.gdrive ? { ...backend.credentials.gdrive } : undefined,
      sftp: backend.credentials.sftp ? { ...backend.credentials.sftp } : undefined,
      saved_backend_id: selectedSavedBackendId
    };
  }

  function buildCreds(): Credentials | null {
    if (credentials.type === 'local' && credentials.local) {
      if (!credentials.local.path.trim()) {
        error = 'Enter a local path';
        return null;
      }
      return { type: 'local', local: { path: credentials.local.path.trim() } };
    }
    
    if (credentials.type === 'smb' && credentials.smb) {
      if (!credentials.smb.host.trim() || !credentials.smb.share.trim()) {
        error = 'Host and share are required';
        return null;
      }
      return {
        type: 'smb',
        smb: {
          host: credentials.smb.host.trim(),
          share: credentials.smb.share.trim(),
          path: credentials.smb.path.trim(),
          username: credentials.smb.username.trim(),
          password: credentials.smb.password,
          port: credentials.smb.port
        }
      };
    }
    
    if (credentials.type === 's3' && credentials.s3) {
      if (!credentials.s3.bucket.trim()) {
        error = 'Bucket name is required';
        return null;
      }
      return {
        type: 's3',
        s3: {
          bucket: credentials.s3.bucket.trim(),
          access_key: credentials.s3.access_key.trim(),
          secret_key: credentials.s3.secret_key,
          endpoint: credentials.s3.endpoint.trim(),
          region: credentials.s3.region.trim(),
          storage_class: credentials.s3.storage_class.trim()
        }
      };
    }
    
    if (credentials.type === 'webdav' && credentials.webdav) {
      if (!credentials.webdav.url.trim()) {
        error = 'WebDAV URL is required';
        return null;
      }
      return {
        type: 'webdav',
        webdav: {
          url: credentials.webdav.url.trim(),
          username: credentials.webdav.username.trim(),
          password: credentials.webdav.password,
          path: credentials.webdav.path.trim(),
          insecure: credentials.webdav.insecure
        }
      };
    }
    
    if (credentials.type === 'azure' && credentials.azure) {
      if (!credentials.azure.container.trim()) {
        error = 'Container name is required';
        return null;
      }
      return {
        type: 'azure',
        azure: {
          connection_string: credentials.azure.connection_string,
          container: credentials.azure.container.trim()
        }
      };
    }
    
    if (credentials.type === 'dropbox' && credentials.dropbox) {
      if (!credentials.dropbox.app_key.trim()) {
        error = 'App Key is required';
        return null;
      }
      return {
        type: 'dropbox',
        dropbox: {
          access_token: credentials.dropbox.access_token,
          app_key: credentials.dropbox.app_key.trim(),
          app_secret: credentials.dropbox.app_secret,
          remote_path: credentials.dropbox.remote_path.trim()
        }
      };
    }
    
    if (credentials.type === 'gdrive' && credentials.gdrive) {
      if (!credentials.gdrive.folder_id.trim()) {
        error = 'Folder ID is required';
        return null;
      }
      return {
        type: 'gdrive',
        gdrive: {
          folder_id: credentials.gdrive.folder_id.trim(),
          credentials: credentials.gdrive.credentials?.trim(),
          impersonate: credentials.gdrive.impersonate.trim()
        }
      };
    }
    
    if (credentials.type === 'sftp' && credentials.sftp) {
      if (!credentials.sftp.host.trim() || !credentials.sftp.user.trim()) {
        error = 'Host and user are required';
        return null;
      }
      return {
        type: 'sftp',
        sftp: {
          host: credentials.sftp.host.trim(),
          user: credentials.sftp.user.trim(),
          port: credentials.sftp.port,
          password: credentials.sftp.password,
          private_key: credentials.sftp.private_key,
          remote_path: credentials.sftp.remote_path.trim()
        }
      };
    }

    return null;
  }

  async function handleTestConnection() {
    error = '';
    testSuccess = false;

    const creds = buildCreds();
    if (!creds) return;

    testing = true;
    try {
      await updateCredentials(projectId, creds);
      testSuccess = true;
    } catch (e) {
      error = e instanceof Error ? e.message : 'Connection test failed';
    } finally {
      testing = false;
    }
  }

  async function handleSave() {
    if (!testSuccess) return;

    const creds = buildCreds();
    if (!creds) return;

    saving = true;
    error = '';
    try {
      await updateCredentials(projectId, creds);

      // Save as reusable backend if requested
      if (saveAsBackend && savedBackendName.trim()) {
        try {
          await createSavedBackend({ name: savedBackendName.trim(), credentials: creds });
        } catch {
          // non-fatal
        }
      }

      onSaved();
      onClose();
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to save credentials';
    } finally {
      saving = false;
    }
  }
</script>

<!-- Backdrop -->
<div
  class="fixed inset-0 bg-black/40 z-40 flex items-center justify-center p-4"
  role="dialog"
  aria-modal="true"
  aria-labelledby="storage-edit-title"
>
  <div class="bg-slate-900 rounded-xl shadow-xl w-full max-w-lg max-h-[90vh] flex flex-col">
    <!-- Header -->
    <div class="flex items-center justify-between px-5 py-4 border-b border-slate-700/50">
      <h2 id="storage-edit-title" class="text-lg font-semibold text-slate-100">Edit Storage</h2>
      <button
        onclick={onClose}
        class="text-slate-500 hover:text-slate-400 transition-all duration-200 active:scale-[0.98]"
        aria-label="Close"
      >
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M6 18L18 6M6 6l12 12"
          />
        </svg>
      </button>
    </div>

    <!-- Body -->
    <div class="flex-1 overflow-y-auto px-5 py-4 space-y-5">
      {#if savedBackends.length > 0}
        <div>
          <label for="edit-saved-backend" class="block text-sm font-medium text-slate-300 mb-2">
            Use a saved backend
          </label>
          <select
            id="edit-saved-backend"
            bind:value={selectedSavedBackendId}
            onchange={applySavedBackend}
            class="w-full px-3 py-2 border border-slate-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-slate-900 text-slate-100"
          >
            <option value="">Configure manually...</option>
            {#each savedBackends as backend}
              <option value={backend.id}>{backend.name} ({backend.credentials.type})</option>
            {/each}
          </select>
        </div>
      {/if}

      <!-- Type selector -->
      <div>
        <p class="text-sm font-medium text-slate-300 mb-3">Storage Backend</p>
        <div class="grid grid-cols-2 gap-2">
          {#each [
            { type: 'local', name: 'Local Path', beta: false },
            { type: 'smb', name: 'SMB Share', beta: false },
            { type: 's3', name: 'AWS S3', beta: true },
            { type: 'webdav', name: 'WebDAV', beta: true },
            { type: 'azure', name: 'Azure Blob', beta: true },
            { type: 'dropbox', name: 'Dropbox', beta: true },
            { type: 'gdrive', name: 'Google Drive', beta: true },
            { type: 'sftp', name: 'SFTP / SSH', beta: true }
          ] as opt}
            <button
              onclick={() => selectType(opt.type as any)}
              class="py-2 px-3 rounded-lg border-2 text-sm font-medium transition-colors text-left flex items-center justify-between {credentials.type === opt.type ? 'border-indigo-500 bg-indigo-500/10 text-indigo-400' : 'border-slate-700/50 text-slate-400 hover:border-slate-600'}"
            >
              <span>{opt.name}</span>
              {#if opt.beta}
                <span class="text-[10px] px-1 py-0.5 bg-yellow-100 text-yellow-800 rounded font-semibold uppercase tracking-wider">Beta</span>
              {/if}
            </button>
          {/each}
        </div>
      </div>

      <div class="space-y-4">
        <CredentialFields bind:credentials={credentials} isEdit={true} />
      </div>

      {#if error}
        <div class="p-3 bg-red-500/10 border border-red-500/20 rounded-lg text-red-400 text-sm">
          {error}
        </div>
      {/if}

      {#if testSuccess}
        <div class="p-3 bg-emerald-500/10 border border-emerald-500/20 rounded-lg text-emerald-400 text-sm">
          Connection successful.
        </div>
      {/if}
    </div>

    <!-- Footer -->
    <div class="flex items-center justify-end gap-2 px-5 py-4 border-t border-slate-700/50">
      <button
        onclick={onClose}
        class="px-4 py-2 text-sm text-slate-300 border border-slate-600 rounded-lg hover:bg-slate-800/50 transition-all duration-200 active:scale-[0.98]"
      >
        Cancel
      </button>
      <button
        onclick={handleTestConnection}
        disabled={testing || saving}
        class="px-4 py-2 text-sm border border-indigo-600 text-indigo-400 hover:bg-indigo-500/10 disabled:opacity-50 rounded-lg font-medium transition-all duration-200 active:scale-[0.98]"
      >
        {testing ? 'Testing...' : 'Test Connection'}
      </button>
      <button
        onclick={handleSave}
        disabled={!testSuccess || saving}
        class="px-4 py-2 text-sm text-white bg-gradient-to-r from-indigo-500 to-violet-500 hover:from-indigo-400 hover:to-violet-400 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {saving ? 'Saving...' : 'Save'}
      </button>
    </div>
  </div>
</div>
