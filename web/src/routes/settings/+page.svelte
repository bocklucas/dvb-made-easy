<script lang="ts">
  import { onMount } from 'svelte';
  import type { SavedSource, SavedBackend, GitSource, PortainerSource, Credentials } from '$lib/types';
  import {
    listSavedSources,
    getSavedSource,
    createSavedSource,
    updateSavedSource,
    deleteSavedSource,
    listSavedBackends,
    getSavedBackend,
    createSavedBackend,
    updateSavedBackend,
    deleteSavedBackend
  } from '$lib/api';
  import CredentialFields from '$lib/components/CredentialFields.svelte';

  let activeTab = $state<'sources' | 'backends'>('sources');

  // Sources state
  let sources = $state<SavedSource[]>([]);
  let sourcesLoading = $state(true);
  let sourcesError = $state('');
  let editingSourceId = $state<string | null>(null);
  let addingSource = $state(false);
  let confirmDeleteSourceId = $state<string | null>(null);

  // Source form state
  let sourceFormName = $state('');
  let sourceFormType = $state<'git' | 'portainer'>('git');
  let sourceFormRepoUrl = $state('');
  let sourceFormBranch = $state('main');
  let sourceFormAuthType = $state<'none' | 'token' | 'ssh'>('none');
  let sourceFormAuthToken = $state('');
  let sourceFormSshKey = $state('');
  let sourceFormPortainerUrl = $state('');
  let sourceFormApiKey = $state('');
  let sourceFormSaving = $state(false);
  let sourceFormError = $state('');

  // Backends state
  let backends = $state<SavedBackend[]>([]);
  let backendsLoading = $state(true);
  let backendsError = $state('');
  let editingBackendId = $state<string | null>(null);
  let addingBackend = $state(false);
  let confirmDeleteBackendId = $state<string | null>(null);

  // Backend form state
  let backendFormName = $state('');
  let backendFormCredentials = $state<Credentials>({
    type: 'local',
    local: { path: '' }
  });
  let backendFormSaving = $state(false);
  let backendFormError = $state('');

  onMount(async () => {
    await Promise.all([loadSources(), loadBackends()]);
  });

  async function loadSources() {
    sourcesLoading = true;
    sourcesError = '';
    try {
      sources = await listSavedSources();
    } catch (e) {
      sourcesError = e instanceof Error ? e.message : 'Failed to load sources';
    } finally {
      sourcesLoading = false;
    }
  }

  async function loadBackends() {
    backendsLoading = true;
    backendsError = '';
    try {
      backends = await listSavedBackends();
    } catch (e) {
      backendsError = e instanceof Error ? e.message : 'Failed to load backends';
    } finally {
      backendsLoading = false;
    }
  }

  // --- Source form helpers ---

  function resetSourceForm() {
    sourceFormName = '';
    sourceFormType = 'git';
    sourceFormRepoUrl = '';
    sourceFormBranch = 'main';
    sourceFormAuthType = 'none';
    sourceFormAuthToken = '';
    sourceFormSshKey = '';
    sourceFormPortainerUrl = '';
    sourceFormApiKey = '';
    sourceFormSaving = false;
    sourceFormError = '';
  }

  function openAddSource() {
    editingSourceId = null;
    resetSourceForm();
    addingSource = true;
  }

  function cancelSourceForm() {
    addingSource = false;
    editingSourceId = null;
    resetSourceForm();
  }

  async function openEditSource(id: string) {
    addingSource = false;
    editingSourceId = id;
    resetSourceForm();
    try {
      const source = await getSavedSource(id);
      sourceFormName = source.name;
      sourceFormType = source.type;
      if (source.git_config) {
        sourceFormRepoUrl = source.git_config.repo_url ?? '';
        sourceFormBranch = source.git_config.branch ?? 'main';
        if (source.git_config.auth_token) {
          sourceFormAuthType = 'token';
          sourceFormAuthToken = source.git_config.auth_token;
        } else if (source.git_config.ssh_private_key) {
          sourceFormAuthType = 'ssh';
          sourceFormSshKey = source.git_config.ssh_private_key;
        }
      }
      if (source.portainer_config) {
        sourceFormPortainerUrl = source.portainer_config.portainer_url ?? '';
        sourceFormApiKey = source.portainer_config.api_key ?? '';
      }
    } catch {
      // Fall back to sanitized data
      const source = sources.find((s) => s.id === id);
      if (source) {
        sourceFormName = source.name;
        sourceFormType = source.type;
        if (source.git_config) {
          sourceFormRepoUrl = source.git_config.repo_url ?? '';
          sourceFormBranch = source.git_config.branch ?? 'main';
        }
        if (source.portainer_config) {
          sourceFormPortainerUrl = source.portainer_config.portainer_url ?? '';
        }
      }
    }
  }

  function buildSourcePayload(): { name: string; type: string; git_config?: GitSource; portainer_config?: PortainerSource } | null {
    if (!sourceFormName.trim()) {
      sourceFormError = 'Name is required';
      return null;
    }

    if (sourceFormType === 'git') {
      if (!sourceFormRepoUrl.trim()) {
        sourceFormError = 'Repository URL is required';
        return null;
      }
      return {
        name: sourceFormName.trim(),
        type: 'git',
        git_config: {
          repo_url: sourceFormRepoUrl.trim(),
          branch: sourceFormBranch.trim() || 'main',
          auth_token: sourceFormAuthType === 'token' ? sourceFormAuthToken.trim() : undefined,
          ssh_private_key: sourceFormAuthType === 'ssh' ? sourceFormSshKey.trim() : undefined
        }
      };
    } else {
      if (!sourceFormPortainerUrl.trim()) {
        sourceFormError = 'Portainer URL is required';
        return null;
      }
      if (!sourceFormApiKey.trim()) {
        sourceFormError = 'API key is required';
        return null;
      }
      return {
        name: sourceFormName.trim(),
        type: 'portainer',
        portainer_config: {
          portainer_url: sourceFormPortainerUrl.trim(),
          api_key: sourceFormApiKey.trim(),
          stack_id: 0,
          endpoint_id: 0
        }
      };
    }
  }

  async function handleSaveSource() {
    sourceFormError = '';
    const payload = buildSourcePayload();
    if (!payload) return;

    sourceFormSaving = true;
    try {
      if (editingSourceId) {
        await updateSavedSource(editingSourceId, payload);
      } else {
        await createSavedSource(payload);
      }
      cancelSourceForm();
      await loadSources();
    } catch (e) {
      sourceFormError = e instanceof Error ? e.message : 'Failed to save source';
    } finally {
      sourceFormSaving = false;
    }
  }

  async function handleDeleteSource(id: string) {
    if (confirmDeleteSourceId === id) {
      try {
        await deleteSavedSource(id);
        confirmDeleteSourceId = null;
        if (editingSourceId === id) cancelSourceForm();
        await loadSources();
      } catch (e) {
        sourcesError = e instanceof Error ? e.message : 'Failed to delete source';
      }
    } else {
      confirmDeleteSourceId = id;
    }
  }

  // --- Backend form helpers ---

  function resetBackendForm() {
    backendFormName = '';
    backendFormCredentials = {
      type: 'local',
      local: { path: '' }
    };
    backendFormSaving = false;
    backendFormError = '';
  }

  function openAddBackend() {
    editingBackendId = null;
    resetBackendForm();
    addingBackend = true;
  }

  function cancelBackendForm() {
    addingBackend = false;
    editingBackendId = null;
    resetBackendForm();
  }

  async function openEditBackend(id: string) {
    addingBackend = false;
    editingBackendId = id;
    resetBackendForm();
    try {
      const backend = await getSavedBackend(id);
      backendFormName = backend.name;
      backendFormCredentials = {
        type: backend.credentials.type,
        local: backend.credentials.local ? { ...backend.credentials.local } : undefined,
        smb: backend.credentials.smb ? { ...backend.credentials.smb, password: '' } : undefined,
        s3: backend.credentials.s3 ? { ...backend.credentials.s3, secret_key: '' } : undefined,
        webdav: backend.credentials.webdav ? { ...backend.credentials.webdav, password: '' } : undefined,
        azure: backend.credentials.azure ? { ...backend.credentials.azure, connection_string: '' } : undefined,
        dropbox: backend.credentials.dropbox ? { ...backend.credentials.dropbox, access_token: '', app_secret: '' } : undefined,
        gdrive: backend.credentials.gdrive ? { ...backend.credentials.gdrive, credentials: '' } : undefined,
        sftp: backend.credentials.sftp ? { ...backend.credentials.sftp, password: '', private_key: '' } : undefined
      };
    } catch {
      // Fall back to sanitized data
      const backend = backends.find((b) => b.id === id);
      if (backend) {
        backendFormName = backend.name;
        backendFormCredentials = {
          type: backend.credentials.type,
          local: backend.credentials.local ? { ...backend.credentials.local } : undefined,
          smb: backend.credentials.smb ? { ...backend.credentials.smb, password: '' } : undefined,
          s3: backend.credentials.s3 ? { ...backend.credentials.s3, secret_key: '' } : undefined,
          webdav: backend.credentials.webdav ? { ...backend.credentials.webdav, password: '' } : undefined,
          azure: backend.credentials.azure ? { ...backend.credentials.azure, connection_string: '' } : undefined,
          dropbox: backend.credentials.dropbox ? { ...backend.credentials.dropbox, access_token: '', app_secret: '' } : undefined,
          gdrive: backend.credentials.gdrive ? { ...backend.credentials.gdrive, credentials: '' } : undefined,
          sftp: backend.credentials.sftp ? { ...backend.credentials.sftp, password: '', private_key: '' } : undefined
        };
      }
    }
  }

  function buildBackendCredentials(): Credentials | null {
    const credentials = backendFormCredentials;
    if (credentials.type === 'local' && credentials.local) {
      if (!credentials.local.path.trim()) {
        backendFormError = 'Backup path is required';
        return null;
      }
      return { type: 'local', local: { path: credentials.local.path.trim() } };
    }
    
    if (credentials.type === 'smb' && credentials.smb) {
      if (!credentials.smb.host.trim() || !credentials.smb.share.trim()) {
        backendFormError = 'Host and share are required';
        return null;
      }
      return {
        type: 'smb',
        smb: {
          host: credentials.smb.host.trim(),
          share: credentials.smb.share.trim(),
          username: credentials.smb.username.trim(),
          password: credentials.smb.password,
          port: credentials.smb.port,
          path: credentials.smb.path.trim()
        }
      };
    }
    
    if (credentials.type === 's3' && credentials.s3) {
      if (!credentials.s3.bucket.trim()) {
        backendFormError = 'Bucket name is required';
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
        backendFormError = 'WebDAV URL is required';
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
        backendFormError = 'Container name is required';
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
        backendFormError = 'App Key is required';
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
        backendFormError = 'Folder ID is required';
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
        backendFormError = 'Host and user are required';
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

  async function handleSaveBackend() {
    backendFormError = '';
    if (!backendFormName.trim()) {
      backendFormError = 'Name is required';
      return;
    }
    const creds = buildBackendCredentials();
    if (!creds) return;

    backendFormSaving = true;
    try {
      const payload = { name: backendFormName.trim(), credentials: creds };
      if (editingBackendId) {
        await updateSavedBackend(editingBackendId, payload);
      } else {
        await createSavedBackend(payload);
      }
      cancelBackendForm();
      await loadBackends();
    } catch (e) {
      backendFormError = e instanceof Error ? e.message : 'Failed to save backend';
    } finally {
      backendFormSaving = false;
    }
  }

  async function handleDeleteBackend(id: string) {
    if (confirmDeleteBackendId === id) {
      try {
        await deleteSavedBackend(id);
        confirmDeleteBackendId = null;
        if (editingBackendId === id) cancelBackendForm();
        await loadBackends();
      } catch (e) {
        backendsError = e instanceof Error ? e.message : 'Failed to delete backend';
      }
    } else {
      confirmDeleteBackendId = id;
    }
  }

  function sourceDisplayUrl(source: SavedSource): string {
    if (source.type === 'git' && source.git_config) {
      return source.git_config.repo_url;
    }
    if (source.type === 'portainer' && source.portainer_config) {
      return source.portainer_config.portainer_url;
    }
    return '';
  }

  function backendDisplayInfo(backend: SavedBackend): string {
    if (backend.credentials.type === 'local' && backend.credentials.local) {
      return backend.credentials.local.path;
    }
    if (backend.credentials.type === 'smb' && backend.credentials.smb) {
      const smb = backend.credentials.smb;
      return `//${smb.host}/${smb.share}`;
    }
    if (backend.credentials.type === 's3' && backend.credentials.s3) {
      return `s3://${backend.credentials.s3.bucket}`;
    }
    if (backend.credentials.type === 'webdav' && backend.credentials.webdav) {
      return backend.credentials.webdav.url;
    }
    if (backend.credentials.type === 'azure' && backend.credentials.azure) {
      return `azure://${backend.credentials.azure.container}`;
    }
    if (backend.credentials.type === 'dropbox' && backend.credentials.dropbox) {
      return `dropbox://${backend.credentials.dropbox.remote_path || '/'}`;
    }
    if (backend.credentials.type === 'gdrive' && backend.credentials.gdrive) {
      return `gdrive://${backend.credentials.gdrive.folder_id}`;
    }
    if (backend.credentials.type === 'sftp' && backend.credentials.sftp) {
      const sftp = backend.credentials.sftp;
      return `sftp://${sftp.user}@${sftp.host}:${sftp.port}${sftp.remote_path || '/'}`;
    }
    return '';
  }
</script>

<div class="max-w-4xl mx-auto p-8 animate-fade-in">
  <div class="mb-8">
    <h1 class="text-2xl font-light text-slate-100">Settings</h1>
    <p class="text-sm text-slate-500 mt-1">Manage your saved sources and storage backends.</p>
  </div>

  <!-- Tabs -->
  <div class="flex border-b border-slate-700/50 mb-6">
    <button
      onclick={() => activeTab = 'sources'}
      class="px-4 py-2.5 text-sm font-medium border-b-2 transition-colors {activeTab === 'sources' ? 'border-indigo-400 text-indigo-400' : 'border-transparent text-slate-500 hover:text-slate-300 hover:border-slate-500'}"
    >
      Sources
    </button>
    <button
      onclick={() => activeTab = 'backends'}
      class="px-4 py-2.5 text-sm font-medium border-b-2 transition-colors {activeTab === 'backends' ? 'border-indigo-400 text-indigo-400' : 'border-transparent text-slate-500 hover:text-slate-300 hover:border-slate-500'}"
    >
      Backends
    </button>
  </div>

  <!-- Sources Tab -->
  {#if activeTab === 'sources'}
    {#if sourcesLoading}
      <div class="text-sm text-slate-500">Loading sources...</div>
    {:else if sourcesError}
      <div class="p-3 bg-red-500/10 border border-red-500/30 rounded-lg text-red-400 text-sm">{sourcesError}</div>
    {:else}
      <div class="space-y-3">
        {#each sources as source (source.id)}
          <div class="bg-slate-800/50 border border-slate-700/50 rounded-lg">
            <div class="flex items-center justify-between px-4 py-3">
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2">
                  <span class="font-medium text-slate-100">{source.name}</span>
                  <span class="px-2 py-0.5 text-xs font-medium rounded-full {source.type === 'git' ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/30' : 'bg-violet-500/10 text-violet-400 border border-violet-500/30'}">
                    {source.type === 'git' ? 'Git' : 'Portainer'}
                  </span>
                </div>
                <div class="text-sm text-slate-400 truncate mt-0.5">{sourceDisplayUrl(source)}</div>
              </div>
              <div class="flex items-center gap-2 ml-4 shrink-0">
                <button
                  onclick={() => openEditSource(source.id)}
                  class="px-3 py-1.5 text-xs font-medium text-slate-400 border border-slate-600 rounded-lg hover:bg-slate-700/50 transition-colors"
                >
                  Edit
                </button>
                <button
                  onclick={() => handleDeleteSource(source.id)}
                  class="px-3 py-1.5 text-xs font-medium rounded-lg transition-colors {confirmDeleteSourceId === source.id ? 'text-white bg-red-600 hover:bg-red-500' : 'text-red-400 border border-red-500/30 hover:bg-red-500/10'}"
                >
                  {confirmDeleteSourceId === source.id ? 'Confirm' : 'Delete'}
                </button>
              </div>
            </div>

            <!-- Edit form for this source -->
            {#if editingSourceId === source.id}
              <div class="border-t border-slate-700/50 px-4 py-4 bg-slate-900/50 rounded-b-lg space-y-4">
                <div>
                  <label class="block text-sm font-medium text-slate-300 mb-1">Name</label>
                  <input type="text" bind:value={sourceFormName} class="w-full px-3 py-2 bg-slate-900 border border-slate-600 rounded-lg text-sm text-slate-100 focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500" />
                </div>

                <div>
                  <p class="text-sm font-medium text-slate-300 mb-2">Type</p>
                  <div class="flex gap-3">
                    <button
                      onclick={() => sourceFormType = 'git'}
                      class="flex-1 py-2 px-4 rounded-lg border-2 text-sm font-medium transition-colors {sourceFormType === 'git' ? 'border-indigo-500 bg-indigo-500/10 text-indigo-400' : 'border-slate-600 text-slate-400 hover:border-slate-500'}"
                    >Git</button>
                    <button
                      onclick={() => sourceFormType = 'portainer'}
                      class="flex-1 py-2 px-4 rounded-lg border-2 text-sm font-medium transition-colors {sourceFormType === 'portainer' ? 'border-indigo-500 bg-indigo-500/10 text-indigo-400' : 'border-slate-600 text-slate-400 hover:border-slate-500'}"
                    >Portainer</button>
                  </div>
                </div>

                {#if sourceFormType === 'git'}
                  <div>
                    <label class="block text-sm font-medium text-slate-300 mb-1">Repository URL</label>
                    <input type="text" bind:value={sourceFormRepoUrl} class="w-full px-3 py-2 bg-slate-900 border border-slate-600 rounded-lg text-sm text-slate-100 focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500" placeholder="https://github.com/user/repo.git" />
                  </div>
                  <div>
                    <label class="block text-sm font-medium text-slate-300 mb-1">Branch</label>
                    <input type="text" bind:value={sourceFormBranch} class="w-full px-3 py-2 bg-slate-900 border border-slate-600 rounded-lg text-sm text-slate-100 focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500" placeholder="main" />
                  </div>
                  <div>
                    <span class="block text-sm font-medium text-slate-300 mb-2">Authentication</span>
                    <div class="flex gap-3">
                      <button onclick={() => sourceFormAuthType = 'none'} class="px-3 py-1.5 text-sm rounded-lg border transition-colors {sourceFormAuthType === 'none' ? 'bg-indigo-500/10 border-indigo-500/30 text-indigo-400' : 'border-slate-600 text-slate-400 hover:bg-slate-700/50'}">None</button>
                      <button onclick={() => sourceFormAuthType = 'token'} class="px-3 py-1.5 text-sm rounded-lg border transition-colors {sourceFormAuthType === 'token' ? 'bg-indigo-500/10 border-indigo-500/30 text-indigo-400' : 'border-slate-600 text-slate-400 hover:bg-slate-700/50'}">Token</button>
                      <button onclick={() => sourceFormAuthType = 'ssh'} class="px-3 py-1.5 text-sm rounded-lg border transition-colors {sourceFormAuthType === 'ssh' ? 'bg-indigo-500/10 border-indigo-500/30 text-indigo-400' : 'border-slate-600 text-slate-400 hover:bg-slate-700/50'}">SSH Key</button>
                    </div>
                  </div>
                  {#if sourceFormAuthType === 'token'}
                    <div>
                      <label class="block text-sm font-medium text-slate-300 mb-1">Access Token</label>
                      <input type="password" bind:value={sourceFormAuthToken} class="w-full px-3 py-2 bg-slate-900 border border-slate-600 rounded-lg text-sm text-slate-100 focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500" placeholder="ghp_... or personal access token" />
                    </div>
                  {/if}
                  {#if sourceFormAuthType === 'ssh'}
                    <div>
                      <label class="block text-sm font-medium text-slate-300 mb-1">SSH Private Key</label>
                      <textarea bind:value={sourceFormSshKey} rows="5" class="w-full px-3 py-2 bg-slate-900 border border-slate-600 rounded-lg font-mono text-xs text-slate-100 focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500" placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"></textarea>
                    </div>
                  {/if}
                {:else}
                  <div>
                    <label class="block text-sm font-medium text-slate-300 mb-1">Portainer URL</label>
                    <input type="url" bind:value={sourceFormPortainerUrl} class="w-full px-3 py-2 bg-slate-900 border border-slate-600 rounded-lg text-sm text-slate-100 focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500" placeholder="https://portainer.example.com" />
                  </div>
                  <div>
                    <label class="block text-sm font-medium text-slate-300 mb-1">API Key</label>
                    <input type="password" bind:value={sourceFormApiKey} class="w-full px-3 py-2 bg-slate-900 border border-slate-600 rounded-lg text-sm text-slate-100 focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500" placeholder="ptr_..." />
                  </div>
                {/if}

                {#if sourceFormError}
                  <div class="p-3 bg-red-500/10 border border-red-500/30 rounded-lg text-red-400 text-sm">{sourceFormError}</div>
                {/if}

                <div class="flex gap-3">
                  <button onclick={cancelSourceForm} class="flex-1 py-2 px-4 border border-slate-600 text-slate-300 hover:bg-slate-700/50 rounded-lg font-medium text-sm transition-colors">
                    Cancel
                  </button>
                  <button onclick={handleSaveSource} disabled={sourceFormSaving} class="flex-1 py-2 px-4 bg-gradient-to-r from-indigo-500 to-violet-500 hover:from-indigo-400 hover:to-violet-400 disabled:opacity-50 text-white rounded-lg font-medium text-sm transition-all">
                    {sourceFormSaving ? 'Saving...' : 'Save'}
                  </button>
                </div>
              </div>
            {/if}
          </div>
        {/each}

        {#if sources.length === 0 && !addingSource}
          <div class="text-center py-8 text-slate-500 text-sm">
            No saved sources yet. Add one to reuse it across projects.
          </div>
        {/if}

        <!-- Add source form -->
        {#if addingSource}
          <div class="bg-slate-800/50 border border-indigo-500/30 rounded-lg px-4 py-4 space-y-4">
            <p class="text-sm font-semibold text-slate-100">New Source</p>

            <div>
              <label class="block text-sm font-medium text-slate-300 mb-1">Name</label>
              <input type="text" bind:value={sourceFormName} class="w-full px-3 py-2 bg-slate-900 border border-slate-600 rounded-lg text-sm text-slate-100 focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500" placeholder="e.g. My Git Repo" />
            </div>

            <div>
              <p class="text-sm font-medium text-slate-300 mb-2">Type</p>
              <div class="flex gap-3">
                <button
                  onclick={() => sourceFormType = 'git'}
                  class="flex-1 py-2 px-4 rounded-lg border-2 text-sm font-medium transition-colors {sourceFormType === 'git' ? 'border-indigo-500 bg-indigo-500/10 text-indigo-400' : 'border-slate-600 text-slate-400 hover:border-slate-500'}"
                >Git</button>
                <button
                  onclick={() => sourceFormType = 'portainer'}
                  class="flex-1 py-2 px-4 rounded-lg border-2 text-sm font-medium transition-colors {sourceFormType === 'portainer' ? 'border-indigo-500 bg-indigo-500/10 text-indigo-400' : 'border-slate-600 text-slate-400 hover:border-slate-500'}"
                >Portainer</button>
              </div>
            </div>

            {#if sourceFormType === 'git'}
              <div>
                <label class="block text-sm font-medium text-slate-300 mb-1">Repository URL</label>
                <input type="text" bind:value={sourceFormRepoUrl} class="w-full px-3 py-2 bg-slate-900 border border-slate-600 rounded-lg text-sm text-slate-100 focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500" placeholder="https://github.com/user/repo.git" />
              </div>
              <div>
                <label class="block text-sm font-medium text-slate-300 mb-1">Branch</label>
                <input type="text" bind:value={sourceFormBranch} class="w-full px-3 py-2 bg-slate-900 border border-slate-600 rounded-lg text-sm text-slate-100 focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500" placeholder="main" />
              </div>
              <div>
                <span class="block text-sm font-medium text-slate-300 mb-2">Authentication</span>
                <div class="flex gap-3">
                  <button onclick={() => sourceFormAuthType = 'none'} class="px-3 py-1.5 text-sm rounded-lg border transition-colors {sourceFormAuthType === 'none' ? 'bg-indigo-500/10 border-indigo-500/30 text-indigo-400' : 'border-slate-600 text-slate-400 hover:bg-slate-700/50'}">None</button>
                  <button onclick={() => sourceFormAuthType = 'token'} class="px-3 py-1.5 text-sm rounded-lg border transition-colors {sourceFormAuthType === 'token' ? 'bg-indigo-500/10 border-indigo-500/30 text-indigo-400' : 'border-slate-600 text-slate-400 hover:bg-slate-700/50'}">Token</button>
                  <button onclick={() => sourceFormAuthType = 'ssh'} class="px-3 py-1.5 text-sm rounded-lg border transition-colors {sourceFormAuthType === 'ssh' ? 'bg-indigo-500/10 border-indigo-500/30 text-indigo-400' : 'border-slate-600 text-slate-400 hover:bg-slate-700/50'}">SSH Key</button>
                </div>
              </div>
              {#if sourceFormAuthType === 'token'}
                <div>
                  <label class="block text-sm font-medium text-slate-300 mb-1">Access Token</label>
                  <input type="password" bind:value={sourceFormAuthToken} class="w-full px-3 py-2 bg-slate-900 border border-slate-600 rounded-lg text-sm text-slate-100 focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500" placeholder="ghp_... or personal access token" />
                </div>
              {/if}
              {#if sourceFormAuthType === 'ssh'}
                <div>
                  <label class="block text-sm font-medium text-slate-300 mb-1">SSH Private Key</label>
                  <textarea bind:value={sourceFormSshKey} rows="5" class="w-full px-3 py-2 bg-slate-900 border border-slate-600 rounded-lg font-mono text-xs text-slate-100 focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500" placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"></textarea>
                </div>
              {/if}
            {:else}
              <div>
                <label class="block text-sm font-medium text-slate-300 mb-1">Portainer URL</label>
                <input type="url" bind:value={sourceFormPortainerUrl} class="w-full px-3 py-2 bg-slate-900 border border-slate-600 rounded-lg text-sm text-slate-100 focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500" placeholder="https://portainer.example.com" />
              </div>
              <div>
                <label class="block text-sm font-medium text-slate-300 mb-1">API Key</label>
                <input type="password" bind:value={sourceFormApiKey} class="w-full px-3 py-2 bg-slate-900 border border-slate-600 rounded-lg text-sm text-slate-100 focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500" placeholder="ptr_..." />
              </div>
            {/if}

            {#if sourceFormError}
              <div class="p-3 bg-red-500/10 border border-red-500/30 rounded-lg text-red-400 text-sm">{sourceFormError}</div>
            {/if}

            <div class="flex gap-3">
              <button onclick={cancelSourceForm} class="flex-1 py-2 px-4 border border-slate-600 text-slate-300 hover:bg-slate-700/50 rounded-lg font-medium text-sm transition-colors">
                Cancel
              </button>
              <button onclick={handleSaveSource} disabled={sourceFormSaving} class="flex-1 py-2 px-4 bg-gradient-to-r from-indigo-500 to-violet-500 hover:from-indigo-400 hover:to-violet-400 disabled:opacity-50 text-white rounded-lg font-medium text-sm transition-all">
                {sourceFormSaving ? 'Saving...' : 'Add Source'}
              </button>
            </div>
          </div>
        {/if}
      </div>

      {#if !addingSource && editingSourceId === null}
        <button onclick={openAddSource} class="mt-4 w-full py-2 px-4 border-2 border-dashed border-slate-600 text-slate-500 hover:border-indigo-500/50 hover:text-indigo-400 rounded-lg text-sm font-medium transition-colors">
          + Add Source
        </button>
      {/if}
    {/if}
  {/if}

  <!-- Backends Tab -->
  {#if activeTab === 'backends'}
    {#if backendsLoading}
      <div class="text-sm text-slate-500">Loading backends...</div>
    {:else if backendsError}
      <div class="p-3 bg-red-500/10 border border-red-500/30 rounded-lg text-red-400 text-sm">{backendsError}</div>
    {:else}
      <div class="space-y-3">
        {#each backends as backend (backend.id)}
          <div class="bg-slate-800/50 border border-slate-700/50 rounded-lg">
            <div class="flex items-center justify-between px-4 py-3">
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2">
                  <span class="font-medium text-slate-100">{backend.name}</span>
                  <span class="px-2 py-0.5 text-xs font-medium rounded-full {backend.credentials.type === 'local' ? 'bg-slate-700/50 text-slate-400 border border-slate-600' : 'bg-amber-500/10 text-amber-400 border border-amber-500/30'}">
                    {backend.credentials.type.toUpperCase()}
                  </span>
                </div>
                <div class="text-sm text-slate-400 truncate mt-0.5 font-mono">{backendDisplayInfo(backend)}</div>
              </div>
              <div class="flex items-center gap-2 ml-4 shrink-0">
                <button
                  onclick={() => openEditBackend(backend.id)}
                  class="px-3 py-1.5 text-xs font-medium text-slate-400 border border-slate-600 rounded-lg hover:bg-slate-700/50 transition-colors"
                >
                  Edit
                </button>
                <button
                  onclick={() => handleDeleteBackend(backend.id)}
                  class="px-3 py-1.5 text-xs font-medium rounded-lg transition-colors {confirmDeleteBackendId === backend.id ? 'text-white bg-red-600 hover:bg-red-500' : 'text-red-400 border border-red-500/30 hover:bg-red-500/10'}"
                >
                  {confirmDeleteBackendId === backend.id ? 'Confirm' : 'Delete'}
                </button>
              </div>
            </div>

            <!-- Edit form for this backend -->
            {#if editingBackendId === backend.id}
              <div class="border-t border-slate-700/50 px-4 py-4 bg-slate-900/50 rounded-b-lg space-y-4">
                <div>
                  <label class="block text-sm font-medium text-slate-300 mb-1">Name</label>
                  <input type="text" bind:value={backendFormName} class="w-full px-3 py-2 bg-slate-900 border border-slate-600 rounded-lg text-sm text-slate-100 focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500" />
                </div>

                <div>
                  <label class="block text-sm font-medium text-slate-300 mb-1">Storage Type</label>
                  <select bind:value={backendFormCredentials.type} class="w-full px-3 py-2 bg-slate-900 border border-slate-600 rounded-lg text-sm text-slate-100 focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500">
                    <option value="local">Local Path</option>
                    <option value="smb">SMB / Network Share</option>
                    <option value="s3">AWS S3 (Beta)</option>
                    <option value="webdav">WebDAV (Beta)</option>
                    <option value="azure">Azure Blob (Beta)</option>
                    <option value="dropbox">Dropbox (Beta)</option>
                    <option value="gdrive">Google Drive (Beta)</option>
                    <option value="sftp">SFTP / SSH (Beta)</option>
                  </select>
                </div>

                <div class="mt-4">
                  <CredentialFields bind:credentials={backendFormCredentials} isEdit={true} />
                </div>

                {#if backendFormError}
                  <div class="p-3 bg-red-500/10 border border-red-500/30 rounded-lg text-red-400 text-sm">{backendFormError}</div>
                {/if}

                <div class="flex gap-3">
                  <button onclick={cancelBackendForm} class="flex-1 py-2 px-4 border border-slate-600 text-slate-300 hover:bg-slate-700/50 rounded-lg font-medium text-sm transition-colors">
                    Cancel
                  </button>
                  <button onclick={handleSaveBackend} disabled={backendFormSaving} class="flex-1 py-2 px-4 bg-gradient-to-r from-indigo-500 to-violet-500 hover:from-indigo-400 hover:to-violet-400 disabled:opacity-50 text-white rounded-lg font-medium text-sm transition-all">
                    {backendFormSaving ? 'Saving...' : 'Save'}
                  </button>
                </div>
              </div>
            {/if}
          </div>
        {/each}

        {#if backends.length === 0 && !addingBackend}
          <div class="text-center py-8 text-slate-500 text-sm">
            No saved backends yet. Add one to reuse it across projects.
          </div>
        {/if}

        <!-- Add backend form -->
        {#if addingBackend}
          <div class="bg-slate-800/50 border border-indigo-500/30 rounded-lg px-4 py-4 space-y-4">
            <p class="text-sm font-semibold text-slate-100">New Backend</p>

            <div>
              <label class="block text-sm font-medium text-slate-300 mb-1">Name</label>
              <input type="text" bind:value={backendFormName} class="w-full px-3 py-2 bg-slate-900 border border-slate-600 rounded-lg text-sm text-slate-100 focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500" placeholder="e.g. NAS Backup" />
            </div>

            <div>
              <label class="block text-sm font-medium text-slate-300 mb-1">Storage Type</label>
              <select bind:value={backendFormCredentials.type} class="w-full px-3 py-2 bg-slate-900 border border-slate-600 rounded-lg text-sm text-slate-100 focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500">
                <option value="local">Local Path</option>
                <option value="smb">SMB / Network Share</option>
                <option value="s3">AWS S3 (Beta)</option>
                <option value="webdav">WebDAV (Beta)</option>
                <option value="azure">Azure Blob (Beta)</option>
                <option value="dropbox">Dropbox (Beta)</option>
                <option value="gdrive">Google Drive (Beta)</option>
                <option value="sftp">SFTP / SSH (Beta)</option>
              </select>
            </div>

            <div class="mt-4">
              <CredentialFields bind:credentials={backendFormCredentials} isEdit={false} />
            </div>

            {#if backendFormError}
              <div class="p-3 bg-red-500/10 border border-red-500/30 rounded-lg text-red-400 text-sm">{backendFormError}</div>
            {/if}

            <div class="flex gap-3">
              <button onclick={cancelBackendForm} class="flex-1 py-2 px-4 border border-slate-600 text-slate-300 hover:bg-slate-700/50 rounded-lg font-medium text-sm transition-colors">
                Cancel
              </button>
              <button onclick={handleSaveBackend} disabled={backendFormSaving} class="flex-1 py-2 px-4 bg-gradient-to-r from-indigo-500 to-violet-500 hover:from-indigo-400 hover:to-violet-400 disabled:opacity-50 text-white rounded-lg font-medium text-sm transition-all">
                {backendFormSaving ? 'Saving...' : 'Add Backend'}
              </button>
            </div>
          </div>
        {/if}
      </div>

      {#if !addingBackend && editingBackendId === null}
        <button onclick={openAddBackend} class="mt-4 w-full py-2 px-4 border-2 border-dashed border-slate-600 text-slate-500 hover:border-indigo-500/50 hover:text-indigo-400 rounded-lg text-sm font-medium transition-colors">
          + Add Backend
        </button>
      {/if}
    {/if}
  {/if}
</div>
