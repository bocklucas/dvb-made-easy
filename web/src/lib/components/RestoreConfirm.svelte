<script lang="ts">
  import { untrack } from 'svelte';
  import { goto } from '$app/navigation';
  import { startRestore, checkVolumesExist } from '$lib/api';
  import type { BackupFile } from '$lib/types';

  let {
    projectId,
    volumeName,
    backup,
    storedPassphrase = '',
    stackName,
    onClose
  }: {
    projectId: string;
    volumeName: string;
    backup: BackupFile;
    storedPassphrase?: string;
    stackName: string;
    onClose: () => void;
  } = $props();

  let mode = $state<'new_volume' | 'full_stack'>('new_volume');
  let targetVolumeName = $state(untrack(() => `${stackName}_${volumeName}`));

  let expectedVolumeName = `${stackName}_${volumeName}`;
  let willOverwrite = $derived(targetVolumeName.trim() === expectedVolumeName);
  let passphrase = $state('');
  let loading = $state(false);
  let error = $state<string | null>(null);

  // Volume existence check state
  let volumeExists = $state<boolean | null>(null);
  let checkingExists = $state(false);
  let wipeAcknowledged = $state(false);

  let lastChecked = '';
  async function checkExists(name: string) {
    if (name === lastChecked) return;
    lastChecked = name;
    checkingExists = true;
    volumeExists = null;
    try {
      const res = await checkVolumesExist(projectId, [name]);
      volumeExists = !!res[name];
      if (!volumeExists) {
        wipeAcknowledged = false;
      }
    } catch (e) {
      console.error(e);
      volumeExists = null;
    } finally {
      checkingExists = false;
    }
  }

  $effect(() => {
    const nameToCheck = mode === 'full_stack' ? expectedVolumeName : targetVolumeName.trim();
    if (nameToCheck) {
      checkExists(nameToCheck);
    } else {
      volumeExists = null;
    }
  });

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
  <div class="relative z-50 bg-slate-900 rounded-xl shadow-xl w-full max-w-md mx-4 p-6">
    <h2 class="text-lg font-semibold text-slate-100 mb-1">Confirm Restore</h2>
    <p class="text-sm text-slate-500 mb-4">
      Backup: <span class="font-mono text-slate-200 break-all">{backup.key}</span>
    </p>

    <!-- Mode selection -->
    <div class="grid grid-cols-2 gap-3 mb-4">
      <button
        onclick={() => (mode = 'new_volume')}
        class="border-2 rounded-lg p-3 text-left transition-colors {mode === 'new_volume'
          ? 'border-indigo-500 bg-indigo-500/10'
          : 'border-slate-700/50 hover:border-slate-600'}"
      >
        <div class="font-medium text-sm text-slate-100">New Volume</div>
        <div class="text-xs text-slate-500 mt-0.5">Restore into a new Docker volume</div>
      </button>
      <button
        onclick={() => (mode = 'full_stack')}
        class="border-2 rounded-lg p-3 text-left transition-colors {mode === 'full_stack'
          ? 'border-indigo-500 bg-indigo-500/10'
          : 'border-slate-700/50 hover:border-slate-600'}"
      >
        <div class="font-medium text-sm text-slate-100">Full Stack</div>
        <div class="text-xs text-slate-500 mt-0.5">Restore and restart containers</div>
      </button>
    </div>

    <!-- New Volume: target name input -->
    {#if mode === 'new_volume'}
      <div class="mb-4">
        <label for="target-volume" class="block text-sm font-medium text-slate-300 mb-1">
          Target Volume Name
        </label>
        <input
          id="target-volume"
          type="text"
          bind:value={targetVolumeName}
          class="w-full border border-slate-600 rounded-lg px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500"
        />
        {#if checkingExists}
          <p class="mt-1.5 text-xs text-slate-500">Checking volume status...</p>
        {:else if volumeExists === false}
          <p class="mt-1.5 text-xs text-emerald-400 bg-emerald-500/10 border border-emerald-500/20 rounded px-2 py-1 flex items-center gap-1.5">
            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7" />
            </svg>
            This volume does not exist and will be created.
          </p>
        {:else if volumeExists === true}
          <p class="mt-1.5 text-xs text-amber-400 bg-amber-500/10 border border-amber-500/20 rounded px-2 py-1 flex items-center gap-1.5">
            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
            This volume already exists and will be completely wiped.
          </p>
        {/if}
      </div>
    {/if}

    <!-- Full Stack: warning -->
    {#if mode === 'full_stack'}
      <div class="mb-4 flex flex-col gap-2">
        <div class="bg-amber-500/10 border border-amber-500/20 rounded-lg px-3 py-2 text-sm text-amber-400">
          <strong>Warning:</strong> This will stop and restart the associated containers during restore.
        </div>
        {#if checkingExists}
          <div class="text-xs text-slate-500">Checking volume status...</div>
        {:else if volumeExists === false}
          <div class="text-xs text-emerald-400 bg-emerald-500/10 border border-emerald-500/20 rounded px-2 py-1 flex items-center gap-1.5">
            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7" />
            </svg>
            Volume "{expectedVolumeName}" does not exist and will be created.
          </div>
        {:else if volumeExists === true}
          <div class="text-xs text-amber-400 bg-amber-500/10 border border-amber-500/20 rounded px-2 py-1 flex items-center gap-1.5">
            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
            Volume "{expectedVolumeName}" already exists and will be completely wiped.
          </div>
        {/if}
      </div>
    {/if}

    <!-- Wipe acknowledgement checkbox (if volume exists) -->
    {#if volumeExists === true}
      <div class="mb-4 bg-red-500/10 border border-red-500/20 rounded-lg p-3">
        <label class="flex items-start gap-2.5 cursor-pointer select-none">
          <input
            type="checkbox"
            bind:checked={wipeAcknowledged}
            class="mt-1 rounded border-slate-700 bg-slate-900 text-indigo-600 focus:ring-indigo-500"
          />
          <div class="text-xs text-slate-300 leading-tight">
            I understand and agree that the existing volume <span class="font-mono text-white">"{mode === 'full_stack' ? expectedVolumeName : targetVolumeName.trim()}"</span> will be <strong>completely wiped (deleted and recreated)</strong> prior to restoring to prevent older files mixing with newer ones.
          </div>
        </label>
      </div>
    {/if}

    <!-- Passphrase (if encrypted) -->
    {#if backup.is_encrypted}
      <div class="mb-4">
        <div class="flex items-center justify-between mb-1">
          <label for="passphrase" class="block text-sm font-medium text-slate-300">
            Passphrase
          </label>
          {#if usingStoredPassphrase}
            <span class="text-xs text-emerald-400 flex items-center gap-1">
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
          class="w-full border border-slate-600 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
        />
      </div>
    {/if}

    <!-- Error -->
    {#if error}
      <div class="mb-4 text-sm text-red-400 bg-red-500/10 border border-red-500/20 rounded-lg px-3 py-2">
        {error}
      </div>
    {/if}

    <!-- Actions -->
    <div class="flex justify-end gap-3">
      <button
        onclick={onClose}
        class="px-4 py-2 text-sm text-slate-300 border border-slate-600 rounded-lg hover:bg-slate-800/50 transition-all duration-200 active:scale-[0.98]"
      >
        Cancel
      </button>
      <button
        onclick={handleStart}
        disabled={loading || checkingExists || (volumeExists === true && !wipeAcknowledged)}
        class="px-4 py-2 text-sm text-white bg-gradient-to-r from-indigo-500 to-violet-500 hover:from-indigo-400 hover:to-violet-400 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {loading ? 'Starting...' : 'Start Restore'}
      </button>
    </div>
  </div>
</div>
