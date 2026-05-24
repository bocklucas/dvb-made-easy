<script lang="ts">
  import { untrack } from 'svelte';
  import { goto } from '$app/navigation';
  import { startRestore } from '$lib/api';
  import type { BackupFile } from '$lib/types';

  let {
    projectId,
    volumeName,
    backup,
    storedPassphrase = '',
    onClose
  }: {
    projectId: string;
    volumeName: string;
    backup: BackupFile;
    storedPassphrase?: string;
    onClose: () => void;
  } = $props();

  let mode = $state<'new_volume' | 'full_stack'>('new_volume');
  let targetVolumeName = $state(untrack(() => volumeName));

  // True when the target name matches the source volume (will overwrite).
  let willOverwrite = $derived(targetVolumeName.trim() === volumeName);
  let passphrase = $state('');
  let loading = $state(false);
  let error = $state<string | null>(null);

  // Pre-fill passphrase from stored value when component mounts.
  $effect(() => {
    if (storedPassphrase && !passphrase) {
      passphrase = storedPassphrase;
    }
  });

  let usingStoredPassphrase = $derived(backup.is_encrypted && !!storedPassphrase && passphrase === storedPassphrase);

  async function handleStart() {
    loading = true;
    error = null;
    try {
      const req = {
        mode,
        backup_key: backup.key,
        ...(mode === 'new_volume' ? { target_volume_name: targetVolumeName } : {}),
        ...(backup.is_encrypted ? { passphrase } : {})
      };
      const resp = await startRestore(projectId, volumeName, req);
      goto(`/restore/${resp.restore_token}`);
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to start restore';
      loading = false;
    }
  }
</script>

<!-- Backdrop -->
<div
  class="fixed inset-0 bg-black/50 z-40 flex items-center justify-center"
  role="dialog"
  aria-modal="true"
>
  <!-- Click outside to close -->
  <button
    class="absolute inset-0 w-full h-full cursor-default"
    onclick={onClose}
    tabindex="-1"
    aria-label="Close modal"
  ></button>

  <!-- Modal card -->
  <div class="relative z-50 bg-white rounded-xl shadow-xl w-full max-w-md mx-4 p-6">
    <h2 class="text-lg font-semibold text-gray-900 mb-1">Confirm Restore</h2>
    <p class="text-sm text-gray-500 mb-4">
      Backup: <span class="font-mono text-gray-800 break-all">{backup.key}</span>
    </p>

    <!-- Mode selection -->
    <div class="grid grid-cols-2 gap-3 mb-4">
      <button
        onclick={() => (mode = 'new_volume')}
        class="border-2 rounded-lg p-3 text-left transition-colors {mode === 'new_volume'
          ? 'border-blue-500 bg-blue-50'
          : 'border-gray-200 hover:border-gray-300'}"
      >
        <div class="font-medium text-sm text-gray-900">New Volume</div>
        <div class="text-xs text-gray-500 mt-0.5">Restore into a new Docker volume</div>
      </button>
      <button
        onclick={() => (mode = 'full_stack')}
        class="border-2 rounded-lg p-3 text-left transition-colors {mode === 'full_stack'
          ? 'border-blue-500 bg-blue-50'
          : 'border-gray-200 hover:border-gray-300'}"
      >
        <div class="font-medium text-sm text-gray-900">Full Stack</div>
        <div class="text-xs text-gray-500 mt-0.5">Restore and restart containers</div>
      </button>
    </div>

    <!-- New Volume: target name input -->
    {#if mode === 'new_volume'}
      <div class="mb-4">
        <label for="target-volume" class="block text-sm font-medium text-gray-700 mb-1">
          Target Volume Name
        </label>
        <input
          id="target-volume"
          type="text"
          bind:value={targetVolumeName}
          class="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
        {#if willOverwrite}
          <p class="mt-1.5 text-xs text-amber-700 bg-amber-50 border border-amber-200 rounded px-2 py-1">
            <strong>Warning:</strong> This will overwrite the existing volume "{volumeName}".
          </p>
        {/if}
      </div>
    {/if}

    <!-- Full Stack: warning -->
    {#if mode === 'full_stack'}
      <div class="mb-4 bg-amber-50 border border-amber-200 rounded-lg px-3 py-2 text-sm text-amber-800">
        <strong>Warning:</strong> This will stop and restart the associated containers during restore.
      </div>
    {/if}

    <!-- Passphrase (if encrypted) -->
    {#if backup.is_encrypted}
      <div class="mb-4">
        <div class="flex items-center justify-between mb-1">
          <label for="passphrase" class="block text-sm font-medium text-gray-700">
            Passphrase
          </label>
          {#if usingStoredPassphrase}
            <span class="text-xs text-green-600 flex items-center gap-1">
              <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
              </svg>
              Using stored passphrase
            </span>
          {/if}
        </div>
        <input
          id="passphrase"
          type="password"
          bind:value={passphrase}
          placeholder="Enter decryption passphrase"
          class="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      </div>
    {/if}

    <!-- Error -->
    {#if error}
      <div class="mb-4 text-sm text-red-600 bg-red-50 border border-red-200 rounded-lg px-3 py-2">
        {error}
      </div>
    {/if}

    <!-- Actions -->
    <div class="flex justify-end gap-3">
      <button
        onclick={onClose}
        class="px-4 py-2 text-sm text-gray-700 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
      >
        Cancel
      </button>
      <button
        onclick={handleStart}
        disabled={loading}
        class="px-4 py-2 text-sm text-white bg-blue-600 hover:bg-blue-500 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {loading ? 'Starting...' : 'Start Restore'}
      </button>
    </div>
  </div>
</div>
