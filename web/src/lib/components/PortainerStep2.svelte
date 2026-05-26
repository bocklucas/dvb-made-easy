<script lang="ts">
  import type { PortainerStack, VolumeInfo } from '$lib/types';
  import { portainerStacks, importPortainer } from '$lib/api';

  interface Props {
    portainerUrl: string;
    apiKey: string;
    endpointId: number;
    savedSourceId?: string;
    onComplete: (projectId: string, volumes: VolumeInfo[], name: string) => void;
    onBack?: () => void;
  }

  let { portainerUrl, apiKey, endpointId, savedSourceId, onComplete, onBack }: Props = $props();

  let stacks = $state<PortainerStack[]>([]);
  let loading = $state(true);
  let error = $state('');
  let selectedStack = $state<PortainerStack | null>(null);
  let projectName = $state('');
  let deploymentMode = $state<'standalone' | 'swarm'>('standalone');
  let importing = $state(false);

  let searchQuery = $state('');
  let selectedNode = $state('');

  let uniqueNodes = $derived.by(() => {
    const nodes = new Set<string>();
    for (const stack of stacks) {
      if (stack.endpoint_name) {
        nodes.add(stack.endpoint_name);
      }
    }
    return Array.from(nodes).sort();
  });

  let filteredStacks = $derived.by(() => {
    return stacks.filter((stack) => {
      const matchesSearch = !searchQuery.trim() || stack.Name.toLowerCase().includes(searchQuery.trim().toLowerCase());
      const matchesNode = !selectedNode || stack.endpoint_name === selectedNode;
      return matchesSearch && matchesNode;
    });
  });

  async function loadStacks() {
    loading = true;
    error = '';
    try {
      stacks = await portainerStacks(portainerUrl, apiKey, endpointId, savedSourceId || undefined);
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to load stacks';
    } finally {
      loading = false;
    }
  }

  async function handleImport() {
    if (!selectedStack) return;
    if (!projectName.trim()) {
      error = 'Project name is required';
      return;
    }

    importing = true;
    error = '';
    try {
      const result = await importPortainer({
        portainer_url: portainerUrl,
        api_key: apiKey,
        stack_id: selectedStack.Id,
        endpoint_id: selectedStack.EndpointId,
        project_name: projectName.trim(),
        deployment_mode: deploymentMode,
        saved_source_id: savedSourceId || undefined
      });
      onComplete(result.id, result.volumes, projectName.trim());
    } catch (e) {
      error = e instanceof Error ? e.message : 'Import failed';
    } finally {
      importing = false;
    }
  }

  function handleSelectStack(stack: PortainerStack) {
    selectedStack = stack;
    if (!projectName) {
      projectName = stack.Name;
    }
    error = '';
  }

  // Load stacks on mount
  $effect(() => {
    loadStacks();
  });
</script>

<div class="space-y-5">
  <div>
    <h3 class="text-lg font-semibold text-slate-100">Select Stack</h3>
    <p class="text-sm text-slate-500 mt-1">Choose a stack to import. Offen-backed stacks are highlighted.</p>
  </div>

  {#if loading}
    <div class="text-sm text-slate-500 py-4 text-center">Loading stacks...</div>
  {:else if error && stacks.length === 0}
    <div class="p-3 bg-red-500/10 border border-red-500/20 rounded-lg text-red-400 text-sm">{error}</div>
  {:else if stacks.length === 0}
    <div class="p-3 bg-yellow-500/10 border border-yellow-500/20 rounded-lg text-yellow-450 text-sm">
      No stacks found on this Portainer instance.
    </div>
  {:else}
    <div class="flex gap-3 mb-4">
      <div class="flex-1">
        <label for="search-stacks" class="sr-only">Search Stacks</label>
        <input
          id="search-stacks"
          type="text"
          bind:value={searchQuery}
          placeholder="Search stacks by name..."
          class="w-full px-3 py-2 bg-slate-900 border border-slate-700/60 rounded-lg text-sm text-slate-100 placeholder-slate-500 focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500"
        />
      </div>
      <div class="w-48">
        <label for="filter-node" class="sr-only">Filter by Node</label>
        <select
          id="filter-node"
          bind:value={selectedNode}
          class="w-full px-3 py-2 bg-slate-900 border border-slate-700/60 rounded-lg text-sm text-slate-100 focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500"
        >
          <option value="">All nodes</option>
          {#each uniqueNodes as node}
            <option value={node}>{node}</option>
          {/each}
        </select>
      </div>
    </div>

    <div class="space-y-2 max-h-[28rem] overflow-y-auto pr-1">
      {#if filteredStacks.length === 0}
        <div class="text-sm text-slate-500 py-8 text-center bg-slate-950/20 border border-slate-800/40 rounded-lg">
          No matching stacks found.
        </div>
      {:else}
        {#each filteredStacks as stack}
          <button
            onclick={() => handleSelectStack(stack)}
            class="w-full flex items-center gap-3 p-3 border rounded-lg text-left transition-colors {selectedStack?.Id === stack.Id ? 'border-indigo-400 bg-indigo-500/10' : 'border-slate-700/50 hover:border-indigo-500/30 hover:bg-slate-800/50'}"
          >
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2">
                <span class="font-medium text-slate-100 text-sm">{stack.Name}</span>
                {#if stack.is_offen_backed}
                  <span class="inline-flex items-center gap-1 text-xs font-medium px-1.5 py-0.5 rounded bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                    <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                    </svg>
                    offen-backed
                  </span>
                {/if}
              </div>
              {#if stack.endpoint_name}
                <div class="text-xs text-indigo-400/90 font-medium mt-0.5">
                  {stack.endpoint_name}
                </div>
              {/if}
              <div class="text-xs text-slate-500 mt-0.5">
                ID: {stack.Id} &middot; Status: {stack.Status === 1 ? 'Running' : 'Stopped'}
              </div>
            </div>
            {#if selectedStack?.Id === stack.Id}
              <svg class="w-5 h-5 text-indigo-400 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
              </svg>
            {/if}
          </button>
        {/each}
      {/if}
    </div>
  {/if}

  {#if selectedStack}
    <div>
      <label for="project-name" class="block text-sm font-medium text-slate-300 mb-1">
        Project Name
      </label>
      <input
        id="project-name"
        type="text"
        bind:value={projectName}
        class="w-full px-3 py-2 border border-slate-600 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
        placeholder="e.g. homelab, production"
      />
    </div>

    <div>
      <p class="text-sm font-medium text-slate-300 mb-2">Deployment Mode</p>
      <div class="flex gap-3">
        <button
          onclick={() => (deploymentMode = 'standalone')}
          class="flex-1 py-2 px-4 rounded-lg border-2 text-sm font-medium transition-colors {deploymentMode === 'standalone' ? 'border-indigo-500 bg-indigo-500/10 text-indigo-400' : 'border-slate-700/50 text-slate-400 hover:border-slate-600'}"
        >
          Standalone
        </button>
        <button
          onclick={() => (deploymentMode = 'swarm')}
          class="flex-1 py-2 px-4 rounded-lg border-2 text-sm font-medium transition-colors {deploymentMode === 'swarm' ? 'border-indigo-500 bg-indigo-500/10 text-indigo-400' : 'border-slate-700/50 text-slate-400 hover:border-slate-600'}"
        >
          Docker Swarm
        </button>
      </div>
      <p class="text-xs text-slate-500 mt-1">
        {deploymentMode === 'swarm' ? 'Volumes are prefixed with the stack name (e.g., mystack_data)' : 'Volumes are prefixed with the project name (e.g., myproject_data)'}
      </p>
    </div>
  {/if}

  {#if error}
    <div class="p-3 bg-red-500/10 border border-red-500/20 rounded-lg text-red-400 text-sm">{error}</div>
  {/if}

  <div class="flex gap-3">
    {#if onBack}
      <button
        onclick={onBack}
        disabled={importing}
        class="flex-1 py-2 px-4 border border-slate-600 hover:bg-slate-800/50 disabled:opacity-50 text-slate-300 rounded-lg font-medium transition-all duration-200 active:scale-[0.98]"
      >
        Back
      </button>
    {/if}
    <button
      onclick={handleImport}
      disabled={importing || !selectedStack || !projectName.trim()}
      class="{onBack ? 'flex-1' : 'w-full'} py-2 px-4 bg-gradient-to-r from-indigo-500 to-violet-500 hover:from-indigo-400 hover:to-violet-400 disabled:opacity-50 text-white rounded-lg font-medium transition-all duration-200 active:scale-[0.98]"
    >
      {importing ? 'Importing...' : 'Import Stack'}
    </button>
  </div>
</div>
