<script lang="ts">
  import { goto } from '$app/navigation';
  import { getBackupTimestamps, startComposeRestore, checkVolumesExist } from '$lib/api';
  import type { TimestampGroup } from '$lib/types';

  let {
    projectId,
    storedPassphrases = {},
    stackName,
    onClose
  }: {
    projectId: string;
    storedPassphrases?: Record<string, string>;
    stackName: string;
    onClose: () => void;
  } = $props();

  let step = $state<'timestamps' | 'configure' | 'confirm'>('timestamps');
  let loading = $state(true);
  let error = $state<string | null>(null);
  let submitting = $state(false);

  let timestampGroups = $state<TimestampGroup[]>([]);
  let selectedGroup = $state<TimestampGroup | null>(null);
  let mode = 'new_volume';
  let passphrases = $state<Record<string, string>>({});
  let targetNames = $state<Record<string, string>>({});

  // Volume existence check state
  let existingVolumes = $state<Record<string, boolean>>({});
  let checkingExists = $state(false);
  let wipeAcknowledged = $state(false);

  let hasExistingVolumes = $derived(
    selectedGroup ? selectedGroup.backups.some(b => {
      const targetName = targetNames[b.volume_name]?.trim();
      return !!existingVolumes[targetName];
    }) : false
  );

  $effect(() => {
    loadTimestamps();
  });

  async function loadTimestamps() {
    loading = true;
    error = null;
    try {
      timestampGroups = await getBackupTimestamps(projectId);
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to load timestamps';
    } finally {
      loading = false;
    }
  }

  function selectTimestamp(group: TimestampGroup) {
    selectedGroup = group;
    passphrases = {};
    targetNames = {};
    for (const b of group.backups) {
      if (b.is_encrypted) {
        // Pre-fill with stored passphrase if available.
        passphrases[b.volume_name] = storedPassphrases[b.volume_name] ?? '';
      }
      targetNames[b.volume_name] = `${stackName}_${b.volume_name}`;
    }
    step = 'configure';
  }

  async function goToConfirm() {
    checkingExists = true;
    error = null;
    wipeAcknowledged = false;
    const namesToCheck = Object.values(targetNames).map(n => n.trim()).filter(Boolean);
    try {
      if (namesToCheck.length > 0) {
        existingVolumes = await checkVolumesExist(projectId, namesToCheck);
      } else {
        existingVolumes = {};
      }
      step = 'confirm';
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to check volume status';
    } finally {
      checkingExists = false;
    }
  }

  function goBack() {
    if (step === 'confirm') {
      step = 'configure';
    } else if (step === 'configure') {
      selectedGroup = null;
      step = 'timestamps';
    } else {
      onClose();
    }
  }

  function formatSize(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  }

  async function handleStart() {
    if (!selectedGroup) return;
    submitting = true;
    error = null;
    try {
      const resp = await startComposeRestore(projectId, {
        mode: 'new_volume',
        timestamp: selectedGroup.timestamp,
        volumes: selectedGroup.backups.map((b) => ({
          volume_name: b.volume_name,
          backup_key: b.key,
          passphrase: passphrases[b.volume_name] ?? '',
          target_volume_name: targetNames[b.volume_name] ?? ''
        }))
      });
      goto(`/restore/${resp.restore_token}`);
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to start restore';
      submitting = false;
    }
  }
</script>

<div
  class="fixed inset-0 bg-black/50 z-40 flex items-center justify-center"
  role="dialog"
  aria-modal="true"
>
  <button
    class="absolute inset-0 w-full h-full cursor-default"
    onclick={onClose}
    tabindex="-1"
    aria-label="Close modal"
  ></button>

  <div class="relative z-50 bg-slate-900 rounded-xl shadow-xl w-full max-w-lg mx-4 p-6 max-h-[80vh] overflow-y-auto">
    <div class="flex items-center justify-between mb-4">
      <h2 class="text-lg font-semibold text-slate-100">
        {#if step === 'timestamps'}
          Select Backup Point
        {:else if step === 'configure'}
          Configure Volumes
        {:else}
          Confirm Restore
        {/if}
      </h2>
      <button onclick={onClose} class="text-slate-500 hover:text-slate-400 text-xl leading-none">&times;</button>
    </div>

    {#if loading}
      <div class="text-slate-500 py-8 text-center">Loading timestamps...</div>
    {:else if error && step === 'timestamps'}
      <div class="text-red-500 py-4">{error}</div>
    {:else if step === 'timestamps'}
      {#if timestampGroups.length === 0}
        <div class="text-slate-500 py-8 text-center">No backup timestamps found.</div>
      {:else}
        <div class="flex flex-col gap-2">
          {#each timestampGroups as group}
            <button
              onclick={() => selectTimestamp(group)}
              class="border rounded-lg p-3 text-left transition-colors hover:border-indigo-500/30 hover:bg-indigo-500/10 {group.complete
                ? 'border-slate-700/50'
                : 'border-slate-700/50 opacity-60'}"
            >
              <div class="flex items-center justify-between">
                <span class="font-mono text-sm font-medium text-slate-100">{group.timestamp}</span>
                <span class="text-xs text-slate-500">
                  {group.backups.length} volume{group.backups.length !== 1 ? 's' : ''}
                </span>
              </div>
              <div class="text-xs text-slate-500 mt-1">
                {#if group.complete}
                  <span class="text-emerald-400">Complete</span>
                {:else}
                  <span class="text-amber-600">Incomplete</span>
                {/if}
                — {group.backups.map((b) => b.volume_name).join(', ')}
              </div>
            </button>
          {/each}
        </div>
      {/if}

    {:else if step === 'configure' && selectedGroup}
      <div class="flex flex-col gap-3">
        {#each selectedGroup.backups as backup}
          <div class="border border-slate-700/50 rounded-lg p-3">
            <div class="flex items-center justify-between mb-1">
              <span class="text-sm font-medium text-slate-100">{backup.volume_name}</span>
              <span class="text-xs text-slate-500">{formatSize(backup.size)}</span>
            </div>
            <div class="text-xs text-slate-500 font-mono mb-2 break-all">{backup.key}</div>

            {#if backup.is_encrypted}
              <div class="mb-2">
                <div class="flex items-center justify-between mb-1">
                  <label for="pass-{backup.volume_name}" class="block text-xs font-medium text-slate-400">Passphrase</label>
                  {#if passphrases[backup.volume_name] && passphrases[backup.volume_name] === (storedPassphrases[backup.volume_name] ?? '')}
                    <span class="text-xs text-emerald-400 flex items-center gap-0.5">
                      <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                          d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                      </svg>
                      Stored
                    </span>
                  {/if}
                </div>
                <input
                  id="pass-{backup.volume_name}"
                  type="password"
                  bind:value={passphrases[backup.volume_name]}
                  placeholder="Enter decryption passphrase"
                  class="w-full border border-slate-600 rounded-lg px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                />
              </div>
            {/if}

            <div>
              <label for="target-{backup.volume_name}" class="block text-xs font-medium text-slate-400 mb-1">Target Volume Name</label>
              <input
                id="target-{backup.volume_name}"
                type="text"
                bind:value={targetNames[backup.volume_name]}
                class="w-full border border-slate-600 rounded-lg px-3 py-1.5 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500"
              />
            </div>
          </div>
        {/each}
      </div>

    {:else if step === 'confirm' && selectedGroup}
      <div class="mb-4">
        <div class="text-sm text-slate-400 mb-2">
          <strong>Timestamp:</strong> <span class="font-mono">{selectedGroup.timestamp}</span>
        </div>
        <div class="text-sm text-slate-400">
          <strong>Volumes:</strong> {selectedGroup.backups.length}
        </div>
      </div>

      <div class="flex flex-col gap-2 mb-4">
        {#each selectedGroup.backups as backup}
          {@const targetName = targetNames[backup.volume_name]?.trim()}
          <div class="bg-slate-800/50 rounded-lg px-3 py-2 text-sm flex flex-wrap items-center justify-between gap-2">
            <div>
              <span class="font-medium text-slate-100">{backup.volume_name}</span>
              <span class="text-slate-500 mx-1">&rarr;</span>
              <span class="font-mono text-slate-400">{targetName}</span>
              {#if backup.is_encrypted}
                <span class="ml-2 text-xs text-amber-600 font-medium bg-amber-600/10 px-1 rounded">encrypted</span>
              {/if}
            </div>
            <div>
              {#if existingVolumes[targetName]}
                <span class="text-xs text-amber-400 bg-amber-400/10 border border-amber-400/20 px-1.5 py-0.5 rounded font-medium">exists (will wipe)</span>
              {:else}
                <span class="text-xs text-emerald-400 bg-emerald-400/10 border border-emerald-400/20 px-1.5 py-0.5 rounded font-medium">new volume</span>
              {/if}
            </div>
          </div>
        {/each}
      </div>

      {#if hasExistingVolumes}
        <div class="mb-4 bg-red-500/10 border border-red-500/20 rounded-lg p-3">
          <label class="flex items-start gap-2.5 cursor-pointer select-none">
            <input
              type="checkbox"
              bind:checked={wipeAcknowledged}
              class="mt-1 rounded border-slate-700 bg-slate-900 text-indigo-600 focus:ring-indigo-500"
            />
            <div class="text-xs text-slate-300 leading-tight">
              I understand and agree that the existing volume(s) marked above will be <strong>completely wiped (deleted and recreated)</strong> prior to restoring to prevent older files mixing with newer ones.
            </div>
          </label>
        </div>
      {/if}
    {/if}

    {#if error && step !== 'timestamps'}
      <div class="mb-4 text-sm text-red-400 bg-red-500/10 border border-red-500/20 rounded-lg px-3 py-2">
        {error}
      </div>
    {/if}

    <div class="flex justify-between mt-4 pt-4 border-t border-slate-700/50">
      <button
        onclick={goBack}
        class="px-4 py-2 text-sm text-slate-300 border border-slate-600 rounded-lg hover:bg-slate-800/50 transition-all duration-200 active:scale-[0.98]"
      >
        {step === 'timestamps' ? 'Cancel' : 'Back'}
      </button>

      {#if step === 'configure'}
        <button
          onclick={goToConfirm}
          disabled={checkingExists}
          class="px-4 py-2 text-sm text-white bg-gradient-to-r from-indigo-500 to-violet-500 hover:from-indigo-400 hover:to-violet-400 rounded-lg transition-all duration-200 active:scale-[0.98] disabled:opacity-50"
        >
          {checkingExists ? 'Checking...' : 'Review'}
        </button>
      {:else if step === 'confirm'}
        <button
          onclick={handleStart}
          disabled={submitting || (hasExistingVolumes && !wipeAcknowledged)}
          class="px-4 py-2 text-sm text-white bg-gradient-to-r from-indigo-500 to-violet-500 hover:from-indigo-400 hover:to-violet-400 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {submitting ? 'Starting...' : 'Start Restore'}
        </button>
      {/if}
    </div>
  </div>
</div>
