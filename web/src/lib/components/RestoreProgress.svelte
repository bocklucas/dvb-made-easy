<script lang="ts">
  import type { ProgressEvent } from '$lib/types';

  let { events }: { events: ProgressEvent[] } = $props();

  const stepLabels: Record<string, string> = {
    stopping_containers: 'Stopping containers...',
    creating_volume: 'Creating target volume...',
    downloading: 'Downloading backup',
    extracting: 'Extracting archive into volume...',
    starting_containers: 'Restarting containers...',
    complete: 'Restore complete',
    failed: 'Restore failed'
  };

  function getLabel(step: string): string {
    return stepLabels[step] ?? step;
  }

  function circleColor(status: ProgressEvent['status']): string {
    if (status === 'done') return 'bg-green-500';
    if (status === 'failed' || status === 'error') return 'bg-red-500';
    if (status === 'in_progress') return 'bg-blue-500';
    return 'bg-gray-400';
  }
</script>

<div class="flex flex-col">
  {#each events as event, i (i)}
    <div class="flex gap-4">
      <!-- Icon column -->
      <div class="flex flex-col items-center">
        <!-- Colored circle with icon -->
        <div
          class="w-8 h-8 rounded-full flex items-center justify-center flex-shrink-0 {circleColor(event.status)}"
        >
          {#if event.status === 'in_progress'}
            <!-- Spinner SVG -->
            <svg
              class="w-4 h-4 text-white animate-spin"
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle
                class="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                stroke-width="4"
              ></circle>
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
              ></path>
            </svg>
          {:else if event.status === 'done'}
            <!-- Checkmark -->
            <svg
              class="w-4 h-4 text-white"
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width="3"
            >
              <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
            </svg>
          {:else if event.status === 'failed' || event.status === 'error'}
            <!-- X mark -->
            <svg
              class="w-4 h-4 text-white"
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width="3"
            >
              <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          {:else}
            <div class="w-2 h-2 bg-white rounded-full"></div>
          {/if}
        </div>
        <!-- Vertical connecting line (not after last) -->
        {#if i < events.length - 1}
          <div class="w-0.5 flex-1 bg-gray-200 my-1 min-h-4"></div>
        {/if}
      </div>

      <!-- Content column -->
      <div class="pb-6 flex-1 min-w-0">
        <!-- Volume indicator -->
        {#if event.volume_total && event.volume_total > 0}
          <div class="text-xs text-gray-400 mb-0.5">
            Volume {event.volume_index} of {event.volume_total}: <span class="font-mono">{event.volume}</span>
          </div>
        {/if}
        <!-- Step label -->
        <div class="text-sm font-medium text-gray-900 leading-8">{getLabel(event.step)}</div>

        <!-- Message -->
        {#if event.message}
          <div class="text-sm text-gray-500 mt-0.5">{event.message}</div>
        {/if}

        <!-- Progress bar for downloading step -->
        {#if event.step === 'downloading' && event.percent !== undefined}
          <div class="mt-2 w-full bg-gray-200 rounded-full h-2">
            <div
              class="bg-blue-500 h-2 rounded-full transition-all duration-300"
              style="width: {event.percent}%"
            ></div>
          </div>
          <div class="text-xs text-gray-400 mt-1">{event.percent}%</div>
        {/if}

        <!-- Error details -->
        {#if event.details}
          <pre
            class="mt-2 text-xs text-red-700 bg-red-50 border border-red-200 rounded p-2 font-mono whitespace-pre-wrap break-all">{event.details}</pre>
        {/if}
      </div>
    </div>
  {/each}
</div>
