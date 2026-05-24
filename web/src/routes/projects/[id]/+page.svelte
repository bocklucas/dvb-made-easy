<script lang="ts">
  import { page } from '$app/stores';
  import { getProject, getCredentials, setProjectPassphrase, deleteProjectPassphrase, syncGit } from '$lib/api';
  import * as projectState from '$lib/state/projects.svelte';
  import VolumeCard from '$lib/components/VolumeCard.svelte';
  import RestoreConfirm from '$lib/components/RestoreConfirm.svelte';
  import ComposeRestoreFlow from '$lib/components/ComposeRestoreFlow.svelte';
  import ComposeEditModal from '$lib/components/ComposeEditModal.svelte';
  import StorageEditModal from '$lib/components/StorageEditModal.svelte';
  import type { BackupFile, ComposeDiff, CredentialResponse, Project } from '$lib/types';

  let project = $state<Project | null>(null);
  let loading = $state(true);
  let error = $state<string | null>(null);

  let restoreVolume = $state<string | null>(null);
  let restoreBackup = $state<BackupFile | null>(null);
  let showComposeRestore = $state(false);
  let showComposeEdit = $state(false);
  let showStorageEdit = $state(false);
  let existingCredentials = $state<CredentialResponse | null>(null);

  // Git sync state
  let syncLoading = $state(false);
  let syncError = $state<string | null>(null);
  let syncDiff = $state<ComposeDiff | null>(null);
  let syncCommit = $state<string | null>(null);

  async function handleGitSync() {
    if (!project) return;
    syncLoading = true;
    syncError = null;
    syncDiff = null;
    syncCommit = null;
    try {
      const result = await syncGit(project.id);
      syncCommit = result.commit;
      if (result.changed && result.diff) {
        syncDiff = result.diff;
      }
      await loadProject(project.id);
    } catch (e) {
      syncError = e instanceof Error ? e.message : 'Sync failed';
    } finally {
      syncLoading = false;
    }
  }

  function dismissSyncDiff() {
    syncDiff = null;
    syncCommit = null;
  }

  // Default passphrase UI state
  let showDefaultPassphraseForm = $state(false);
  let defaultPassphraseInput = $state('');
  let defaultPassphraseLoading = $state(false);
  let defaultPassphraseError = $state<string | null>(null);

  async function loadProject(id: string) {
    loading = true;
    error = null;
    project = null;
    existingCredentials = null;
    projectState.setSelectedId(id);
    try {
      project = await getProject(id);
      try {
        existingCredentials = await getCredentials(id);
      } catch {
        // No credentials configured — that's fine.
      }
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to load project';
    } finally {
      loading = false;
    }
  }

  async function openStorageEdit() {
    if (!project) return;
    // Re-fetch credentials in case they changed since page load.
    try {
      existingCredentials = await getCredentials(project.id);
    } catch {
      existingCredentials = null;
    }
    showStorageEdit = true;
  }

  $effect(() => {
    const id = $page.params.id;
    if (id) {
      loadProject(id);
    }
  });

  function handleRestore(volumeName: string, backup: BackupFile) {
    restoreVolume = volumeName;
    restoreBackup = backup;
  }

  function closeModal() {
    restoreVolume = null;
    restoreBackup = null;
  }

  async function handleSetDefaultPassphrase() {
    if (!project || !defaultPassphraseInput.trim()) return;
    defaultPassphraseLoading = true;
    defaultPassphraseError = null;
    try {
      await setProjectPassphrase(project.id, defaultPassphraseInput.trim());
      defaultPassphraseInput = '';
      showDefaultPassphraseForm = false;
      await loadProject(project.id);
    } catch (e) {
      defaultPassphraseError = e instanceof Error ? e.message : 'Failed to save passphrase';
    } finally {
      defaultPassphraseLoading = false;
    }
  }

  async function handleClearDefaultPassphrase() {
    if (!project) return;
    defaultPassphraseLoading = true;
    defaultPassphraseError = null;
    try {
      await deleteProjectPassphrase(project.id);
      await loadProject(project.id);
    } catch (e) {
      defaultPassphraseError = e instanceof Error ? e.message : 'Failed to clear passphrase';
    } finally {
      defaultPassphraseLoading = false;
    }
  }

  function cancelDefaultPassphrase() {
    showDefaultPassphraseForm = false;
    defaultPassphraseInput = '';
    defaultPassphraseError = null;
  }

  // Find stored passphrase for a given volume (volume-specific > project default)
  function getStoredPassphrase(volumeName: string): string {
    if (!project) return '';
    const vol = project.volumes.find((v) => v.name === volumeName);
    if (vol?.passphrase) return vol.passphrase;
    return project.default_passphrase ?? '';
  }

  let groupedVolumes = $derived.by(() => {
    if (!project) return [];
    const groups = new Map<string, typeof project.volumes>();
    for (const vol of project.volumes) {
      const existing = groups.get(vol.compose_service) ?? [];
      existing.push(vol);
      groups.set(vol.compose_service, existing);
    }
    return Array.from(groups.entries()).map(([service, vols]) => ({ service, volumes: vols }));
  });
</script>

<div class="p-6 max-w-3xl">
  {#if loading}
    <div class="text-gray-400">Loading project...</div>
  {:else if error}
    <div class="text-red-500">{error}</div>
  {:else if project}
    <div class="mb-6">
      <h1 class="text-2xl font-semibold text-gray-900">{project.name}</h1>
      <p class="text-sm text-gray-500 mt-1">
        {project.volumes.length} volume{project.volumes.length !== 1 ? 's' : ''}
      </p>

      {#if project.source === 'portainer' && project.portainer_source}
        <div class="mt-2 flex items-center gap-3 flex-wrap">
          <span class="inline-flex items-center gap-1 text-xs font-medium px-2 py-0.5 rounded bg-teal-50 text-teal-700 border border-teal-200">
            Source: Portainer
          </span>
          <span class="text-xs text-gray-500 font-mono truncate max-w-xs" title={project.portainer_source.portainer_url}>
            {project.portainer_source.portainer_url}
          </span>
          <span class="text-xs text-gray-400">
            stack: {project.portainer_source.stack_id}
          </span>
        </div>
      {/if}

      {#if project.source === 'git' && project.git_source}
        <div class="mt-2 flex items-center gap-3 flex-wrap">
          <span class="inline-flex items-center gap-1 text-xs font-medium px-2 py-0.5 rounded bg-purple-50 text-purple-700 border border-purple-200">
            Source: Git
          </span>
          <span class="text-xs text-gray-500 font-mono truncate max-w-xs" title={project.git_source.repo_url}>
            {project.git_source.repo_url}
          </span>
          <span class="text-xs text-gray-400">
            branch: {project.git_source.branch}
          </span>
          {#if project.git_source.last_synced_commit}
            <span class="text-xs text-gray-400 font-mono" title={project.git_source.last_synced_commit}>
              {project.git_source.last_synced_commit.slice(0, 7)}
            </span>
          {/if}
          <button
            onclick={handleGitSync}
            disabled={syncLoading}
            class="px-3 py-1 text-xs text-white bg-purple-600 hover:bg-purple-500 disabled:bg-purple-300 rounded-lg transition-colors"
          >
            {syncLoading ? 'Syncing...' : 'Sync Now'}
          </button>
        </div>
        {#if syncError}
          <div class="mt-2 p-2 bg-red-50 border border-red-200 rounded text-red-700 text-xs">{syncError}</div>
        {/if}
        {#if syncDiff}
          <div class="mt-2 p-3 bg-gray-50 border border-gray-200 rounded-lg text-sm">
            <div class="flex items-center justify-between mb-2">
              <span class="font-medium text-gray-700">Compose updated</span>
              {#if syncCommit}
                <span class="text-xs text-gray-400 font-mono">{syncCommit.slice(0, 7)}</span>
              {/if}
            </div>
            {#if syncDiff.added.length > 0}
              <div class="mb-1">
                <span class="text-green-700 text-xs font-medium">Added:</span>
                {#each syncDiff.added as vol}
                  <span class="ml-1 text-xs text-green-600">{vol.name}</span>
                {/each}
              </div>
            {/if}
            {#if syncDiff.removed.length > 0}
              <div class="mb-1">
                <span class="text-red-700 text-xs font-medium">Removed:</span>
                {#each syncDiff.removed as vol}
                  <span class="ml-1 text-xs text-red-600">{vol.name}</span>
                {/each}
              </div>
            {/if}
            {#if syncDiff.unchanged.length > 0}
              <div class="mb-1">
                <span class="text-gray-500 text-xs font-medium">Unchanged:</span>
                {#each syncDiff.unchanged as vol}
                  <span class="ml-1 text-xs text-gray-500">{vol.name}</span>
                {/each}
              </div>
            {/if}
            <button
              onclick={dismissSyncDiff}
              class="mt-2 text-xs text-gray-500 hover:text-gray-700 underline"
            >
              Dismiss
            </button>
          </div>
        {/if}
      {/if}

      <div class="mt-3 flex items-center gap-3">
        <button
          onclick={() => (showComposeRestore = true)}
          class="px-4 py-2 text-sm text-white bg-blue-600 hover:bg-blue-500 rounded-lg transition-colors"
        >
          Restore Compose
        </button>
        <button
          onclick={() => (showComposeEdit = true)}
          class="px-4 py-2 text-sm text-gray-700 border border-gray-300 hover:bg-gray-50 rounded-lg transition-colors"
        >
          {project.git_source ? 'View Compose' : 'Edit Compose'}
        </button>
        {#if project.deployment_mode}
          <span class="inline-flex items-center text-xs font-medium px-2 py-0.5 rounded bg-gray-100 text-gray-700 border border-gray-200">
            {project.deployment_mode === 'swarm' ? 'Swarm' : 'Standalone'}
          </span>
        {/if}
      </div>

      <!-- Storage config summary -->
      <div class="mt-4 flex items-center gap-3">
        {#if existingCredentials}
          <div class="flex items-center gap-2">
            <span
              class="inline-flex items-center text-xs font-medium px-2 py-0.5 rounded bg-gray-100 text-gray-700 border border-gray-200 uppercase tracking-wide"
            >
              {existingCredentials.type}
            </span>
            <span class="text-sm text-gray-500">
              {#if existingCredentials.type === 'local' && existingCredentials.local}
                {existingCredentials.local.path}
              {:else if existingCredentials.type === 'smb' && existingCredentials.smb}
                {existingCredentials.smb.host}/{existingCredentials.smb.share}
              {/if}
            </span>
          </div>
        {:else}
          <span class="text-sm text-gray-400">No storage configured</span>
        {/if}
        <button
          onclick={openStorageEdit}
          class="px-3 py-1 text-xs text-gray-700 border border-gray-300 hover:bg-gray-50 rounded-lg transition-colors"
        >
          {existingCredentials ? 'Edit Storage' : 'Add Storage'}
        </button>
      </div>
    </div>

    <!-- Default passphrase section -->
    <div class="mb-6 bg-white rounded-lg shadow-sm border border-gray-200 px-4 py-3">
      <div class="flex items-center justify-between">
        <div>
          <div class="flex items-center gap-2">
            <span class="text-sm font-medium text-gray-700">Default Passphrase</span>
            {#if project.default_passphrase}
              <span
                class="inline-flex items-center gap-1 text-xs text-green-700 bg-green-50 border border-green-200 rounded px-1.5 py-0.5"
              >
                <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                </svg>
                Stored
              </span>
            {/if}
          </div>
          <p class="text-xs text-gray-400 mt-0.5">
            Used for encrypted restores when no volume-specific passphrase is set.
          </p>
        </div>
        <div class="flex items-center gap-2">
          {#if project.default_passphrase}
            <button
              onclick={() => (showDefaultPassphraseForm = !showDefaultPassphraseForm)}
              class="text-xs px-2 py-1 border border-gray-300 rounded hover:bg-gray-50 transition-colors text-gray-600"
              disabled={defaultPassphraseLoading}
            >
              Change
            </button>
            <button
              onclick={handleClearDefaultPassphrase}
              class="text-xs px-2 py-1 border border-red-200 rounded hover:bg-red-50 transition-colors text-red-600"
              disabled={defaultPassphraseLoading}
            >
              {defaultPassphraseLoading ? '...' : 'Clear'}
            </button>
          {:else}
            <button
              onclick={() => (showDefaultPassphraseForm = !showDefaultPassphraseForm)}
              class="text-xs px-2 py-1 border border-gray-300 rounded hover:bg-gray-50 transition-colors text-gray-600"
            >
              Set Default Passphrase
            </button>
          {/if}
        </div>
      </div>

      {#if showDefaultPassphraseForm}
        <div class="mt-3 pt-3 border-t border-gray-100">
          <label for="default-passphrase" class="block text-sm font-medium text-gray-700 mb-1">
            {project.default_passphrase ? 'Change Default Passphrase' : 'Set Default Passphrase'}
          </label>
          <div class="flex gap-2">
            <input
              id="default-passphrase"
              type="password"
              bind:value={defaultPassphraseInput}
              placeholder="Enter passphrase"
              class="flex-1 border border-gray-300 rounded-lg px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <button
              onclick={handleSetDefaultPassphrase}
              disabled={defaultPassphraseLoading || !defaultPassphraseInput.trim()}
              class="px-3 py-1.5 text-sm text-white bg-blue-600 hover:bg-blue-500 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {defaultPassphraseLoading ? 'Saving...' : 'Save'}
            </button>
            <button
              onclick={cancelDefaultPassphrase}
              class="px-3 py-1.5 text-sm text-gray-700 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
            >
              Cancel
            </button>
          </div>
          {#if defaultPassphraseError}
            <div class="mt-2 text-sm text-red-600">{defaultPassphraseError}</div>
          {/if}
        </div>
      {/if}
    </div>

    <div class="flex flex-col gap-6">
      {#each groupedVolumes as group (group.service)}
        <div>
          <h3 class="text-sm font-medium text-gray-500 uppercase tracking-wide mb-2">{group.service}</h3>
          <div class="flex flex-col gap-3">
            {#each group.volumes as volume (volume.name + '-' + volume.compose_service + '-' + volume.compose_mount_path)}
              <VolumeCard
                projectId={project.id}
                {volume}
                onRestore={handleRestore}
                onPassphraseChange={() => loadProject(project!.id)}
              />
            {/each}
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

{#if restoreVolume && restoreBackup && project}
  <RestoreConfirm
    projectId={project.id}
    volumeName={restoreVolume}
    backup={restoreBackup}
    storedPassphrase={getStoredPassphrase(restoreVolume)}
    onClose={closeModal}
  />
{/if}

{#if showComposeRestore && project}
  <ComposeRestoreFlow
    projectId={project.id}
    storedPassphrases={Object.fromEntries(
      project.volumes
        .filter((v) => !!v.passphrase || !!project?.default_passphrase)
        .map((v) => [v.name, v.passphrase || project!.default_passphrase || ''])
    )}
    onClose={() => (showComposeRestore = false)}
  />
{/if}

{#if showComposeEdit && project}
  <ComposeEditModal
    projectId={project.id}
    initialContent={project.compose_content}
    onClose={() => (showComposeEdit = false)}
    onSaved={() => loadProject(project!.id)}
    readOnly={!!project.git_source}
  />
{/if}

{#if showStorageEdit && project}
  <StorageEditModal
    projectId={project.id}
    existing={existingCredentials ?? { type: 'local' }}
    onClose={() => (showStorageEdit = false)}
    onSaved={() => loadProject(project!.id)}
  />
{/if}
