<script lang="ts">
  import type { VolumeInfo } from '$lib/types';

  interface Props {
    volumes: VolumeInfo[];
    onNext: () => void;
    onBack?: () => void;
  }

  let { volumes, onNext, onBack }: Props = $props();
</script>

<div class="space-y-5">
  <div>
    <h3 class="text-lg font-semibold text-gray-900">Imported Volumes</h3>
    <p class="text-sm text-gray-500 mt-1">
      {volumes.length} volume{volumes.length !== 1 ? 's' : ''} found in the compose file.
    </p>
  </div>

  {#if volumes.length === 0}
    <div class="p-4 bg-yellow-50 border border-yellow-200 rounded-lg text-yellow-700 text-sm">
      No named volumes found in this compose file.
    </div>
  {:else}
    <div class="space-y-2">
      {#each volumes as vol}
        <div class="flex items-center gap-3 p-3 border border-gray-200 rounded-lg bg-gray-50">
          <div class="flex-shrink-0 w-8 h-8 rounded bg-blue-100 flex items-center justify-center">
            <svg class="w-4 h-4 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
            </svg>
          </div>
          <div class="min-w-0 flex-1">
            <div class="font-medium text-gray-900 text-sm">{vol.name}</div>
            <div class="text-xs text-gray-500">
              {vol.compose_service} &rarr; {vol.compose_mount_path}
            </div>
          </div>
          {#if vol.backup_pattern}
            <span class="text-xs text-gray-400 font-mono">{vol.backup_pattern}</span>
          {/if}
        </div>
      {/each}
    </div>
  {/if}

  <div class="flex gap-3">
    {#if onBack}
      <button
        onclick={onBack}
        class="flex-1 py-2 px-4 border border-gray-300 hover:bg-gray-50 text-gray-700 rounded-lg font-medium transition-colors"
      >
        Back
      </button>
    {/if}
    <button
      onclick={onNext}
      class="{onBack ? 'flex-1' : 'w-full'} py-2 px-4 bg-blue-600 hover:bg-blue-500 text-white rounded-lg font-medium transition-colors"
    >
      Next
    </button>
  </div>
</div>
