<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { page } from '$app/stores';
  import * as restoreState from '$lib/state/restore.svelte';
  import RestoreProgress from '$lib/components/RestoreProgress.svelte';

  onMount(() => {
    const token = $page.params.token;
    if (token) {
      restoreState.subscribe(token);
    }
  });

  onDestroy(() => {
    restoreState.cleanup();
  });

  let events = $derived(restoreState.getEvents());
  let connectionLost = $derived(restoreState.getConnectionLost());
  let lastEvent = $derived(events.length > 0 ? events[events.length - 1] : null);
  let isComplete = $derived(lastEvent?.step === 'complete');
  let isFailed = $derived(lastEvent?.step === 'failed' || lastEvent?.status === 'failed' || lastEvent?.status === 'error');
  let isTerminal = $derived(isComplete || isFailed);
</script>

<div class="p-6 max-w-2xl">
  <!-- Header -->
  <div class="mb-6">
    <h1 class="text-2xl font-semibold text-gray-900">Restore Progress</h1>
    <p class="text-sm text-gray-500 mt-1">
      Token: <span class="font-mono text-gray-700">{$page.params.token}</span>
    </p>
  </div>

  {#if events.length === 0}
    {#if connectionLost}
      <!-- No events and connection lost -->
      <div class="text-gray-500">
        <p class="mb-3">Restore session not found or already completed.</p>
        <a href="/" class="text-blue-600 hover:underline text-sm">← Back to projects</a>
      </div>
    {:else}
      <!-- Waiting for first event -->
      <div class="flex items-center gap-2 text-gray-500">
        <svg
          class="w-4 h-4 animate-spin text-blue-500"
          xmlns="http://www.w3.org/2000/svg"
          fill="none"
          viewBox="0 0 24 24"
        >
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"
          ></circle>
          <path
            class="opacity-75"
            fill="currentColor"
            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
          ></path>
        </svg>
        <span>Waiting for restore to start...</span>
      </div>
    {/if}
  {:else}
    <!-- Timeline -->
    <RestoreProgress {events} />

    <!-- Connection lost warning (non-terminal) -->
    {#if connectionLost && !isTerminal}
      <div
        class="mt-4 bg-amber-50 border border-amber-200 rounded-lg px-4 py-3 text-sm text-amber-800"
      >
        Connection lost — the restore may still be running.
      </div>
    {/if}

    <!-- Success banner -->
    {#if isComplete}
      <div
        class="mt-4 bg-green-50 border border-green-200 rounded-lg px-4 py-3 text-sm text-green-800"
      >
        <p class="font-medium">Restore completed successfully.</p>
        <a href="/" class="mt-1 inline-block text-green-700 underline hover:text-green-900"
          >← Back to projects</a
        >
      </div>
    {/if}

    <!-- Failure banner -->
    {#if isFailed}
      <div
        class="mt-4 bg-red-50 border border-red-200 rounded-lg px-4 py-3 text-sm text-red-800"
      >
        <p class="font-medium">Restore failed.</p>
        {#if lastEvent?.details}
          <pre class="mt-2 text-xs font-mono whitespace-pre-wrap break-all text-red-700">{lastEvent.details}</pre>
        {/if}
        <a href="/" class="mt-2 inline-block text-red-700 underline hover:text-red-900"
          >← Back to projects</a
        >
      </div>
    {/if}
  {/if}
</div>
