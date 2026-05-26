<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import type { VolumeInfo, BackupFile } from '$lib/types';
  import { listBackups } from '$lib/api';
  import * as projectState from '$lib/state/projects.svelte';

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

  async function handleFinish() {
    await projectState.load();
    goto(`/projects/${projectId}`);
  }
</script>

<div class="space-y-6">
  <div class="bg-slate-800/50 rounded-lg p-4 space-y-3">
    <div class="flex justify-between text-sm">
      <span class="text-slate-500">Project</span>
      <span class="font-medium text-slate-100">{projectName}</span>
    </div>
    <div class="flex justify-between text-sm">
      <span class="text-slate-500">Volumes</span>
      <span class="font-medium text-slate-100">{uniqueVolumes.length}</span>
    </div>
    <div class="flex justify-between text-sm">
      <span class="text-slate-500">Storage Backend</span>
      <span class="font-medium text-slate-100 uppercase">{backendType}</span>
    </div>
  </div>

  <div>
    <p class="text-sm font-medium text-slate-300 mb-3">Backup Verification</p>
    {#if loading}
      <div class="text-sm text-slate-500">Checking backups...</div>
    {:else}
      <div class="space-y-2">
        {#each uniqueVolumes as vol}
          <div class="flex items-center justify-between py-2 px-3 bg-slate-800/50 rounded-lg">
            <span class="text-sm text-slate-300 font-mono">{vol.name}</span>
            {#if backupCounts[vol.name] !== undefined}
              <span
                class="text-sm font-medium {backupCounts[vol.name] > 0 ? 'text-emerald-400' : 'text-slate-500'}"
              >
                {backupCounts[vol.name]} backup{backupCounts[vol.name] === 1 ? '' : 's'}
              </span>
            {:else}
              <span class="text-sm text-slate-500">—</span>
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
      class="flex-1 py-2 px-4 border border-slate-600 text-slate-300 hover:bg-slate-800/50 rounded-lg font-medium transition-all duration-200 active:scale-[0.98]"
    >
      ← Back
    </button>
    <button
      onclick={handleFinish}
      class="flex-1 py-2 px-4 bg-gradient-to-r from-indigo-500 to-violet-500 hover:from-indigo-400 hover:to-violet-400 text-white rounded-lg font-medium transition-all duration-200 active:scale-[0.98]"
    >
      Finish
    </button>
  </div>
</div>
