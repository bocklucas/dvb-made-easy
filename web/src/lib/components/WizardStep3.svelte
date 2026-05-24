<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import type { VolumeInfo, BackupFile } from '$lib/types';
  import { listBackups } from '$lib/api';

  interface Props {
    projectId: string;
    projectName: string;
    volumes: VolumeInfo[];
    backendType: string;
    onBack: () => void;
  }

  let { projectId, projectName, volumes, backendType, onBack }: Props = $props();

  let uniqueVolumes = $derived(
    volumes.filter((v, i, arr) => arr.findIndex((u) => u.name === v.name) === i)
  );

  let backupCounts = $state<Record<string, number>>({});
  let loading = $state(true);

  let totalBackups = $derived(Object.values(backupCounts).reduce((sum, n) => sum + n, 0));

  onMount(async () => {
    const results = await Promise.allSettled(
      uniqueVolumes.map(async (vol) => {
        const files: BackupFile[] = await listBackups(projectId, vol.name);
        return { name: vol.name, count: files.length };
      })
    );

    const counts: Record<string, number> = {};
    for (const result of results) {
      if (result.status === 'fulfilled') {
        counts[result.value.name] = result.value.count;
      }
    }
    backupCounts = counts;
    loading = false;
  });

  function handleFinish() {
    goto(`/projects/${projectId}`);
  }
</script>

<div class="space-y-6">
  <div class="bg-gray-50 rounded-lg p-4 space-y-3">
    <div class="flex justify-between text-sm">
      <span class="text-gray-500">Project</span>
      <span class="font-medium text-gray-900">{projectName}</span>
    </div>
    <div class="flex justify-between text-sm">
      <span class="text-gray-500">Volumes</span>
      <span class="font-medium text-gray-900">{uniqueVolumes.length}</span>
    </div>
    <div class="flex justify-between text-sm">
      <span class="text-gray-500">Storage Backend</span>
      <span class="font-medium text-gray-900 uppercase">{backendType}</span>
    </div>
  </div>

  <div>
    <p class="text-sm font-medium text-gray-700 mb-3">Backup Verification</p>
    {#if loading}
      <div class="text-sm text-gray-500">Checking backups...</div>
    {:else}
      <div class="space-y-2">
        {#each uniqueVolumes as vol}
          <div class="flex items-center justify-between py-2 px-3 bg-gray-50 rounded-lg">
            <span class="text-sm text-gray-700 font-mono">{vol.name}</span>
            {#if backupCounts[vol.name] !== undefined}
              <span
                class="text-sm font-medium {backupCounts[vol.name] > 0 ? 'text-green-600' : 'text-gray-400'}"
              >
                {backupCounts[vol.name]} backup{backupCounts[vol.name] === 1 ? '' : 's'}
              </span>
            {:else}
              <span class="text-sm text-gray-400">—</span>
            {/if}
          </div>
        {/each}
      </div>

      {#if totalBackups === 0}
        <div class="mt-3 p-3 bg-yellow-50 border border-yellow-200 rounded-lg text-yellow-800 text-sm">
          No backups found — check your storage path and backup naming pattern.
        </div>
      {/if}
    {/if}
  </div>

  <div class="flex gap-3">
    <button
      onclick={onBack}
      class="flex-1 py-2 px-4 border border-gray-300 text-gray-700 hover:bg-gray-50 rounded-lg font-medium transition-colors"
    >
      ← Back
    </button>
    <button
      onclick={handleFinish}
      class="flex-1 py-2 px-4 bg-blue-600 hover:bg-blue-500 text-white rounded-lg font-medium transition-colors"
    >
      Finish
    </button>
  </div>
</div>
