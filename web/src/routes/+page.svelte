<script lang="ts">
  import { goto } from '$app/navigation';
  import * as projectState from '$lib/state/projects.svelte';
  import { onMount } from 'svelte';

  let error = $state<string | null>(null);

  onMount(async () => {
    try {
      await projectState.load();
      const projects = projectState.getProjects();
      if (projects.length === 0) {
        goto('/setup');
      } else {
        goto(`/projects/${projects[0].id}`);
      }
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to load projects';
    }
  });
</script>

<div class="flex items-center justify-center h-full">
  {#if error}
    <div class="text-red-500">{error}</div>
  {:else}
    <div class="text-gray-400">Loading...</div>
  {/if}
</div>
