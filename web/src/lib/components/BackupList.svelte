<script lang="ts">
  import { onMount } from 'svelte';
  import { listBackups } from '$lib/api';
  import type { BackupFile } from '$lib/types';

  let {
    projectId,
    volumeName,
    onRestore
  }: {
    projectId: string;
    volumeName: string;
    onRestore: (backup: BackupFile) => void;
  } = $props();

  let backups = $state<BackupFile[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);

  function formatSize(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
  }

  function formatDate(iso: string): string {
    return new Date(iso).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  }

  onMount(async () => {
    try {
      backups = await listBackups(projectId, volumeName);
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to load backups';
    } finally {
      loading = false;
    }
  });
</script>

{#if loading}
  <div class="py-4 text-center text-gray-400 text-sm">Loading backups...</div>
{:else if error}
  <div class="py-4 text-center text-red-500 text-sm">{error}</div>
{:else if backups.length === 0}
  <div class="py-4 text-center text-gray-400 text-sm">No backups found.</div>
{:else}
  <div class="divide-y divide-gray-100">
    {#each backups as backup (backup.key)}
      <div class="flex items-center justify-between py-2 px-1 gap-4">
        <div class="min-w-0 flex-1">
          <div class="font-mono text-sm truncate text-gray-800">{backup.key}</div>
          <div class="flex items-center gap-2 mt-0.5">
            <span class="text-xs text-gray-500">{formatSize(backup.size)}</span>
            <span class="text-xs text-gray-400">·</span>
            <span class="text-xs text-gray-500">{formatDate(backup.last_modified)}</span>
            {#if backup.is_encrypted}
              <span class="text-xs bg-yellow-100 text-yellow-700 px-1.5 py-0.5 rounded font-medium">encrypted</span>
            {/if}
          </div>
        </div>
        <button
          onclick={() => onRestore(backup)}
          class="shrink-0 text-sm px-3 py-1 bg-blue-600 hover:bg-blue-500 text-white rounded transition-colors"
        >
          Restore
        </button>
      </div>
    {/each}
  </div>
{/if}
