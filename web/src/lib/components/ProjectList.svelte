<script lang="ts">
  import type { Project } from '$lib/types';
  import * as projectState from '$lib/state/projects.svelte';
  import { goto } from '$app/navigation';

  let projects = $derived(projectState.getProjects());
  let selectedId = $derived(projectState.getSelectedId());
  let confirmDeleteId: string | null = $state(null);

  async function handleDelete(id: string) {
    if (confirmDeleteId === id) {
      await projectState.remove(id);
      confirmDeleteId = null;
      goto('/');
    } else {
      confirmDeleteId = id;
    }
  }
</script>

<nav class="w-64 bg-slate-900/80 backdrop-blur-xl border-r border-slate-700/50 text-white h-screen flex flex-col">
  <div class="p-4 border-b border-slate-700/50">
    <a href="/" class="block hover:opacity-80 transition-opacity">
      <h1 class="text-lg font-semibold bg-gradient-to-r from-indigo-400 to-violet-400 bg-clip-text text-transparent">offen made easy</h1>
    </a>
  </div>

  <div class="flex-1 overflow-y-auto">
    {#each projects as project (project.id)}
      <a
        href="/projects/{project.id}"
        class="flex items-center justify-between px-4 py-3 transition-all duration-200 {selectedId === project.id ? 'bg-slate-800/80 border-l-2 border-indigo-400 shadow-[inset_0_0_20px_rgba(99,102,241,0.05)]' : 'border-l-2 border-transparent hover:bg-slate-800/50'}"
      >
        <div class="min-w-0 flex-1">
          <div class="font-medium truncate text-slate-100">{project.name}</div>
          <div class="text-sm text-slate-500">{project.volumes.length} volume{project.volumes.length !== 1 ? 's' : ''}</div>
        </div>
        <button
          onclick={(e) => { e.preventDefault(); handleDelete(project.id); }}
          class="ml-2 text-slate-600 hover:text-red-400 transition-colors shrink-0"
          title={confirmDeleteId === project.id ? 'Click again to confirm' : 'Delete project'}
        >
          {#if confirmDeleteId === project.id}
            <span class="text-red-400 text-xs">confirm?</span>
          {:else}
            <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
          {/if}
        </button>
      </a>
    {/each}
  </div>

  <div class="p-4 border-t border-slate-700/50 space-y-2">
    <a
      href="/setup"
      class="flex items-center justify-center w-full py-2 px-4 bg-gradient-to-r from-indigo-600 to-violet-600 hover:from-indigo-500 hover:to-violet-500 rounded-lg text-sm font-medium transition-all duration-200 active:scale-[0.98]"
    >
      + New Project
    </a>
    <a
      href="/backup-creator"
      class="flex items-center justify-center w-full py-2 px-4 bg-gradient-to-r from-emerald-600 to-emerald-500 hover:from-emerald-500 hover:to-emerald-400 rounded-lg text-sm font-medium transition-all duration-200 active:scale-[0.98]"
    >
      + Backup Creator
    </a>
    <a
      href="/settings"
      class="flex items-center gap-2 px-4 py-2 text-sm text-slate-500 hover:text-slate-200 transition-all duration-200 rounded-lg hover:bg-slate-800/60"
    >
      <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
        <path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
      </svg>
      Settings
    </a>
  </div>
</nav>
