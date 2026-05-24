<script lang="ts">
  import type { PortainerStack, VolumeInfo } from '$lib/types';
  import { portainerStacks, importPortainer } from '$lib/api';

  interface Props {
    portainerUrl: string;
    apiKey: string;
    endpointId: number;
    onComplete: (projectId: string, volumes: VolumeInfo[], name: string) => void;
    onBack?: () => void;
  }

  let { portainerUrl, apiKey, endpointId, onComplete, onBack }: Props = $props();

  let stacks = $state<PortainerStack[]>([]);
  let loading = $state(true);
  let error = $state('');
  let selectedStack = $state<PortainerStack | null>(null);
  let projectName = $state('');
  let importing = $state(false);

  async function loadStacks() {
    loading = true;
    error = '';
    try {
      stacks = await portainerStacks(portainerUrl, apiKey, endpointId);
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
        endpoint_id: endpointId,
        project_name: projectName.trim()
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
    <h3 class="text-lg font-semibold text-gray-900">Select Stack</h3>
    <p class="text-sm text-gray-500 mt-1">Choose a stack to import. Offen-backed stacks are highlighted.</p>
  </div>

  {#if loading}
    <div class="text-sm text-gray-400 py-4 text-center">Loading stacks...</div>
  {:else if error && stacks.length === 0}
    <div class="p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">{error}</div>
  {:else if stacks.length === 0}
    <div class="p-3 bg-yellow-50 border border-yellow-200 rounded-lg text-yellow-700 text-sm">
      No stacks found on this endpoint.
    </div>
  {:else}
    <div class="space-y-2 max-h-64 overflow-y-auto">
      {#each stacks as stack}
        <button
          onclick={() => handleSelectStack(stack)}
          class="w-full flex items-center gap-3 p-3 border rounded-lg text-left transition-colors {selectedStack?.Id === stack.Id ? 'border-blue-400 bg-blue-50' : 'border-gray-200 hover:border-blue-300 hover:bg-gray-50'}"
        >
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <span class="font-medium text-gray-900 text-sm">{stack.Name}</span>
              {#if stack.is_offen_backed}
                <span class="inline-flex items-center gap-1 text-xs font-medium px-1.5 py-0.5 rounded bg-green-100 text-green-700 border border-green-200">
                  <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                  </svg>
                  offen-backed
                </span>
              {/if}
            </div>
            <div class="text-xs text-gray-400 mt-0.5">
              ID: {stack.Id} &middot; Status: {stack.Status === 1 ? 'Running' : 'Stopped'}
            </div>
          </div>
          {#if selectedStack?.Id === stack.Id}
            <svg class="w-5 h-5 text-blue-500 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
              <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
            </svg>
          {/if}
        </button>
      {/each}
    </div>
  {/if}

  {#if selectedStack}
    <div>
      <label for="project-name" class="block text-sm font-medium text-gray-700 mb-1">
        Project Name
      </label>
      <input
        id="project-name"
        type="text"
        bind:value={projectName}
        class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
        placeholder="e.g. homelab, production"
      />
    </div>
  {/if}

  {#if error}
    <div class="p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">{error}</div>
  {/if}

  <div class="flex gap-3">
    {#if onBack}
      <button
        onclick={onBack}
        disabled={importing}
        class="flex-1 py-2 px-4 border border-gray-300 hover:bg-gray-50 disabled:opacity-50 text-gray-700 rounded-lg font-medium transition-colors"
      >
        Back
      </button>
    {/if}
    <button
      onclick={handleImport}
      disabled={importing || !selectedStack || !projectName.trim()}
      class="{onBack ? 'flex-1' : 'w-full'} py-2 px-4 bg-blue-600 hover:bg-blue-500 disabled:bg-blue-300 text-white rounded-lg font-medium transition-colors"
    >
      {importing ? 'Importing...' : 'Import Stack'}
    </button>
  </div>
</div>
