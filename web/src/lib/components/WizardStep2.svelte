<script lang="ts">
  import type { Credentials, SavedBackend } from '$lib/types';
  import { saveCredentials, listSavedBackends, createSavedBackend } from '$lib/api';
  import { onMount } from 'svelte';
  import CredentialFields from '$lib/components/CredentialFields.svelte';

  interface Props {
    projectId: string;
    onComplete: (backendType: Credentials['type']) => void;
    onBack: () => void;
  }

  let { projectId, onComplete, onBack }: Props = $props();

  let credentials = $state<Credentials>({
    type: 'local',
    local: { path: '' }
  });

  // Saved backends
  let savedBackends = $state<SavedBackend[]>([]);
  let selectedSavedBackendId = $state('');
  let saveAsBackend = $state(false);
  let savedBackendName = $state('');

  let error = $state('');
  let success = $state(false);
  let loading = $state(false);

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
    success = false;
    selectedSavedBackendId = '';
  }

  function applySavedBackend() {
    const backend = savedBackends.find((b) => b.id === selectedSavedBackendId);
    if (!backend) return;

    error = '';
    success = false;
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

    applyDirectly(credentials);
  }

  async function applyDirectly(creds: Credentials) {
    loading = true;
    error = '';
    try {
      await saveCredentials(projectId, creds);
      success = true;
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to apply saved backend';
    } finally {
      loading = false;
    }
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
    success = false;

    const creds = buildCreds();
    if (!creds) return;

    loading = true;
    try {
      await saveCredentials(projectId, creds);
      success = true;

      // If user wants to save this backend for reuse
      if (saveAsBackend && savedBackendName.trim()) {
        try {
          await createSavedBackend({ name: savedBackendName.trim(), credentials: creds });
          savedBackends = await listSavedBackends();
        } catch {
          // non-fatal
        }
      }
    } catch (e) {
      error = e instanceof Error ? e.message : 'Connection test failed';
    } finally {
      loading = false;
    }
  }
</script>

<div class="space-y-6">
  {#if savedBackends.length > 0}
    <div>
      <label for="saved-backend" class="block text-sm font-medium text-slate-300 mb-2">
        Use a saved backend
      </label>
      <select
        id="saved-backend"
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
    <CredentialFields bind:credentials={credentials} />
  </div>

  {#if error}
    <div class="p-3 bg-red-500/10 border border-red-500/20 rounded-lg text-red-400 text-sm">{error}</div>
  {/if}

  {#if success}
    <div class="p-3 bg-emerald-500/10 border border-emerald-500/20 rounded-lg text-emerald-400 text-sm">
      Connection successful — credentials saved.
    </div>
  {/if}

  {#if !selectedSavedBackendId}
    <div class="flex items-start gap-3">
      <input
        id="save-backend"
        type="checkbox"
        bind:checked={saveAsBackend}
        class="mt-1 h-4 w-4 rounded border-slate-600 text-indigo-400 focus:ring-indigo-500"
      />
      <div class="flex-1">
        <label for="save-backend" class="text-sm font-medium text-slate-300">
          Save this backend for later
        </label>
        {#if saveAsBackend}
          <input
            type="text"
            bind:value={savedBackendName}
            placeholder="Backend name"
            class="mt-2 w-full px-3 py-2 border border-slate-600 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 text-sm bg-slate-900 text-slate-100"
          />
        {/if}
      </div>
    </div>
  {/if}

  <button
    onclick={handleTestConnection}
    disabled={loading}
    class="w-full py-2 px-4 border border-indigo-600 text-indigo-400 hover:bg-indigo-500/10 disabled:opacity-50 rounded-lg font-medium transition-all duration-200 active:scale-[0.98]"
  >
    {loading ? 'Testing...' : 'Test Connection'}
  </button>

  <div class="flex gap-3">
    <button
      onclick={onBack}
      class="flex-1 py-2 px-4 border border-slate-600 text-slate-300 hover:bg-slate-800/50 rounded-lg font-medium transition-all duration-200 active:scale-[0.98]"
    >
      ← Back
    </button>
    <button
      onclick={() => onComplete(credentials.type)}
      disabled={!success}
      class="flex-1 py-2 px-4 bg-gradient-to-r from-indigo-500 to-violet-500 hover:from-indigo-400 hover:to-violet-400 disabled:opacity-50 text-white rounded-lg font-medium transition-all duration-200 active:scale-[0.98]"
    >
      Next
    </button>
  </div>
</div>
