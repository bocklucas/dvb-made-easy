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

<div class="p-6 max-w-2xl animate-fade-in">
  <div class="mb-6">
    <h1 class="text-2xl font-light text-slate-100">Restore Progress</h1>
    <p class="text-sm text-slate-500 mt-1">
      Token: <span class="font-mono text-slate-400 bg-slate-800 px-1.5 py-0.5 rounded">{$page.params.token}</span>
    </p>
  </div>

  {#if events.length === 0}
    {#if connectionLost}
      <div class="text-slate-500">
        <p class="mb-3">Restore session not found or already completed.</p>
        <a href="/" class="text-indigo-400 hover:text-indigo-300 hover:underline text-sm transition-colors">← Back to projects</a>
      </div>
    {:else}
      <div class="flex items-center gap-2 text-slate-500">
        <svg
          class="w-4 h-4 animate-spin text-indigo-400"
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
    <RestoreProgress {events} />

    {#if connectionLost && !isTerminal}
      <div
        class="mt-4 bg-amber-500/10 border border-amber-500/20 rounded-xl px-4 py-3 text-sm text-amber-400"
      >
        Connection lost — the restore may still be running.
      </div>
    {/if}

    {#if isComplete}
      <div
        class="mt-4 bg-emerald-500/10 border border-emerald-500/20 rounded-xl px-4 py-3 text-sm text-emerald-400"
      >
        <p class="font-medium">Restore completed successfully.</p>
        <a href="/" class="mt-1 inline-block text-emerald-400 underline hover:text-emerald-300 transition-colors"
          >← Back to projects</a
        >
      </div>
    {/if}

    {#if isFailed}
      <div
        class="mt-4 bg-red-500/10 border border-red-500/20 rounded-xl px-4 py-3 text-sm text-red-400"
      >
        <p class="font-medium">Restore failed.</p>
        {#if lastEvent?.details}
          <pre class="mt-2 text-xs font-mono whitespace-pre-wrap break-all text-red-300">{lastEvent.details}</pre>
        {/if}
        <a href="/" class="mt-2 inline-block text-red-400 underline hover:text-red-300 transition-colors"
          >← Back to projects</a
        >
      </div>
    {/if}
  {/if}
</div>
