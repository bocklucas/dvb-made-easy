<script lang="ts">
  import { untrack } from 'svelte';
  import type { InferenceResult, InferredVolume, StorageSuggestion } from '$lib/types';

  interface Props {
    inference: InferenceResult;
    onVolumesChange?: (volumes: InferredVolume[]) => void;
    onStorageChange?: (storage: StorageSuggestion | undefined) => void;
    onContinue: () => void;
    onBack: () => void;
  }

  let { inference, onVolumesChange, onStorageChange, onContinue, onBack }: Props = $props();

  // Local editable copies — initialized once from props using untrack to avoid
  // the Svelte 5 "state_referenced_locally" warning.
  let volumes = $state<InferredVolume[]>(untrack(() => inference.volumes.map((v) => ({ ...v }))));
  let editingVolumeIdx = $state<number | null>(null);

  // Storage overrides
  let storageType = $state<string>(untrack(() => inference.storage?.type ?? 'local'));
  let localPath = $state(untrack(() => inference.storage?.local_path ?? ''));
  let smbHost = $state(untrack(() => inference.storage?.smb_host ?? ''));
  let smbShare = $state(untrack(() => inference.storage?.smb_share ?? ''));
  let smbUser = $state(untrack(() => inference.storage?.smb_username ?? ''));
  let editingStorage = $state(false);

  function confidenceBadgeClass(confidence: string) {
    if (confidence === 'high') return 'bg-green-100 text-green-700';
    if (confidence === 'medium') return 'bg-yellow-100 text-yellow-700';
    return 'bg-gray-100 text-gray-600';
  }

  function saveVolumeEdit(idx: number) {
    editingVolumeIdx = null;
    onVolumesChange?.(volumes);
  }

  function saveStorageEdit() {
    editingStorage = false;
    const updated: StorageSuggestion | undefined = inference.storage
      ? {
          ...inference.storage,
          type: storageType,
          local_path: localPath || undefined,
          smb_host: smbHost || undefined,
          smb_share: smbShare || undefined,
          smb_username: smbUser || undefined
        }
      : undefined;
    onStorageChange?.(updated);
  }

  let detectedCount = $derived(volumes.filter((v) => v.backup_pattern).length);
</script>

<div class="space-y-5">
  <!-- Summary headline -->
  <div class="bg-blue-50 border border-blue-200 rounded-lg px-4 py-3">
    <p class="text-sm text-blue-800 font-medium">Auto-detected from your compose file</p>
    <p class="text-sm text-blue-700 mt-0.5">
      Found {volumes.length} volume{volumes.length !== 1 ? 's' : ''}{detectedCount > 0
        ? `, ${detectedCount} with backup patterns`
        : ''}{inference.storage ? `, and identified your storage backend` : ''}.
      Review and edit anything below, then continue.
    </p>
  </div>

  <!-- Volumes section -->
  <div>
    <h3 class="text-sm font-semibold text-gray-700 mb-2">Volumes</h3>
    <div class="space-y-2">
      {#each volumes as vol, idx}
        <div class="border border-gray-200 rounded-lg p-3 bg-white">
          {#if editingVolumeIdx === idx}
            <!-- Edit mode -->
            <div class="space-y-2">
              <div>
                <label for="pattern-{idx}" class="block text-xs text-gray-500 mb-0.5">Backup Pattern</label>
                <input
                  id="pattern-{idx}"
                  type="text"
                  bind:value={volumes[idx].backup_pattern}
                  class="w-full border border-gray-300 rounded px-2 py-1 text-xs font-mono focus:ring-1 focus:ring-blue-500 focus:outline-none"
                  placeholder="e.g. backup-%Y-%m-%dT%H-%M-%S.tar.gz"
                />
              </div>
              <div>
                <label for="target-{idx}" class="block text-xs text-gray-500 mb-0.5">Target Volume Name</label>
                <input
                  id="target-{idx}"
                  type="text"
                  bind:value={volumes[idx].target_volume_name}
                  class="w-full border border-gray-300 rounded px-2 py-1 text-xs font-mono focus:ring-1 focus:ring-blue-500 focus:outline-none"
                />
              </div>
              <div class="flex items-center gap-2">
                <input
                  type="checkbox"
                  id="encrypted-{idx}"
                  bind:checked={volumes[idx].is_encrypted}
                  class="rounded border-gray-300"
                />
                <label for="encrypted-{idx}" class="text-xs text-gray-600">Encrypted (GPG)</label>
              </div>
              <div class="flex justify-end gap-2 pt-1">
                <button
                  onclick={() => (editingVolumeIdx = null)}
                  class="text-xs px-3 py-1 text-gray-600 border border-gray-300 rounded hover:bg-gray-50"
                >Cancel</button>
                <button
                  onclick={() => saveVolumeEdit(idx)}
                  class="text-xs px-3 py-1 text-white bg-blue-600 hover:bg-blue-500 rounded"
                >Save</button>
              </div>
            </div>
          {:else}
            <!-- View mode -->
            <div class="flex items-start justify-between gap-2">
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2 flex-wrap">
                  <span class="font-mono text-sm font-medium text-gray-900">{vol.name}</span>
                  <span class="text-xs text-gray-400">{vol.compose_service} · {vol.compose_mount_path}</span>
                  {#if vol.is_encrypted}
                    <span class="text-xs bg-purple-100 text-purple-700 px-1.5 py-0.5 rounded font-medium">Encrypted</span>
                  {/if}
                </div>
                {#if vol.backup_pattern}
                  <p class="text-xs text-gray-500 mt-1 font-mono truncate">
                    Pattern: {vol.backup_pattern}
                  </p>
                {/if}
                <p class="text-xs text-gray-500 mt-0.5">
                  Target: <span class="font-mono">{vol.target_volume_name}</span>
                </p>
              </div>
              <button
                onclick={() => (editingVolumeIdx = idx)}
                class="flex-shrink-0 text-gray-400 hover:text-gray-600 p-1 rounded hover:bg-gray-100 transition-colors"
                title="Edit volume"
              >
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                </svg>
              </button>
            </div>
          {/if}
        </div>
      {/each}
    </div>
  </div>

  <!-- Storage section -->
  {#if inference.storage}
    <div>
      <h3 class="text-sm font-semibold text-gray-700 mb-2">Storage Backend</h3>
      <div class="border border-gray-200 rounded-lg p-3 bg-white">
        {#if editingStorage}
          <div class="space-y-2">
            <div>
              <label for="storage-type" class="block text-xs text-gray-500 mb-0.5">Type</label>
              <select
                id="storage-type"
                bind:value={storageType}
                class="w-full border border-gray-300 rounded px-2 py-1 text-xs focus:ring-1 focus:ring-blue-500 focus:outline-none"
              >
                <option value="local">Local</option>
                <option value="smb">SMB / Network Share</option>
              </select>
            </div>
            {#if storageType === 'local'}
              <div>
                <label for="storage-local-path" class="block text-xs text-gray-500 mb-0.5">Path</label>
                <input
                  id="storage-local-path"
                  type="text"
                  bind:value={localPath}
                  class="w-full border border-gray-300 rounded px-2 py-1 text-xs font-mono focus:ring-1 focus:ring-blue-500 focus:outline-none"
                  placeholder="/mnt/backups"
                />
              </div>
            {:else}
              <div class="grid grid-cols-2 gap-2">
                <div>
                  <label for="storage-smb-host" class="block text-xs text-gray-500 mb-0.5">Host</label>
                  <input
                    id="storage-smb-host"
                    type="text"
                    bind:value={smbHost}
                    class="w-full border border-gray-300 rounded px-2 py-1 text-xs font-mono focus:ring-1 focus:ring-blue-500 focus:outline-none"
                    placeholder="192.168.1.10"
                  />
                </div>
                <div>
                  <label for="storage-smb-share" class="block text-xs text-gray-500 mb-0.5">Share</label>
                  <input
                    id="storage-smb-share"
                    type="text"
                    bind:value={smbShare}
                    class="w-full border border-gray-300 rounded px-2 py-1 text-xs font-mono focus:ring-1 focus:ring-blue-500 focus:outline-none"
                    placeholder="backups"
                  />
                </div>
                <div class="col-span-2">
                  <label for="storage-smb-user" class="block text-xs text-gray-500 mb-0.5">Username</label>
                  <input
                    id="storage-smb-user"
                    type="text"
                    bind:value={smbUser}
                    class="w-full border border-gray-300 rounded px-2 py-1 text-xs font-mono focus:ring-1 focus:ring-blue-500 focus:outline-none"
                    placeholder="user"
                  />
                </div>
              </div>
            {/if}
            <div class="flex justify-end gap-2 pt-1">
              <button
                onclick={() => (editingStorage = false)}
                class="text-xs px-3 py-1 text-gray-600 border border-gray-300 rounded hover:bg-gray-50"
              >Cancel</button>
              <button
                onclick={saveStorageEdit}
                class="text-xs px-3 py-1 text-white bg-blue-600 hover:bg-blue-500 rounded"
              >Save</button>
            </div>
          </div>
        {:else}
          <div class="flex items-start justify-between gap-2">
            <div>
              <div class="flex items-center gap-2 flex-wrap">
                <span class="text-sm font-medium text-gray-900 capitalize">{storageType}</span>
                <span
                  class="text-xs px-1.5 py-0.5 rounded font-medium {confidenceBadgeClass(inference.storage.confidence)}"
                >
                  {inference.storage.confidence} confidence
                </span>
                <span class="text-xs text-gray-400">from {inference.storage.source}</span>
              </div>
              {#if storageType === 'local' && localPath}
                <p class="text-xs text-gray-500 mt-1 font-mono">{localPath}</p>
              {:else if storageType === 'smb' && smbHost}
                <p class="text-xs text-gray-500 mt-1 font-mono">
                  {smbUser ? smbUser + '@' : ''}{smbHost}/{smbShare}
                </p>
              {/if}
            </div>
            <button
              onclick={() => (editingStorage = true)}
              class="flex-shrink-0 text-gray-400 hover:text-gray-600 p-1 rounded hover:bg-gray-100 transition-colors"
              title="Edit storage"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
              </svg>
            </button>
          </div>
        {/if}
      </div>
    </div>
  {/if}

  <!-- Navigation -->
  <div class="flex gap-3 pt-1">
    <button
      onclick={onBack}
      class="flex-1 py-2 px-4 border border-gray-300 text-gray-700 hover:bg-gray-50 rounded-lg font-medium transition-colors"
    >
      Back
    </button>
    <button
      onclick={onContinue}
      class="flex-1 py-2 px-4 bg-blue-600 hover:bg-blue-500 text-white rounded-lg font-medium transition-colors"
    >
      Continue
    </button>
  </div>
</div>
