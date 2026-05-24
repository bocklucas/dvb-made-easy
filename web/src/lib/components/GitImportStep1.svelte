<script lang="ts">
  import type { VolumeInfo, SavedSource } from '$lib/types';
  import { importGit, listSavedSources, createSavedSource } from '$lib/api';
  import { onMount } from 'svelte';

  interface Props {
    onComplete: (projectId: string, volumes: VolumeInfo[], name: string) => void;
    onBack?: () => void;
  }

  let { onComplete, onBack }: Props = $props();

  let repoUrl = $state('');
  let branch = $state('main');
  let filePath = $state('docker-compose.yml');
  let projectName = $state('');
  let authType = $state<'none' | 'token' | 'ssh'>('none');
  let authToken = $state('');
  let sshPrivateKey = $state('');
  let error = $state('');
  let loading = $state(false);

  let savedSources = $state<SavedSource[]>([]);
  let selectedSourceId = $state('');
  let saveSource = $state(false);
  let sourceName = $state('');

  onMount(async () => {
    try {
      const all = await listSavedSources();
      savedSources = all.filter((s) => s.type === 'git');
    } catch {
      // ignore — saved sources are optional
    }
  });

  function handleSourceSelect(id: string) {
    selectedSourceId = id;
    if (id === '') return;
    const source = savedSources.find((s) => s.id === id);
    if (source?.git_config) {
      repoUrl = source.git_config.repo_url ?? '';
      branch = source.git_config.branch ?? 'main';
      filePath = source.git_config.file_path ?? 'docker-compose.yml';
      // Auth fields are sanitized in the API response, so we clear them
      authType = 'none';
      authToken = '';
      sshPrivateKey = '';
    }
  }

  async function handleImport() {
    error = '';
    if (!repoUrl.trim()) {
      error = 'Repository URL is required';
      return;
    }
    if (!projectName.trim()) {
      error = 'Project name is required';
      return;
    }

    loading = true;
    try {
      const result = await importGit({
        repo_url: repoUrl.trim(),
        branch: branch.trim() || 'main',
        file_path: filePath.trim() || 'docker-compose.yml',
        project_name: projectName.trim(),
        auth_token: authType === 'token' ? authToken.trim() : undefined,
        ssh_private_key: authType === 'ssh' ? sshPrivateKey.trim() : undefined
      });

      if (saveSource) {
        const name = sourceName.trim() || repoUrl.trim().replace(/\.git$/, '').split('/').pop() || 'Git Source';
        try {
          await createSavedSource({
            name,
            type: 'git',
            git_config: {
              repo_url: repoUrl.trim(),
              branch: branch.trim() || 'main',
              file_path: filePath.trim() || 'docker-compose.yml',
              auth_token: authType === 'token' ? authToken.trim() : undefined,
              ssh_private_key: authType === 'ssh' ? sshPrivateKey.trim() : undefined
            }
          });
        } catch {
          // saving the source is best-effort; don't block the import
        }
      }

      onComplete(result.id, result.volumes, projectName.trim());
    } catch (e) {
      error = e instanceof Error ? e.message : 'Import failed';
    } finally {
      loading = false;
    }
  }
</script>

<div class="space-y-5">
  {#if savedSources.length > 0}
    <div>
      <label for="saved-source" class="block text-sm font-medium text-gray-700 mb-1">
        Use a saved source
      </label>
      <select
        id="saved-source"
        value={selectedSourceId}
        onchange={(e) => handleSourceSelect(e.currentTarget.value)}
        class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500 bg-white"
      >
        <option value="">Enter manually</option>
        {#each savedSources as source}
          <option value={source.id}>{source.name}</option>
        {/each}
      </select>
    </div>
  {/if}

  <div>
    <label for="repo-url" class="block text-sm font-medium text-gray-700 mb-1">
      Repository URL
    </label>
    <input
      id="repo-url"
      type="text"
      bind:value={repoUrl}
      class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
      placeholder="https://github.com/user/repo.git"
    />
  </div>

  <div class="grid grid-cols-2 gap-4">
    <div>
      <label for="branch" class="block text-sm font-medium text-gray-700 mb-1">
        Branch
      </label>
      <input
        id="branch"
        type="text"
        bind:value={branch}
        class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
        placeholder="main"
      />
    </div>
    <div>
      <label for="file-path" class="block text-sm font-medium text-gray-700 mb-1">
        Compose File Path
      </label>
      <input
        id="file-path"
        type="text"
        bind:value={filePath}
        class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
        placeholder="docker-compose.yml"
      />
    </div>
  </div>

  <div>
    <label for="project-name" class="block text-sm font-medium text-gray-700 mb-1">
      Project Name
    </label>
    <input
      id="project-name"
      type="text"
      bind:value={projectName}
      class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
      placeholder="e.g. homelab, production"
    />
  </div>

  <div>
    <span class="block text-sm font-medium text-gray-700 mb-2">Authentication</span>
    <div class="flex gap-3">
      <button
        onclick={() => (authType = 'none')}
        class="px-3 py-1.5 text-sm rounded-lg border transition-colors {authType === 'none'
          ? 'bg-blue-50 border-blue-300 text-blue-700'
          : 'border-gray-300 text-gray-600 hover:bg-gray-50'}"
      >
        None
      </button>
      <button
        onclick={() => (authType = 'token')}
        class="px-3 py-1.5 text-sm rounded-lg border transition-colors {authType === 'token'
          ? 'bg-blue-50 border-blue-300 text-blue-700'
          : 'border-gray-300 text-gray-600 hover:bg-gray-50'}"
      >
        Token
      </button>
      <button
        onclick={() => (authType = 'ssh')}
        class="px-3 py-1.5 text-sm rounded-lg border transition-colors {authType === 'ssh'
          ? 'bg-blue-50 border-blue-300 text-blue-700'
          : 'border-gray-300 text-gray-600 hover:bg-gray-50'}"
      >
        SSH Key
      </button>
    </div>
  </div>

  {#if authType === 'token'}
    <div>
      <label for="auth-token" class="block text-sm font-medium text-gray-700 mb-1">
        Access Token
      </label>
      <input
        id="auth-token"
        type="password"
        bind:value={authToken}
        class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
        placeholder="ghp_... or personal access token"
      />
    </div>
  {/if}

  {#if authType === 'ssh'}
    <div>
      <label for="ssh-key" class="block text-sm font-medium text-gray-700 mb-1">
        SSH Private Key
      </label>
      <textarea
        id="ssh-key"
        bind:value={sshPrivateKey}
        rows="6"
        class="w-full px-3 py-2 border border-gray-300 rounded-lg font-mono text-xs focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
        placeholder="-----BEGIN OPENSSH PRIVATE KEY-----&#10;...&#10;-----END OPENSSH PRIVATE KEY-----"
      ></textarea>
    </div>
  {/if}

  <div class="flex items-start gap-2">
    <input
      id="save-source"
      type="checkbox"
      bind:checked={saveSource}
      class="mt-0.5 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
    />
    <div class="flex-1">
      <label for="save-source" class="text-sm font-medium text-gray-700">
        Save this source for reuse
      </label>
      {#if saveSource}
        <input
          type="text"
          bind:value={sourceName}
          class="mt-1 w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
          placeholder="Source name (optional, defaults to repo name)"
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
      onclick={handleImport}
      disabled={loading}
      class="{onBack ? 'flex-1' : 'w-full'} py-2 px-4 bg-blue-600 hover:bg-blue-500 disabled:bg-blue-300 text-white rounded-lg font-medium transition-colors"
    >
      {loading ? 'Importing...' : 'Import'}
    </button>
  </div>
</div>
