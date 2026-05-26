<script lang="ts">
  import type { SavedSource } from '$lib/types';
  import { portainerConnect, listSavedSources, createSavedSource, getSavedSource } from '$lib/api';
  import { onMount } from 'svelte';

  interface Props {
    onNext: (url: string, apiKey: string, endpointId: number, savedSourceId?: string) => void;
    onBack?: () => void;
    initialUrl?: string;
    initialApiKey?: string;
  }

  let { onNext, onBack, initialUrl, initialApiKey }: Props = $props();

  let url = $state(initialUrl ?? '');
  let apiKey = $state(initialApiKey ?? '');
  let loading = $state(false);
  let error = $state('');

  let savedSources = $state<SavedSource[]>([]);
  let selectedSourceId = $state('');
  let saveSource = $state(false);
  let sourceName = $state('');

  onMount(async () => {
    try {
      const all = await listSavedSources();
      savedSources = all.filter((s) => s.type === 'portainer');
    } catch {
      // ignore — saved sources are optional
    }
  });

  async function handleSourceSelect(id: string) {
    selectedSourceId = id;
    if (id === '') return;
    try {
      const source = await getSavedSource(id);
      if (source?.portainer_config) {
        url = source.portainer_config.portainer_url ?? '';
        apiKey = source.portainer_config.api_key ?? '';
      }
    } catch {
      // Fall back to the sanitized data from the list
      const source = savedSources.find((s) => s.id === id);
      if (source?.portainer_config) {
        url = source.portainer_config.portainer_url ?? '';
        apiKey = '';
      }
    }
  }

  async function handleConnect() {
    error = '';
    if (!url.trim()) {
      error = 'Portainer URL is required';
      return;
    }
    if (!selectedSourceId && !apiKey.trim()) {
      error = 'API key is required';
      return;
    }

    loading = true;
    try {
      await portainerConnect(url.trim(), apiKey.trim(), selectedSourceId || undefined);

      if (saveSource) {
        const name = sourceName.trim() || url.trim();
        try {
          await createSavedSource({
            name,
            type: 'portainer',
            portainer_config: {
              portainer_url: url.trim(),
              api_key: apiKey.trim(),
              stack_id: 0,
              endpoint_id: 0
            }
          });
        } catch {
          // saving the source is best-effort; don't block the connection
        }
      }

      onNext(url.trim(), apiKey.trim(), 0, selectedSourceId || undefined);
    } catch (e) {
      error = e instanceof Error ? e.message : 'Connection failed';
    } finally {
      loading = false;
    }
  }
</script>

<div class="space-y-5">
  <div>
    <h3 class="text-lg font-semibold text-slate-100">Connect to Portainer</h3>
    <p class="text-sm text-slate-500 mt-1">Enter your Portainer URL and API key to get started.</p>
  </div>

  {#if savedSources.length > 0}
    <div>
      <label for="saved-source" class="block text-sm font-medium text-slate-300 mb-1">
        Use a saved source
      </label>
      <select
        id="saved-source"
        value={selectedSourceId}
        onchange={(e) => handleSourceSelect(e.currentTarget.value)}
        class="w-full px-3 py-2 border border-slate-600 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-slate-900"
      >
        <option value="">Enter manually</option>
        {#each savedSources as source}
          <option value={source.id}>{source.name}</option>
        {/each}
      </select>
    </div>
  {/if}

  <div>
    <label for="portainer-url" class="block text-sm font-medium text-slate-300 mb-1">
      Portainer URL
    </label>
    <input
      id="portainer-url"
      type="url"
      bind:value={url}
      class="w-full px-3 py-2 border border-slate-600 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-slate-950/40"
      placeholder="https://portainer.example.com"
    />
  </div>

  {#if !selectedSourceId}
    <div>
      <label for="api-key" class="block text-sm font-medium text-slate-300 mb-1">
        API Key
      </label>
      <input
        id="api-key"
        type="password"
        bind:value={apiKey}
        class="w-full px-3 py-2 border border-slate-600 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-slate-950/40"
        placeholder="ptr_..."
      />
    </div>
  {/if}

  {#if !selectedSourceId}
    <div class="flex items-start gap-2">
      <input
        id="save-connection"
        type="checkbox"
        bind:checked={saveSource}
        class="mt-0.5 rounded border-slate-600 text-indigo-400 focus:ring-indigo-500 bg-slate-950/40"
      />
      <div class="flex-1">
        <label for="save-connection" class="text-sm font-medium text-slate-300">
          Save this connection for reuse
        </label>
        {#if saveSource}
          <input
            type="text"
            bind:value={sourceName}
            class="mt-1 w-full px-3 py-2 border border-slate-600 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-slate-950/40"
            placeholder="Connection name (optional, defaults to URL)"
          />
        {/if}
      </div>
    </div>
  {/if}

  {#if error}
    <div class="p-3 bg-red-500/10 border border-red-500/20 rounded-lg text-red-400 text-sm">{error}</div>
  {/if}

  <div class="flex gap-3">
    {#if onBack}
      <button
        onclick={onBack}
        disabled={loading}
        class="flex-1 py-2 px-4 border border-slate-600 hover:bg-slate-800/50 disabled:opacity-50 text-slate-300 rounded-lg font-medium transition-all duration-200 active:scale-[0.98]"
      >
        Back
      </button>
    {/if}
    <button
      onclick={handleConnect}
      disabled={loading}
      class="{onBack ? 'flex-1' : 'w-full'} py-2 px-4 bg-gradient-to-r from-indigo-500 to-violet-500 hover:from-indigo-400 hover:to-violet-400 disabled:opacity-50 text-white rounded-lg font-medium transition-all duration-200 active:scale-[0.98]"
    >
      {loading ? 'Connecting...' : 'Connect'}
    </button>
  </div>
</div>
