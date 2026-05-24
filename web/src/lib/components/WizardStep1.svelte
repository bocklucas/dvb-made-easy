<script lang="ts">
  import type { VolumeInfo, InferenceResult } from '$lib/types';
  import { importCompose, createProject, inferFromCompose } from '$lib/api';

  interface Props {
    onComplete: (projectId: string, volumes: VolumeInfo[], name: string, inference?: InferenceResult) => void;
    onBack?: () => void;
  }

  let { onComplete, onBack }: Props = $props();

  let composeContent = $state('');
  let projectName = $state('');
  let deploymentMode = $state<'standalone' | 'swarm'>('standalone');
  let error = $state('');
  let loading = $state(false);

  async function handleNext() {
    error = '';
    if (!composeContent.trim()) {
      error = 'Paste your docker-compose.yml content';
      return;
    }
    if (!projectName.trim()) {
      error = 'Enter a project name';
      return;
    }

    loading = true;
    try {
      const importResult = await importCompose(composeContent, deploymentMode);
      await createProject(importResult.id, projectName.trim());

      // Best-effort inference — never block the wizard if it fails
      let inference: InferenceResult | undefined;
      try {
        inference = await inferFromCompose(composeContent);
      } catch {
        // ignore inference errors
      }

      onComplete(importResult.id, importResult.volumes, projectName.trim(), inference);
    } catch (e) {
      error = e instanceof Error ? e.message : 'Import failed';
    } finally {
      loading = false;
    }
  }
</script>

<div class="space-y-6">
  <div>
    <label for="compose" class="block text-sm font-medium text-gray-700 mb-2">
      docker-compose.yml
    </label>
    <textarea
      id="compose"
      bind:value={composeContent}
      rows="12"
      class="w-full px-3 py-2 border border-gray-300 rounded-lg font-mono text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
      placeholder="Paste your docker-compose.yml here..."
    ></textarea>
  </div>

  <div>
    <label for="name" class="block text-sm font-medium text-gray-700 mb-2">
      Project Name
    </label>
    <input
      id="name"
      type="text"
      bind:value={projectName}
      class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
      placeholder="e.g. homelab, production"
    />
  </div>

  <div>
    <p class="text-sm font-medium text-gray-700 mb-2">Deployment Mode</p>
    <div class="flex gap-3">
      <button
        onclick={() => (deploymentMode = 'standalone')}
        class="flex-1 py-2 px-4 rounded-lg border-2 text-sm font-medium transition-colors {deploymentMode === 'standalone' ? 'border-blue-500 bg-blue-50 text-blue-700' : 'border-gray-200 text-gray-600 hover:border-gray-300'}"
      >
        Standalone
      </button>
      <button
        onclick={() => (deploymentMode = 'swarm')}
        class="flex-1 py-2 px-4 rounded-lg border-2 text-sm font-medium transition-colors {deploymentMode === 'swarm' ? 'border-blue-500 bg-blue-50 text-blue-700' : 'border-gray-200 text-gray-600 hover:border-gray-300'}"
      >
        Docker Swarm
      </button>
    </div>
    <p class="text-xs text-gray-400 mt-1">
      {deploymentMode === 'swarm' ? 'Volumes are prefixed with the stack name (e.g., mystack_data)' : 'Volumes are prefixed with the project name (e.g., myproject_data)'}
    </p>
  </div>

  {#if error}
    <div class="p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">{error}</div>
  {/if}

  <div class="flex gap-3">
    {#if onBack}
      <button
        onclick={onBack}
        disabled={loading}
        class="flex-1 py-2 px-4 border border-gray-300 hover:bg-gray-50 disabled:opacity-50 text-gray-700 rounded-lg font-medium transition-colors"
      >
        Back
      </button>
    {/if}
    <button
      onclick={handleNext}
      disabled={loading}
      class="{onBack ? 'flex-1' : 'w-full'} py-2 px-4 bg-blue-600 hover:bg-blue-500 disabled:bg-blue-300 text-white rounded-lg font-medium transition-colors"
    >
      {loading ? 'Importing...' : 'Next'}
    </button>
  </div>
</div>
