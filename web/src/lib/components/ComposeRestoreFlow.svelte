<script lang="ts">
  import { goto } from '$app/navigation';
  import { getBackupTimestamps, startComposeRestore } from '$lib/api';
  import type { TimestampGroup } from '$lib/types';

  let {
    projectId,
    storedPassphrases = {},
    onClose
  }: {
    projectId: string;
    storedPassphrases?: Record<string, string>;
    onClose: () => void;
  } = $props();

  let step = $state<'timestamps' | 'configure' | 'confirm'>('timestamps');
  let loading = $state(true);
  let error = $state<string | null>(null);
  let submitting = $state(false);

  let timestampGroups = $state<TimestampGroup[]>([]);
  let selectedGroup = $state<TimestampGroup | null>(null);
  let mode = $state<'new_volume' | 'full_stack'>('new_volume');
  let passphrases = $state<Record<string, string>>({});
  let targetNames = $state<Record<string, string>>({});

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
      targetNames[b.volume_name] = b.volume_name;
    }
    step = 'configure';
  }

  function goToConfirm() {
    step = 'confirm';
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
        mode,
        timestamp: selectedGroup.timestamp,
        volumes: selectedGroup.backups.map((b) => ({
          volume_name: b.volume_name,
          backup_key: b.key,
          passphrase: passphrases[b.volume_name] ?? '',
          target_volume_name: mode === 'new_volume' ? (targetNames[b.volume_name] ?? '') : ''
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

  <div class="relative z-50 bg-white rounded-xl shadow-xl w-full max-w-lg mx-4 p-6 max-h-[80vh] overflow-y-auto">
    <div class="flex items-center justify-between mb-4">
      <h2 class="text-lg font-semibold text-gray-900">
        {#if step === 'timestamps'}
          Select Backup Point
        {:else if step === 'configure'}
          Configure Volumes
        {:else}
          Confirm Restore
        {/if}
      </h2>
      <button onclick={onClose} class="text-gray-400 hover:text-gray-600 text-xl leading-none">&times;</button>
    </div>

    {#if loading}
      <div class="text-gray-400 py-8 text-center">Loading timestamps...</div>
    {:else if error && step === 'timestamps'}
      <div class="text-red-500 py-4">{error}</div>
    {:else if step === 'timestamps'}
      {#if timestampGroups.length === 0}
        <div class="text-gray-500 py-8 text-center">No backup timestamps found.</div>
      {:else}
        <div class="flex flex-col gap-2">
          {#each timestampGroups as group}
            <button
              onclick={() => selectTimestamp(group)}
              class="border rounded-lg p-3 text-left transition-colors hover:border-blue-300 hover:bg-blue-50 {group.complete
                ? 'border-gray-200'
                : 'border-gray-200 opacity-60'}"
            >
              <div class="flex items-center justify-between">
                <span class="font-mono text-sm font-medium text-gray-900">{group.timestamp}</span>
                <span class="text-xs text-gray-500">
                  {group.backups.length} volume{group.backups.length !== 1 ? 's' : ''}
                </span>
              </div>
              <div class="text-xs text-gray-500 mt-1">
                {#if group.complete}
                  <span class="text-green-600">Complete</span>
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
      <div class="grid grid-cols-2 gap-3 mb-4">
        <button
          onclick={() => (mode = 'new_volume')}
          class="border-2 rounded-lg p-3 text-left transition-colors {mode === 'new_volume'
            ? 'border-blue-500 bg-blue-50'
            : 'border-gray-200 hover:border-gray-300'}"
        >
          <div class="font-medium text-sm text-gray-900">New Volume</div>
          <div class="text-xs text-gray-500 mt-0.5">Restore into new Docker volumes</div>
        </button>
        <button
          onclick={() => (mode = 'full_stack')}
          class="border-2 rounded-lg p-3 text-left transition-colors {mode === 'full_stack'
            ? 'border-blue-500 bg-blue-50'
            : 'border-gray-200 hover:border-gray-300'}"
        >
          <div class="font-medium text-sm text-gray-900">Full Stack</div>
          <div class="text-xs text-gray-500 mt-0.5">Restore and restart all containers</div>
        </button>
      </div>

      {#if mode === 'full_stack'}
        <div class="mb-4 bg-amber-50 border border-amber-200 rounded-lg px-3 py-2 text-sm text-amber-800">
          <strong>Warning:</strong> This will stop and restart all associated containers during restore.
        </div>
      {/if}

      <div class="flex flex-col gap-3">
        {#each selectedGroup.backups as backup}
          <div class="border border-gray-200 rounded-lg p-3">
            <div class="flex items-center justify-between mb-1">
              <span class="text-sm font-medium text-gray-900">{backup.volume_name}</span>
              <span class="text-xs text-gray-400">{formatSize(backup.size)}</span>
            </div>
            <div class="text-xs text-gray-500 font-mono mb-2 break-all">{backup.key}</div>

            {#if backup.is_encrypted}
              <div class="mb-2">
                <div class="flex items-center justify-between mb-1">
                  <label for="pass-{backup.volume_name}" class="block text-xs font-medium text-gray-600">Passphrase</label>
                  {#if passphrases[backup.volume_name] && passphrases[backup.volume_name] === (storedPassphrases[backup.volume_name] ?? '')}
                    <span class="text-xs text-green-600 flex items-center gap-0.5">
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
                  class="w-full border border-gray-300 rounded-lg px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
            {/if}

            {#if mode === 'new_volume'}
              <div>
                <label for="target-{backup.volume_name}" class="block text-xs font-medium text-gray-600 mb-1">Target Volume Name</label>
                <input
                  id="target-{backup.volume_name}"
                  type="text"
                  bind:value={targetNames[backup.volume_name]}
                  class="w-full border border-gray-300 rounded-lg px-3 py-1.5 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
            {/if}
          </div>
        {/each}
      </div>

    {:else if step === 'confirm' && selectedGroup}
      <div class="mb-4">
        <div class="text-sm text-gray-600 mb-2">
          <strong>Timestamp:</strong> <span class="font-mono">{selectedGroup.timestamp}</span>
        </div>
        <div class="text-sm text-gray-600 mb-2">
          <strong>Mode:</strong> {mode === 'full_stack' ? 'Full Stack' : 'New Volume'}
        </div>
        <div class="text-sm text-gray-600">
          <strong>Volumes:</strong> {selectedGroup.backups.length}
        </div>
      </div>

      <div class="flex flex-col gap-2 mb-4">
        {#each selectedGroup.backups as backup}
          <div class="bg-gray-50 rounded-lg px-3 py-2 text-sm">
            <span class="font-medium text-gray-900">{backup.volume_name}</span>
            {#if mode === 'new_volume'}
              <span class="text-gray-400 mx-1">&rarr;</span>
              <span class="font-mono text-gray-600">{targetNames[backup.volume_name]}</span>
            {/if}
            {#if backup.is_encrypted}
              <span class="ml-2 text-xs text-amber-600">encrypted</span>
            {/if}
          </div>
        {/each}
      </div>
    {/if}

    {#if error && step !== 'timestamps'}
      <div class="mb-4 text-sm text-red-600 bg-red-50 border border-red-200 rounded-lg px-3 py-2">
        {error}
      </div>
    {/if}

    <div class="flex justify-between mt-4 pt-4 border-t border-gray-100">
      <button
        onclick={goBack}
        class="px-4 py-2 text-sm text-gray-700 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
      >
        {step === 'timestamps' ? 'Cancel' : 'Back'}
      </button>

      {#if step === 'configure'}
        <button
          onclick={goToConfirm}
          class="px-4 py-2 text-sm text-white bg-blue-600 hover:bg-blue-500 rounded-lg transition-colors"
        >
          Review
        </button>
      {:else if step === 'confirm'}
        <button
          onclick={handleStart}
          disabled={submitting}
          class="px-4 py-2 text-sm text-white bg-blue-600 hover:bg-blue-500 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {submitting ? 'Starting...' : 'Start Restore'}
        </button>
      {/if}
    </div>
  </div>
</div>
