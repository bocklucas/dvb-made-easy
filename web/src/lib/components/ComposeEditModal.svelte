<script lang="ts">
  import { updateCompose, confirmVolumeRemoval } from '$lib/api';
  import type { ComposeDiff, ComposeDiffVolume, ProjectSource } from '$lib/types';

  interface Props {
    projectId: string;
    initialContent: string;
    onClose: () => void;
    onSaved: () => void;
    readOnly?: boolean;
    source?: ProjectSource;
  }

  let { projectId, initialContent, onClose, onSaved, readOnly = false, source }: Props = $props();

  type Stage = 'edit' | 'diff' | 'confirm-removal';

  let stage = $state<Stage>('edit');
  // content is initialized from the prop once on mount; subsequent prop changes are intentionally ignored
  // (the modal always unmounts/remounts when re-opened so initialContent is always current).
  // eslint-disable-next-line svelte/no-unused-svelte-ignore
  // svelte-ignore state_referenced_locally
  let content = $state(initialContent);
  let diff = $state<ComposeDiff | null>(null);
  let saving = $state(false);
  let confirming = $state(false);
  let error = $state<string | null>(null);

  async function handleSave() {
    saving = true;
    error = null;
    try {
      diff = await updateCompose(projectId, content);
      if (diff.removed.length > 0) {
        stage = 'confirm-removal';
      } else {
        stage = 'diff';
        onSaved();
      }
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to update compose';
    } finally {
      saving = false;
    }
  }

  async function handleConfirmRemoval() {
    if (!diff) return;
    confirming = true;
    error = null;
    try {
      const removedNames = diff.removed.map((v) => v.name);
      await confirmVolumeRemoval(projectId, removedNames);
      diff = { ...diff, removed: [] };
      stage = 'diff';
      onSaved();
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to confirm removal';
    } finally {
      confirming = false;
    }
  }

  function handleKeepRemoved() {
    // User declines removal — compose content already updated, removed volumes kept in manifest.
    stage = 'diff';
    onSaved();
  }

  function handleDone() {
    onClose();
  }

  function diffSummary(d: ComposeDiff): string {
    const parts: string[] = [];
    if (d.added.length > 0)
      parts.push(`${d.added.length} new volume${d.added.length !== 1 ? 's' : ''} added`);
    if (d.removed.length > 0)
      parts.push(`${d.removed.length} volume${d.removed.length !== 1 ? 's' : ''} removed`);
    if (d.unchanged.length > 0) parts.push(`${d.unchanged.length} unchanged`);
    return parts.length > 0 ? parts.join(', ') : 'No changes detected';
  }

  function volumeLabel(v: ComposeDiffVolume): string {
    return `${v.name} (${v.compose_service}:${v.compose_mount_path})`;
  }
</script>

<!-- Backdrop -->
<div
  class="fixed inset-0 bg-black/40 z-40 flex items-center justify-center p-4"
  role="dialog"
  aria-modal="true"
  aria-labelledby="compose-edit-title"
>
  <div class="bg-slate-900 rounded-xl shadow-xl w-full max-w-2xl max-h-[90vh] flex flex-col">
    <!-- Header -->
    <div class="flex items-center justify-between px-5 py-4 border-b border-slate-700/50">
      <h2 id="compose-edit-title" class="text-lg font-semibold text-slate-100">
        {#if stage === 'edit'}
          {readOnly ? 'Compose File' : 'Edit Compose File'}
        {:else if stage === 'diff'}
          Volume Changes
        {:else}
          Confirm Volume Removal
        {/if}
      </h2>
      <button
        onclick={onClose}
        class="text-slate-500 hover:text-slate-400 transition-all duration-200 active:scale-[0.98]"
        aria-label="Close"
      >
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M6 18L18 6M6 6l12 12"
          />
        </svg>
      </button>
    </div>

    <!-- Body -->
    <div class="flex-1 overflow-y-auto px-5 py-4">
      {#if stage === 'edit'}
        {#if readOnly}
          {#if source === 'git'}
            <p class="text-sm text-slate-500 mb-3">
              This compose file is managed via Git. To make changes, update the repository and sync.
            </p>
          {:else if source === 'portainer'}
            <p class="text-sm text-slate-500 mb-3">
              This compose file is managed via Portainer. To make changes, update the stack in Portainer.
            </p>
          {/if}
        {:else}
          <p class="text-sm text-slate-500 mb-3">
            Edit the docker-compose content below. Volume changes will be detected automatically.
          </p>
        {/if}
        <textarea
          bind:value={content}
          rows="18"
          class="w-full font-mono text-sm border border-slate-600 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-indigo-500 resize-none {readOnly ? 'bg-slate-800/50 text-slate-400 cursor-default' : ''}"
          spellcheck="false"
          readonly={readOnly}
        ></textarea>
      {:else if stage === 'diff' && diff}
        <p class="text-sm text-slate-400 mb-4">{diffSummary(diff)}</p>

        {#if diff.added.length > 0}
          <div class="mb-4">
            <h3 class="text-sm font-medium text-emerald-400 mb-2">Added volumes</h3>
            <ul class="space-y-1">
              {#each diff.added as v (v.name)}
                <li
                  class="text-sm bg-emerald-500/10 border border-emerald-500/20 rounded px-3 py-1.5 text-emerald-400"
                >
                  + {volumeLabel(v)}
                </li>
              {/each}
            </ul>
          </div>
        {/if}

        {#if diff.unchanged.length > 0}
          <div>
            <h3 class="text-sm font-medium text-slate-500 mb-2">Unchanged volumes</h3>
            <ul class="space-y-1">
              {#each diff.unchanged as v (v.name)}
                <li
                  class="text-sm bg-slate-800/50 border border-slate-700/50 rounded px-3 py-1.5 text-slate-400"
                >
                  {volumeLabel(v)}
                </li>
              {/each}
            </ul>
          </div>
        {/if}
      {:else if stage === 'confirm-removal' && diff}
        <div class="mb-4 p-3 bg-amber-500/10 border border-amber-500/20 rounded-lg">
          <p class="text-sm font-medium text-amber-400">
            The following volumes are no longer in the compose file:
          </p>
        </div>

        <ul class="space-y-2 mb-4">
          {#each diff.removed as v (v.name)}
            <li class="bg-red-500/10 border border-red-500/20 rounded-lg px-4 py-3">
              <p class="text-sm font-medium text-red-400">{v.name}</p>
              <p class="text-xs text-red-400 mt-0.5">
                Service: {v.compose_service} &middot; Mount: {v.compose_mount_path}
              </p>
              <p class="text-xs text-red-400 mt-1 font-medium">
                Warning: the stored passphrase for this volume will be permanently lost.
              </p>
            </li>
          {/each}
        </ul>

        <p class="text-sm text-slate-400">
          Confirm to permanently remove these volumes from this project. Or keep them to preserve
          their stored passphrases.
        </p>
      {/if}

      {#if error}
        <div class="mt-3 text-sm text-red-400">{error}</div>
      {/if}
    </div>

    <!-- Footer -->
    <div class="flex items-center justify-end gap-2 px-5 py-4 border-t border-slate-700/50">
      {#if stage === 'edit'}
        {#if readOnly}
          <button
            onclick={onClose}
            class="px-4 py-2 text-sm text-slate-300 border border-slate-600 rounded-lg hover:bg-slate-800/50 transition-all duration-200 active:scale-[0.98]"
          >
            Close
          </button>
        {:else}
          <button
            onclick={onClose}
            class="px-4 py-2 text-sm text-slate-300 border border-slate-600 rounded-lg hover:bg-slate-800/50 transition-all duration-200 active:scale-[0.98]"
          >
            Cancel
          </button>
          <button
            onclick={handleSave}
            disabled={saving || content.trim() === ''}
            class="px-4 py-2 text-sm text-white bg-gradient-to-r from-indigo-500 to-violet-500 hover:from-indigo-400 hover:to-violet-400 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {saving ? 'Saving...' : 'Save Changes'}
          </button>
        {/if}
      {:else if stage === 'diff'}
        <button
          onclick={handleDone}
          class="px-4 py-2 text-sm text-white bg-gradient-to-r from-indigo-500 to-violet-500 hover:from-indigo-400 hover:to-violet-400 rounded-lg transition-all duration-200 active:scale-[0.98]"
        >
          Done
        </button>
      {:else if stage === 'confirm-removal'}
        <button
          onclick={handleKeepRemoved}
          class="px-4 py-2 text-sm text-slate-300 border border-slate-600 rounded-lg hover:bg-slate-800/50 transition-all duration-200 active:scale-[0.98]"
        >
          Keep volumes
        </button>
        <button
          onclick={handleConfirmRemoval}
          disabled={confirming}
          class="px-4 py-2 text-sm text-white bg-red-600 hover:bg-red-500 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {confirming ? 'Removing...' : 'Remove volumes'}
        </button>
      {/if}
    </div>
  </div>
</div>
