<script lang="ts">
  import type { CredentialResponse, Credentials, SavedBackend } from '$lib/types';
  import { updateCredentials, listSavedBackends, createSavedBackend } from '$lib/api';
  import { onMount } from 'svelte';

  interface Props {
    projectId: string;
    existing: CredentialResponse;
    onClose: () => void;
    onSaved: () => void;
  }

  let { projectId, existing, onClose, onSaved }: Props = $props();

  // eslint-disable-next-line svelte/no-unused-svelte-ignore
  // svelte-ignore state_referenced_locally
  let backendType = $state<'local' | 'smb'>(existing.type);

  // Saved backends
  let savedBackends = $state<SavedBackend[]>([]);
  let selectedSavedBackendId = $state('');
  let saveAsBackend = $state(false);
  let savedBackendName = $state('');

  // Local fields — pre-filled from existing
  // eslint-disable-next-line svelte/no-unused-svelte-ignore
  // svelte-ignore state_referenced_locally
  let localPath = $state(existing.local?.path ?? '');

  // SMB fields — pre-filled from existing (password always blank)
  // eslint-disable-next-line svelte/no-unused-svelte-ignore
  // svelte-ignore state_referenced_locally
  let smbHost = $state(existing.smb?.host ?? '');
  // eslint-disable-next-line svelte/no-unused-svelte-ignore
  // svelte-ignore state_referenced_locally
  let smbShare = $state(existing.smb?.share ?? '');
  // eslint-disable-next-line svelte/no-unused-svelte-ignore
  // svelte-ignore state_referenced_locally
  let smbPath = $state(existing.smb?.path ?? '');
  // eslint-disable-next-line svelte/no-unused-svelte-ignore
  // svelte-ignore state_referenced_locally
  let smbUsername = $state(existing.smb?.username ?? '');
  let smbPassword = $state('');
  // eslint-disable-next-line svelte/no-unused-svelte-ignore
  // svelte-ignore state_referenced_locally
  let smbPort = $state(existing.smb?.port ?? 445);

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

  function selectType(type: 'local' | 'smb') {
    backendType = type;
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
  }

  function buildCreds(): Credentials | null {
    if (backendType === 'local') {
      if (!localPath.trim()) {
        error = 'Enter a local path';
        return null;
      }
      return { type: 'local', local: { path: localPath.trim() } };
    } else {
      if (!smbHost.trim() || !smbShare.trim()) {
        error = 'Host and share are required';
        return null;
      }
      return {
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
  <div class="bg-white rounded-xl shadow-xl w-full max-w-lg max-h-[90vh] flex flex-col">
    <!-- Header -->
    <div class="flex items-center justify-between px-5 py-4 border-b border-gray-200">
      <h2 id="storage-edit-title" class="text-lg font-semibold text-gray-900">Edit Storage</h2>
      <button
        onclick={onClose}
        class="text-gray-400 hover:text-gray-600 transition-colors"
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
          <label for="edit-saved-backend" class="block text-sm font-medium text-gray-700 mb-2">
            Use a saved backend
          </label>
          <select
            id="edit-saved-backend"
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

      <!-- Type selector -->
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
          <label for="edit-local-path" class="block text-sm font-medium text-gray-700 mb-2">
            Backup Path
          </label>
          <input
            id="edit-local-path"
            type="text"
            bind:value={localPath}
            class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            placeholder="/mnt/backups"
          />
        </div>
      {:else}
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label for="edit-smb-host" class="block text-sm font-medium text-gray-700 mb-2">
              Host
            </label>
            <input
              id="edit-smb-host"
              type="text"
              bind:value={smbHost}
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              placeholder="192.168.1.10"
            />
          </div>
          <div>
            <label for="edit-smb-share" class="block text-sm font-medium text-gray-700 mb-2">
              Share
            </label>
            <input
              id="edit-smb-share"
              type="text"
              bind:value={smbShare}
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              placeholder="backups"
            />
          </div>
          <div>
            <label for="edit-smb-path" class="block text-sm font-medium text-gray-700 mb-2">
              Path
            </label>
            <input
              id="edit-smb-path"
              type="text"
              bind:value={smbPath}
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              placeholder="subfolder"
            />
          </div>
          <div>
            <label for="edit-smb-port" class="block text-sm font-medium text-gray-700 mb-2">
              Port
            </label>
            <input
              id="edit-smb-port"
              type="number"
              bind:value={smbPort}
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              placeholder="445"
            />
          </div>
          <div>
            <label for="edit-smb-username" class="block text-sm font-medium text-gray-700 mb-2">
              Username
            </label>
            <input
              id="edit-smb-username"
              type="text"
              bind:value={smbUsername}
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              placeholder="user"
            />
          </div>
          <div>
            <label for="edit-smb-password" class="block text-sm font-medium text-gray-700 mb-2">
              Password
            </label>
            <input
              id="edit-smb-password"
              type="password"
              bind:value={smbPassword}
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              placeholder="Enter to change"
            />
          </div>
        </div>
      {/if}

      {#if error}
        <div class="p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">
          {error}
        </div>
      {/if}

      {#if testSuccess}
        <div class="p-3 bg-green-50 border border-green-200 rounded-lg text-green-700 text-sm">
          Connection successful.
        </div>
      {/if}
    </div>

    <!-- Footer -->
    <div class="flex items-center justify-end gap-2 px-5 py-4 border-t border-gray-100">
      <button
        onclick={onClose}
        class="px-4 py-2 text-sm text-gray-700 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
      >
        Cancel
      </button>
      <button
        onclick={handleTestConnection}
        disabled={testing || saving}
        class="px-4 py-2 text-sm border border-blue-600 text-blue-600 hover:bg-blue-50 disabled:opacity-50 rounded-lg font-medium transition-colors"
      >
        {testing ? 'Testing...' : 'Test Connection'}
      </button>
      <button
        onclick={handleSave}
        disabled={!testSuccess || saving}
        class="px-4 py-2 text-sm text-white bg-blue-600 hover:bg-blue-500 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {saving ? 'Saving...' : 'Save'}
      </button>
    </div>
  </div>
</div>
