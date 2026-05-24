<script lang="ts">
  import type { Project } from '$lib/types';
  import * as projectState from '$lib/state/projects.svelte';

  let projects = $derived(projectState.getProjects());
  let selectedId = $derived(projectState.getSelectedId());
  let confirmDeleteId: string | null = $state(null);

  async function handleDelete(id: string) {
    if (confirmDeleteId === id) {
      await projectState.remove(id);
      confirmDeleteId = null;
    } else {
      confirmDeleteId = id;
    }
  }
</script>

<nav class="w-64 bg-gray-900 text-white h-screen flex flex-col">
  <div class="p-4 border-b border-gray-700">
    <h1 class="text-lg font-semibold">offen restore</h1>
  </div>

  <div class="flex-1 overflow-y-auto">
    {#each projects as project (project.id)}
      <a
        href="/projects/{project.id}"
        class="flex items-center justify-between px-4 py-3 hover:bg-gray-800 transition-colors {selectedId === project.id ? 'bg-gray-800 border-l-2 border-blue-400' : 'border-l-2 border-transparent'}"
      >
        <div class="min-w-0 flex-1">
          <div class="font-medium truncate">{project.name}</div>
          <div class="text-sm text-gray-400">{project.volumes.length} volume{project.volumes.length !== 1 ? 's' : ''}</div>
        </div>
        <button
          onclick={(e) => { e.preventDefault(); handleDelete(project.id); }}
          class="ml-2 text-gray-500 hover:text-red-400 transition-colors shrink-0"
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

  <div class="p-4 border-t border-gray-700">
    <a
      href="/setup"
      class="flex items-center justify-center w-full py-2 px-4 bg-blue-600 hover:bg-blue-500 rounded-lg text-sm font-medium transition-colors"
    >
      + New Project
    </a>
  </div>
</nav>
