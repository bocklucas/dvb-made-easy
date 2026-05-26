<script lang="ts">
  import { page } from '$app/stores';
  import { getProject, getCredentials, setProjectPassphrase, deleteProjectPassphrase, syncGit, updateProjectSettings } from '$lib/api';
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
      projectState.load().catch((e) => console.error('Failed to refresh project list:', e));
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

  let showDeploymentModeDropdown = $state(false);
  let editingSwarmName = $state(false);
  let swarmNameInput = $state('');

  async function updateDeploymentMode(mode: 'standalone' | 'swarm') {
    if (!project) return;
    showDeploymentModeDropdown = false;
    try {
      project = await updateProjectSettings(project.id, { deployment_mode: mode });
    } catch (e) {
      alert(e instanceof Error ? e.message : 'Failed to update deployment mode');
    }
  }

  async function saveSwarmName() {
    if (!project) return;
    try {
      project = await updateProjectSettings(project.id, { swarm_name: swarmNameInput.trim() });
      editingSwarmName = false;
    } catch (e) {
      alert(e instanceof Error ? e.message : 'Failed to update swarm name');
    }
  }

  function startEditSwarmName() {
    if (!project) return;
    swarmNameInput = project.swarm_name ?? '';
    editingSwarmName = true;
  }
</script>

<div class="p-8 max-w-3xl animate-fade-in">
  {#if loading}
    <div class="flex items-center gap-3 text-slate-500">
      <div class="w-5 h-5 border-2 border-indigo-500/30 border-t-indigo-500 rounded-full animate-spin"></div>
      <span>Loading project...</span>
    </div>
  {:else if error}
    <div class="p-4 bg-red-500/10 border border-red-500/30 rounded-xl text-red-400">{error}</div>
  {:else if project}
    <div class="mb-6">
      <h1 class="text-2xl font-light text-slate-100">{project.name}</h1>
      <p class="text-sm text-slate-500 mt-1">
        {project.volumes.length} volume{project.volumes.length !== 1 ? 's' : ''}
      </p>

      {#if project.source === 'portainer' && project.portainer_source}
        <div class="mt-2 flex items-center gap-3 flex-wrap">
          <span class="inline-flex items-center gap-1 text-xs font-medium px-2 py-0.5 rounded bg-teal-500/10 text-teal-400 border border-teal-500/30">
            Source: Portainer
          </span>
          <span class="text-xs text-slate-500 font-mono truncate max-w-xs" title={project.portainer_source.portainer_url}>
            {project.portainer_source.portainer_url}
          </span>
          <span class="text-xs text-slate-600">
            stack: {project.portainer_source.stack_id}
          </span>
        </div>
      {/if}

      {#if project.source === 'git' && project.git_source}
        <div class="mt-2 flex items-center gap-3 flex-wrap">
          <span class="inline-flex items-center gap-1 text-xs font-medium px-2 py-0.5 rounded bg-violet-500/10 text-violet-400 border border-violet-500/30">
            Source: Git
          </span>
          <span class="text-xs text-slate-500 font-mono truncate max-w-xs" title={project.git_source.repo_url}>
            {project.git_source.repo_url}
          </span>
          <span class="text-xs text-slate-600">
            branch: {project.git_source.branch}
          </span>
          {#if project.git_source.last_synced_commit}
            <span class="text-xs text-slate-600 font-mono" title={project.git_source.last_synced_commit}>
              {project.git_source.last_synced_commit.slice(0, 7)}
            </span>
          {/if}
          <button
            onclick={handleGitSync}
            disabled={syncLoading}
            class="px-3 py-1 text-xs text-white bg-violet-600 hover:bg-violet-500 disabled:bg-violet-600/50 rounded-lg transition-all duration-200 active:scale-[0.98]"
          >
            {syncLoading ? 'Syncing...' : 'Sync Now'}
          </button>
        </div>
        {#if syncError}
          <div class="mt-2 p-2 bg-red-500/10 border border-red-500/30 rounded text-red-400 text-xs">{syncError}</div>
        {/if}
        {#if syncDiff}
          <div class="mt-2 p-3 bg-slate-800/50 border border-slate-700/50 rounded-lg text-sm">
            <div class="flex items-center justify-between mb-2">
              <span class="font-medium text-slate-300">Compose updated</span>
              {#if syncCommit}
                <span class="text-xs text-slate-500 font-mono">{syncCommit.slice(0, 7)}</span>
              {/if}
            </div>
            {#if syncDiff.added.length > 0}
              <div class="mb-1">
                <span class="text-emerald-400 text-xs font-medium">Added:</span>
                {#each syncDiff.added as vol}
                  <span class="ml-1 text-xs text-emerald-400">{vol.name}</span>
                {/each}
              </div>
            {/if}
            {#if syncDiff.removed.length > 0}
              <div class="mb-1">
                <span class="text-red-400 text-xs font-medium">Removed:</span>
                {#each syncDiff.removed as vol}
                  <span class="ml-1 text-xs text-red-400">{vol.name}</span>
                {/each}
              </div>
            {/if}
            {#if syncDiff.unchanged.length > 0}
              <div class="mb-1">
                <span class="text-slate-500 text-xs font-medium">Unchanged:</span>
                {#each syncDiff.unchanged as vol}
                  <span class="ml-1 text-xs text-slate-500">{vol.name}</span>
                {/each}
              </div>
            {/if}
            <button
              onclick={dismissSyncDiff}
              class="mt-2 text-xs text-slate-500 hover:text-slate-300 underline"
            >
              Dismiss
            </button>
          </div>
        {/if}
      {/if}

      <div class="mt-3 flex items-center gap-3">
        <button
          onclick={() => (showComposeRestore = true)}
          class="px-4 py-2 text-sm text-white bg-gradient-to-r from-indigo-500 to-violet-500 hover:from-indigo-400 hover:to-violet-400 rounded-lg transition-all duration-200 active:scale-[0.98]"
        >
          Restore Compose
        </button>
        <button
          onclick={() => (showComposeEdit = true)}
          class="px-4 py-2 text-sm text-slate-300 border border-slate-600 hover:bg-slate-800 rounded-lg transition-all duration-200"
        >
          {project.git_source || project.portainer_source ? 'View Compose' : 'Edit Compose'}
        </button>
        {#if project.deployment_mode}
          <div class="relative inline-block text-left">
            <button
              onclick={() => showDeploymentModeDropdown = !showDeploymentModeDropdown}
              class="inline-flex items-center gap-1.5 text-xs font-medium px-2.5 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 border border-slate-700 transition-colors cursor-pointer"
            >
              <span>Mode: {project.deployment_mode === 'swarm' ? 'Swarm' : 'Standalone'}</span>
              <svg class="w-3 h-3 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M19 9l-7 7-7-7" />
              </svg>
            </button>
            {#if showDeploymentModeDropdown}
              <button
                type="button"
                onclick={() => showDeploymentModeDropdown = false}
                class="fixed inset-0 cursor-default z-10"
                tabindex="-1"
                aria-label="Close dropdown"
              ></button>
              <div class="absolute left-0 mt-1.5 w-36 rounded-lg shadow-xl bg-slate-800 border border-slate-700 z-20">
                <div class="py-1">
                  <button
                    onclick={() => updateDeploymentMode('standalone')}
                    class="block w-full text-left px-4 py-2 text-xs text-slate-300 hover:bg-slate-700 cursor-pointer {project.deployment_mode === 'standalone' ? 'font-semibold text-indigo-400' : ''}"
                  >
                    Standalone
                  </button>
                  <button
                    onclick={() => updateDeploymentMode('swarm')}
                    class="block w-full text-left px-4 py-2 text-xs text-slate-300 hover:bg-slate-700 cursor-pointer {project.deployment_mode === 'swarm' ? 'font-semibold text-indigo-400' : ''}"
                  >
                    Swarm
                  </button>
                </div>
              </div>
            {/if}
          </div>
        {/if}
        {#if project.deployment_mode === 'swarm'}
          <div class="mt-2 flex items-center gap-2">
            <span class="text-xs text-slate-500">Stack name:</span>
            {#if editingSwarmName}
              <input
                type="text"
                bind:value={swarmNameInput}
                placeholder={project.name}
                class="px-2 py-1 text-xs border border-slate-600 rounded-lg bg-slate-900 text-slate-100 focus:outline-none focus:ring-1 focus:ring-indigo-500 w-40"
              />
              <button
                onclick={saveSwarmName}
                class="text-xs px-2 py-1 text-white bg-indigo-600 hover:bg-indigo-500 rounded-lg transition-colors"
              >
                Save
              </button>
              <button
                onclick={() => (editingSwarmName = false)}
                class="text-xs px-2 py-1 text-slate-400 border border-slate-600 rounded-lg hover:bg-slate-800 transition-colors"
              >
                Cancel
              </button>
            {:else}
              <span class="text-xs font-mono text-slate-300">{project.swarm_name || project.name}</span>
              <button
                onclick={startEditSwarmName}
                class="text-xs px-2 py-1 text-slate-400 border border-slate-600 rounded-lg hover:bg-slate-800 transition-colors"
              >
                {project.swarm_name ? 'Edit' : 'Override'}
              </button>
            {/if}
          </div>
        {/if}
      </div>

      <!-- Storage config summary -->
      <div class="mt-4 flex items-center gap-3">
        {#if existingCredentials}
          <div class="flex items-center gap-2">
            <span
              class="inline-flex items-center text-xs font-medium px-2 py-0.5 rounded bg-slate-700/50 text-slate-300 border border-slate-600 uppercase tracking-wide"
            >
              {existingCredentials.type}
            </span>
            <span class="text-sm text-slate-400">
              {#if existingCredentials.type === 'local' && existingCredentials.local}
                {existingCredentials.local.path}
              {:else if existingCredentials.type === 'smb' && existingCredentials.smb}
                {existingCredentials.smb.host}/{existingCredentials.smb.share}
              {/if}
            </span>
          </div>
        {:else}
          <span class="text-sm text-slate-500">No storage configured</span>
        {/if}
        <button
          onclick={openStorageEdit}
          class="px-3 py-1 text-xs text-slate-400 border border-slate-600 hover:bg-slate-800 hover:text-slate-300 rounded-lg transition-colors"
        >
          {existingCredentials ? 'Edit Storage' : 'Add Storage'}
        </button>
      </div>
    </div>

    <!-- Default passphrase section -->
    <div class="mb-6 bg-slate-800/50 rounded-xl border border-slate-700/50 px-4 py-3">
      <div class="flex items-center justify-between">
        <div>
          <div class="flex items-center gap-2">
            <span class="text-sm font-medium text-slate-300">Default Passphrase</span>
            {#if project.default_passphrase}
              <span
                class="inline-flex items-center gap-1 text-xs text-emerald-400 bg-emerald-500/10 border border-emerald-500/30 rounded px-1.5 py-0.5"
              >
                <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                </svg>
                Stored
              </span>
            {/if}
          </div>
          <p class="text-xs text-slate-500 mt-0.5">
            Used for encrypted restores when no volume-specific passphrase is set.
          </p>
        </div>
        <div class="flex items-center gap-2">
          {#if project.default_passphrase}
            <button
              onclick={() => (showDefaultPassphraseForm = !showDefaultPassphraseForm)}
              class="text-xs px-2 py-1 border border-slate-600 rounded hover:bg-slate-700 transition-colors text-slate-400"
              disabled={defaultPassphraseLoading}
            >
              Change
            </button>
            <button
              onclick={handleClearDefaultPassphrase}
              class="text-xs px-2 py-1 border border-red-500/30 rounded hover:bg-red-500/10 transition-colors text-red-400"
              disabled={defaultPassphraseLoading}
            >
              {defaultPassphraseLoading ? '...' : 'Clear'}
            </button>
          {:else}
            <button
              onclick={() => (showDefaultPassphraseForm = !showDefaultPassphraseForm)}
              class="text-xs px-2 py-1 border border-slate-600 rounded hover:bg-slate-700 transition-colors text-slate-400"
            >
              Set Default Passphrase
            </button>
          {/if}
        </div>
      </div>

      {#if showDefaultPassphraseForm}
        <div class="mt-3 pt-3 border-t border-slate-700/50">
          <label for="default-passphrase" class="block text-sm font-medium text-slate-300 mb-1">
            {project.default_passphrase ? 'Change Default Passphrase' : 'Set Default Passphrase'}
          </label>
          <div class="flex gap-2">
            <input
              id="default-passphrase"
              type="password"
              bind:value={defaultPassphraseInput}
              placeholder="Enter passphrase"
              class="flex-1 bg-slate-900 border border-slate-600 rounded-lg px-3 py-1.5 text-sm text-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500"
            />
            <button
              onclick={handleSetDefaultPassphrase}
              disabled={defaultPassphraseLoading || !defaultPassphraseInput.trim()}
              class="px-3 py-1.5 text-sm text-white bg-gradient-to-r from-indigo-500 to-violet-500 hover:from-indigo-400 hover:to-violet-400 rounded-lg transition-all disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {defaultPassphraseLoading ? 'Saving...' : 'Save'}
            </button>
            <button
              onclick={cancelDefaultPassphrase}
              class="px-3 py-1.5 text-sm text-slate-300 border border-slate-600 rounded-lg hover:bg-slate-700 transition-colors"
            >
              Cancel
            </button>
          </div>
          {#if defaultPassphraseError}
            <div class="mt-2 text-sm text-red-400">{defaultPassphraseError}</div>
          {/if}
        </div>
      {/if}
    </div>

    <div class="flex flex-col gap-6">
      {#each groupedVolumes as group (group.service)}
        <div>
          <h3 class="text-sm font-medium text-slate-500 uppercase tracking-wide mb-2">{group.service}</h3>
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
    stackName={project.swarm_name || project.name}
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
    stackName={project.swarm_name || project.name}
    onClose={() => (showComposeRestore = false)}
  />
{/if}

{#if showComposeEdit && project}
  <ComposeEditModal
    projectId={project.id}
    initialContent={project.compose_content}
    onClose={() => (showComposeEdit = false)}
    onSaved={() => loadProject(project!.id)}
    readOnly={!!project.git_source || !!project.portainer_source}
    source={project.source}
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
