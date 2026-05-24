<script lang="ts">
  import '../app.css';
  import { page } from '$app/stores';
  import ProjectList from '$lib/components/ProjectList.svelte';
  import * as projectState from '$lib/state/projects.svelte';
  import { onMount } from 'svelte';

  let { children } = $props();

  let showSidebar = $derived(
    !$page.url.pathname.startsWith('/setup') &&
    !$page.url.pathname.startsWith('/restore/')
  );

  onMount(() => {
    projectState.load();
  });
</script>

<div class="flex h-screen bg-gray-50">
  {#if showSidebar}
    <ProjectList />
  {/if}
  <main class="flex-1 overflow-y-auto">
    {@render children()}
  </main>
</div>
