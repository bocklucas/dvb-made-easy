<script lang="ts">
  import { goto } from '$app/navigation';
  import * as projectState from '$lib/state/projects.svelte';
  import { onMount } from 'svelte';
  import type { ProjectSource } from '$lib/types';

  let loading = $state(true);
  let error = $state<string | null>(null);

  function relativeTime(isoString: string): string {
    if (!isoString) return 'Never';
    const diff = Date.now() - new Date(isoString).getTime();
    if (diff < 0) return 'Just now';
    const seconds = Math.floor(diff / 1000);
    if (seconds < 60) return 'Just now';
    const minutes = Math.floor(seconds / 60);
    if (minutes < 60) return `${minutes} minute${minutes !== 1 ? 's' : ''} ago`;
    const hours = Math.floor(minutes / 60);
    if (hours < 24) return `${hours} hour${hours !== 1 ? 's' : ''} ago`;
    const days = Math.floor(hours / 24);
    if (days < 30) return `${days} day${days !== 1 ? 's' : ''} ago`;
    const months = Math.floor(days / 30);
    if (months < 12) return `${months} month${months !== 1 ? 's' : ''} ago`;
    const years = Math.floor(months / 12);
    return `${years} year${years !== 1 ? 's' : ''} ago`;
  }

  function sourceBadgeClass(source?: ProjectSource): string {
    if (source === 'portainer') return 'bg-teal-500/10 text-teal-400 border border-teal-500/30';
    if (source === 'git') return 'bg-violet-500/10 text-violet-400 border border-violet-500/30';
    return 'bg-slate-700/50 text-slate-400 border border-slate-600';
  }

  function sourceBadgeLabel(source?: ProjectSource): string {
    if (source === 'portainer') return 'Portainer';
    if (source === 'git') return 'Git';
    return 'Paste';
  }

  let projects = $derived(projectState.getProjects());

  onMount(async () => {
    try {
      await projectState.load();
      if (projectState.getProjects().length === 0) {
        goto('/setup');
      }
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to load projects';
    } finally {
      loading = false;
    }
  });
</script>

<div class="p-8 animate-fade-in">
  {#if loading}
    <div class="flex items-center justify-center h-48 text-slate-500">
      <div class="flex items-center gap-3">
        <div class="w-5 h-5 border-2 border-indigo-500/30 border-t-indigo-500 rounded-full animate-spin"></div>
        <span>Loading...</span>
      </div>
    </div>
  {:else if error}
    <div class="p-4 bg-red-500/10 border border-red-500/30 rounded-xl text-red-400">{error}</div>
  {:else}
    <div class="mb-8">
      <h1 class="text-2xl font-light text-slate-100">Dashboard</h1>
      <p class="text-sm text-slate-500 mt-1">{projects.length} project{projects.length !== 1 ? 's' : ''} configured</p>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {#each projects as project, i (project.id)}
        <a
          href="/projects/{project.id}"
          class="block bg-slate-800/50 rounded-xl border border-slate-700/50 p-5 hover:border-indigo-500/30 hover:-translate-y-0.5 hover:shadow-lg hover:shadow-indigo-500/5 transition-all duration-300 animate-slide-up"
          style="animation-delay: {i * 60}ms"
        >
          <div class="flex items-start justify-between gap-2 mb-3">
            <h2 class="text-base font-semibold text-slate-100 leading-tight truncate">{project.name}</h2>
            <span class="inline-flex items-center shrink-0 text-xs font-medium px-2 py-0.5 rounded {sourceBadgeClass(project.source)}">
              {sourceBadgeLabel(project.source)}
            </span>
          </div>

          <div class="flex items-center justify-between text-sm text-slate-400">
            <span>
              {project.volumes.length} volume{project.volumes.length !== 1 ? 's' : ''}
            </span>
            <span class="text-xs text-slate-500">{relativeTime(project.last_refreshed)}</span>
          </div>
        </a>
      {/each}

      <!-- New Project card -->
      <a
        href="/setup"
        class="flex flex-col items-center justify-center gap-2 bg-slate-800/30 rounded-xl border-2 border-dashed border-slate-700/50 p-5 text-slate-500 hover:border-indigo-500/50 hover:text-indigo-400 hover:shadow-lg hover:shadow-indigo-500/5 transition-all duration-300 min-h-[104px] animate-slide-up"
        style="animation-delay: {projects.length * 60}ms"
      >
        <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
        </svg>
        <span class="text-sm font-medium">New Project</span>
      </a>
    </div>
  {/if}
</div>
