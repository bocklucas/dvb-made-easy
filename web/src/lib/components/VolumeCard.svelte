<script lang="ts">
  import type { BackupFile, Volume } from '$lib/types';
  import { setVolumePassphrase, deleteVolumePassphrase } from '$lib/api';
  import BackupList from './BackupList.svelte';

  let {
    projectId,
    volume,
    onRestore,
    onPassphraseChange
  }: {
    projectId: string;
    volume: Volume;
    onRestore: (volumeName: string, backup: BackupFile) => void;
    onPassphraseChange?: () => void;
  } = $props();

  let expanded = $state(false);
  let showPassphraseForm = $state(false);
  let passphraseInput = $state('');
  let passphraseLoading = $state(false);
  let passphraseError = $state<string | null>(null);

  let hasPassphrase = $derived(!!volume.passphrase);

  async function handleSetPassphrase() {
    if (!passphraseInput.trim()) return;
    passphraseLoading = true;
    passphraseError = null;
    try {
      await setVolumePassphrase(projectId, volume.name, passphraseInput.trim());
      passphraseInput = '';
      showPassphraseForm = false;
      onPassphraseChange?.();
    } catch (e) {
      passphraseError = e instanceof Error ? e.message : 'Failed to save passphrase';
    } finally {
      passphraseLoading = false;
    }
  }

  async function handleClearPassphrase() {
    passphraseLoading = true;
    passphraseError = null;
    try {
      await deleteVolumePassphrase(projectId, volume.name);
      onPassphraseChange?.();
    } catch (e) {
      passphraseError = e instanceof Error ? e.message : 'Failed to clear passphrase';
    } finally {
      passphraseLoading = false;
    }
  }

  function cancelPassphrase() {
    showPassphraseForm = false;
    passphraseInput = '';
    passphraseError = null;
  }
</script>

<div class="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
  <div class="px-4 py-3 flex items-center justify-between">
    <div class="min-w-0 flex-1">
      <div class="flex items-center gap-2">
        <span class="font-medium text-gray-900">{volume.name}</span>
        {#if hasPassphrase}
          <span
            class="inline-flex items-center gap-1 text-xs text-green-700 bg-green-50 border border-green-200 rounded px-1.5 py-0.5"
            title="Passphrase stored"
          >
            <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
            </svg>
            Passphrase stored
          </span>
        {:else}
          <span
            class="inline-flex items-center gap-1 text-xs text-gray-400 bg-gray-50 border border-gray-200 rounded px-1.5 py-0.5"
            title="No passphrase stored"
          >
            <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M8 11V7a4 4 0 018 0m-4 8v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2z" />
            </svg>
            No passphrase
          </span>
        {/if}
      </div>
      <div class="text-sm text-gray-500 mt-0.5">
        {volume.compose_service} → <span class="font-mono">{volume.compose_mount_path}</span>
      </div>
    </div>
    <div class="ml-4 shrink-0 flex items-center gap-2">
      {#if hasPassphrase}
        <button
          onclick={() => (showPassphraseForm = !showPassphraseForm)}
          class="text-xs px-2 py-1 border border-gray-300 rounded hover:bg-gray-50 transition-colors text-gray-600"
          disabled={passphraseLoading}
        >
          Change
        </button>
        <button
          onclick={handleClearPassphrase}
          class="text-xs px-2 py-1 border border-red-200 rounded hover:bg-red-50 transition-colors text-red-600"
          disabled={passphraseLoading}
        >
          {passphraseLoading ? '...' : 'Clear'}
        </button>
      {:else}
        <button
          onclick={() => (showPassphraseForm = !showPassphraseForm)}
          class="text-xs px-2 py-1 border border-gray-300 rounded hover:bg-gray-50 transition-colors text-gray-600"
        >
          Set Passphrase
        </button>
      {/if}
      <button
        onclick={() => (expanded = !expanded)}
        class="text-sm px-3 py-1.5 border border-gray-300 rounded hover:bg-gray-50 transition-colors text-gray-700"
      >
        {expanded ? 'Hide Backups' : 'Browse Backups'}
      </button>
    </div>
  </div>

  {#if showPassphraseForm}
    <div class="border-t border-gray-100 bg-gray-50 px-4 py-3">
      <label for="passphrase-{volume.name}" class="block text-sm font-medium text-gray-700 mb-1">
        {hasPassphrase ? 'Change Passphrase' : 'Set Passphrase'}
      </label>
      <div class="flex gap-2">
        <input
          id="passphrase-{volume.name}"
          type="password"
          bind:value={passphraseInput}
          placeholder="Enter passphrase"
          class="flex-1 border border-gray-300 rounded-lg px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
        <button
          onclick={handleSetPassphrase}
          disabled={passphraseLoading || !passphraseInput.trim()}
          class="px-3 py-1.5 text-sm text-white bg-blue-600 hover:bg-blue-500 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {passphraseLoading ? 'Saving...' : 'Save'}
        </button>
        <button
          onclick={cancelPassphrase}
          class="px-3 py-1.5 text-sm text-gray-700 border border-gray-300 rounded-lg hover:bg-white transition-colors"
        >
          Cancel
        </button>
      </div>
      {#if passphraseError}
        <div class="mt-2 text-sm text-red-600">{passphraseError}</div>
      {/if}
    </div>
  {/if}

  {#if expanded}
    <div class="border-t border-gray-200 px-4 py-2">
      <BackupList
        {projectId}
        volumeName={volume.name}
        onRestore={(backup) => onRestore(volume.name, backup)}
      />
    </div>
  {/if}
</div>
