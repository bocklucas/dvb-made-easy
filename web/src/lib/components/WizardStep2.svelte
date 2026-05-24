<script lang="ts">
  import type { Credentials, SavedBackend } from '$lib/types';
  import { saveCredentials, listSavedBackends, createSavedBackend } from '$lib/api';
  import { onMount } from 'svelte';

  interface Props {
    projectId: string;
    onComplete: (backendType: 'local' | 'smb') => void;
    onBack: () => void;
  }

  let { projectId, onComplete, onBack }: Props = $props();

  let backendType = $state<'local' | 'smb'>('local');

  // Saved backends
  let savedBackends = $state<SavedBackend[]>([]);
  let selectedSavedBackendId = $state('');
  let saveAsBackend = $state(false);
  let savedBackendName = $state('');

  // Local fields
  let localPath = $state('');

  // SMB fields
  let smbHost = $state('');
  let smbShare = $state('');
  let smbPath = $state('');
  let smbUsername = $state('');
  let smbPassword = $state('');
  let smbPort = $state(445);

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

  function selectType(type: 'local' | 'smb') {
    backendType = type;
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

    backendType = backend.credentials.type;
    if (backend.credentials.type === 'local' && backend.credentials.local) {
      localPath = backend.credentials.local.path;
    } else if (backend.credentials.type === 'smb' && backend.credentials.smb) {
      smbHost = backend.credentials.smb.host;
      smbShare = backend.credentials.smb.share;
      smbPath = backend.credentials.smb.path ?? '';
      smbUsername = backend.credentials.smb.username ?? '';
      smbPassword = backend.credentials.smb.password ?? '';
      smbPort = backend.credentials.smb.port ?? 445;
    }

    // Saved backends were already tested when created, so apply directly
    applyDirectly(backend.credentials);
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

  async function handleTestConnection() {
    error = '';
    success = false;

    let creds: Credentials;
    if (backendType === 'local') {
      if (!localPath.trim()) {
        error = 'Enter a local path';
        return;
      }
      creds = { type: 'local', local: { path: localPath.trim() } };
    } else {
      if (!smbHost.trim() || !smbShare.trim()) {
        error = 'Host and share are required';
        return;
      }
      creds = {
        type: 'smb',
        smb: {
          host: smbHost.trim(),
          share: smbShare.trim(),
          path: smbPath.trim(),
          username: smbUsername.trim(),
          password: smbPassword,
          port: smbPort
        }
      };
    }

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
          // non-fatal: credentials were saved, just couldn't save the backend preset
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
      <label for="saved-backend" class="block text-sm font-medium text-gray-700 mb-2">
        Use a saved backend
      </label>
      <select
        id="saved-backend"
        bind:value={selectedSavedBackendId}
        onchange={applySavedBackend}
        class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
      >
        <option value="">Configure manually...</option>
        {#each savedBackends as backend}
          <option value={backend.id}>{backend.name} ({backend.credentials.type})</option>
        {/each}
      </select>
    </div>
  {/if}

  <div>
    <p class="text-sm font-medium text-gray-700 mb-3">Storage Backend</p>
    <div class="flex gap-3">
      <button
        onclick={() => selectType('local')}
        class="flex-1 py-2 px-4 rounded-lg border-2 text-sm font-medium transition-colors {backendType === 'local' ? 'border-blue-500 bg-blue-50 text-blue-700' : 'border-gray-200 text-gray-600 hover:border-gray-300'}"
      >
        Local
      </button>
      <button
        onclick={() => selectType('smb')}
        class="flex-1 py-2 px-4 rounded-lg border-2 text-sm font-medium transition-colors {backendType === 'smb' ? 'border-blue-500 bg-blue-50 text-blue-700' : 'border-gray-200 text-gray-600 hover:border-gray-300'}"
      >
        SMB / Network Share
      </button>
    </div>
  </div>

  {#if backendType === 'local'}
    <div>
      <label for="local-path" class="block text-sm font-medium text-gray-700 mb-2">
        Backup Path
      </label>
      <input
        id="local-path"
        type="text"
        bind:value={localPath}
        class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
        placeholder="/mnt/backups"
      />
    </div>
  {:else}
    <div class="grid grid-cols-2 gap-4">
      <div>
        <label for="smb-host" class="block text-sm font-medium text-gray-700 mb-2">Host</label>
        <input
          id="smb-host"
          type="text"
          bind:value={smbHost}
          class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
          placeholder="192.168.1.10"
        />
      </div>
      <div>
        <label for="smb-share" class="block text-sm font-medium text-gray-700 mb-2">Share</label>
        <input
          id="smb-share"
          type="text"
          bind:value={smbShare}
          class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
          placeholder="backups"
        />
      </div>
      <div>
        <label for="smb-path" class="block text-sm font-medium text-gray-700 mb-2">Path</label>
        <input
          id="smb-path"
          type="text"
          bind:value={smbPath}
          class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
          placeholder="subfolder"
        />
      </div>
      <div>
        <label for="smb-port" class="block text-sm font-medium text-gray-700 mb-2">Port</label>
        <input
          id="smb-port"
          type="number"
          bind:value={smbPort}
          class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
          placeholder="445"
        />
      </div>
      <div>
        <label for="smb-username" class="block text-sm font-medium text-gray-700 mb-2">
          Username
        </label>
        <input
          id="smb-username"
          type="text"
          bind:value={smbUsername}
          class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
          placeholder="user"
        />
      </div>
      <div>
        <label for="smb-password" class="block text-sm font-medium text-gray-700 mb-2">
          Password
        </label>
        <input
          id="smb-password"
          type="password"
          bind:value={smbPassword}
          class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
          placeholder="••••••••"
        />
      </div>
    </div>
  {/if}

  {#if error}
    <div class="p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">{error}</div>
  {/if}

  {#if success}
    <div class="p-3 bg-green-50 border border-green-200 rounded-lg text-green-700 text-sm">
      Connection successful — credentials saved.
    </div>
  {/if}

  {#if !selectedSavedBackendId}
    <div class="flex items-start gap-3">
      <input
        id="save-backend"
        type="checkbox"
        bind:checked={saveAsBackend}
        class="mt-1 h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
      />
      <div class="flex-1">
        <label for="save-backend" class="text-sm font-medium text-gray-700">
          Save this backend for later
        </label>
        {#if saveAsBackend}
          <input
            type="text"
            bind:value={savedBackendName}
            placeholder="Backend name"
            class="mt-2 w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 text-sm"
          />
        {/if}
      </div>
    </div>
  {/if}

  <button
    onclick={handleTestConnection}
    disabled={loading}
    class="w-full py-2 px-4 border border-blue-600 text-blue-600 hover:bg-blue-50 disabled:opacity-50 rounded-lg font-medium transition-colors"
  >
    {loading ? 'Testing...' : 'Test Connection'}
  </button>

  <div class="flex gap-3">
    <button
      onclick={onBack}
      class="flex-1 py-2 px-4 border border-gray-300 text-gray-700 hover:bg-gray-50 rounded-lg font-medium transition-colors"
    >
      ← Back
    </button>
    <button
      onclick={() => onComplete(backendType)}
      disabled={!success}
      class="flex-1 py-2 px-4 bg-blue-600 hover:bg-blue-500 disabled:bg-blue-300 text-white rounded-lg font-medium transition-colors"
    >
      Next
    </button>
  </div>
</div>
