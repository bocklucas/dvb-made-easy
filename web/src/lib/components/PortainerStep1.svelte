<script lang="ts">
  import type { PortainerEndpoint, SavedSource } from '$lib/types';
  import { portainerConnect, listSavedSources, createSavedSource } from '$lib/api';
  import { onMount } from 'svelte';

  interface Props {
    onNext: (url: string, apiKey: string, endpointId: number) => void;
    onBack?: () => void;
    initialUrl?: string;
    initialApiKey?: string;
  }

  let { onNext, onBack, initialUrl, initialApiKey }: Props = $props();

  let url = $state(initialUrl ?? '');
  let apiKey = $state(initialApiKey ?? '');
  let loading = $state(false);
  let error = $state('');
  let endpoints = $state<PortainerEndpoint[]>([]);
  let connected = $state(false);
  let selectedEndpoint = $state<number | null>(null);

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

  function handleSourceSelect(id: string) {
    selectedSourceId = id;
    if (id === '') return;
    const source = savedSources.find((s) => s.id === id);
    if (source?.portainer_config) {
      url = source.portainer_config.portainer_url ?? '';
      // API key is sanitized in the response, so we clear it
      apiKey = '';
    }
  }

  async function handleConnect() {
    error = '';
    if (!url.trim()) {
      error = 'Portainer URL is required';
      return;
    }
    if (!apiKey.trim()) {
      error = 'API key is required';
      return;
    }

    loading = true;
    try {
      const result = await portainerConnect(url.trim(), apiKey.trim());
      endpoints = result.endpoints;
      connected = true;
      if (endpoints.length === 1) {
        selectedEndpoint = endpoints[0].Id;
      }

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
    } catch (e) {
      error = e instanceof Error ? e.message : 'Connection failed';
    } finally {
      loading = false;
    }
  }

  function handleNext() {
    if (selectedEndpoint === null) {
      error = 'Please select an endpoint';
      return;
    }
    onNext(url.trim(), apiKey.trim(), selectedEndpoint);
  }
</script>

<div class="space-y-5">
  <div>
    <h3 class="text-lg font-semibold text-gray-900">Connect to Portainer</h3>
    <p class="text-sm text-gray-500 mt-1">Enter your Portainer URL and API key to get started.</p>
  </div>

  {#if savedSources.length > 0}
    <div>
      <label for="saved-source" class="block text-sm font-medium text-gray-700 mb-1">
        Use a saved source
      </label>
      <select
        id="saved-source"
        value={selectedSourceId}
        onchange={(e) => handleSourceSelect(e.currentTarget.value)}
        disabled={connected}
        class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500 bg-white disabled:bg-gray-50 disabled:text-gray-400"
      >
        <option value="">Enter manually</option>
        {#each savedSources as source}
          <option value={source.id}>{source.name}</option>
        {/each}
      </select>
    </div>
  {/if}

  <div>
    <label for="portainer-url" class="block text-sm font-medium text-gray-700 mb-1">
      Portainer URL
    </label>
    <input
      id="portainer-url"
      type="url"
      bind:value={url}
      disabled={connected}
      class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500 disabled:bg-gray-50 disabled:text-gray-400"
      placeholder="https://portainer.example.com"
    />
  </div>

  <div>
    <label for="api-key" class="block text-sm font-medium text-gray-700 mb-1">
      API Key
    </label>
    <input
      id="api-key"
      type="password"
      bind:value={apiKey}
      disabled={connected}
      class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500 disabled:bg-gray-50 disabled:text-gray-400"
      placeholder="ptr_..."
    />
  </div>

  {#if !connected}
    <div class="flex items-start gap-2">
      <input
        id="save-connection"
        type="checkbox"
        bind:checked={saveSource}
        class="mt-0.5 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
      />
      <div class="flex-1">
        <label for="save-connection" class="text-sm font-medium text-gray-700">
          Save this connection for reuse
        </label>
        {#if saveSource}
          <input
            type="text"
            bind:value={sourceName}
            class="mt-1 w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            placeholder="Connection name (optional, defaults to URL)"
          />
        {/if}
      </div>
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
        onclick={handleConnect}
        disabled={loading}
        class="{onBack ? 'flex-1' : 'w-full'} py-2 px-4 bg-blue-600 hover:bg-blue-500 disabled:bg-blue-300 text-white rounded-lg font-medium transition-colors"
      >
        {loading ? 'Connecting...' : 'Test Connection'}
      </button>
    </div>
  {:else}
    <div class="p-3 bg-green-50 border border-green-200 rounded-lg text-green-700 text-sm flex items-center gap-2">
      <svg class="w-4 h-4 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
      </svg>
      Connected successfully
    </div>

    <div>
      <p class="block text-sm font-medium text-gray-700 mb-2">Select Endpoint</p>
      {#if endpoints.length === 0}
        <div class="p-3 bg-yellow-50 border border-yellow-200 rounded-lg text-yellow-700 text-sm">
          No endpoints found.
        </div>
      {:else}
        <div class="space-y-2">
          {#each endpoints as ep}
            <button
              onclick={() => { selectedEndpoint = ep.Id; error = ''; }}
              class="w-full flex items-center gap-3 p-3 border rounded-lg text-left transition-colors {selectedEndpoint === ep.Id ? 'border-blue-400 bg-blue-50' : 'border-gray-200 hover:border-blue-300 hover:bg-gray-50'}"
            >
              <div class="w-8 h-8 rounded-full bg-blue-100 flex items-center justify-center flex-shrink-0">
                <svg class="w-4 h-4 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M5 12h14M12 5l7 7-7 7" />
                </svg>
              </div>
              <div>
                <div class="font-medium text-gray-900 text-sm">{ep.Name}</div>
                <div class="text-xs text-gray-400">ID: {ep.Id}</div>
              </div>
              {#if selectedEndpoint === ep.Id}
                <svg class="w-5 h-5 text-blue-500 ml-auto" fill="currentColor" viewBox="0 0 20 20">
                  <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
                </svg>
              {/if}
            </button>
          {/each}
        </div>
      {/if}
    </div>

    {#if error}
      <div class="p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">{error}</div>
    {/if}

    <div class="flex gap-3">
      <button
        onclick={() => { connected = false; endpoints = []; selectedEndpoint = null; error = ''; }}
        class="flex-1 py-2 px-4 border border-gray-300 hover:bg-gray-50 text-gray-700 rounded-lg font-medium transition-colors"
      >
        Change Connection
      </button>
      <button
        onclick={handleNext}
        disabled={selectedEndpoint === null}
        class="flex-1 py-2 px-4 bg-blue-600 hover:bg-blue-500 disabled:bg-blue-300 text-white rounded-lg font-medium transition-colors"
      >
        Next
      </button>
    </div>
  {/if}
</div>
