<script lang="ts">
  import type { VolumeInfo, SavedSource } from '$lib/types';
  import { importGit, listSavedSources, createSavedSource, getSavedSource, browseGit } from '$lib/api';
  import { onMount, untrack } from 'svelte';

  interface Props {
    onComplete: (projectId: string, volumes: VolumeInfo[], name: string) => void;
    onBack?: () => void;
  }

  let { onComplete, onBack }: Props = $props();

  let repoUrl = $state('');
  let branch = $state('main');
  let filePath = $state('docker-compose.yml');
  let projectName = $state('');
  let userEditedProjectName = $state(false);

  $effect(() => {
    if (userEditedProjectName) return;

    let inferred = '';
    const cleanRepo = repoUrl.trim().replace(/\.git$/, '');
    const repoName = cleanRepo.split('/').pop() || '';

    if (filePath) {
      const parts = filePath.split('/');
      if (parts.length > 1) {
        const folder = parts[parts.length - 2];
        const genericNames = ['deploy', 'deployment', 'docker', 'compose', 'config', 'setup', 'src', 'app', 'configs', 'local', 'prod', 'production', 'dev', 'development', 'staging'];
        if (folder && !genericNames.includes(folder.toLowerCase())) {
          inferred = folder;
        }
      }
    }

    if (!inferred && repoName) {
      inferred = repoName;
    }

    projectName = inferred;
  });
  let authType = $state<'none' | 'token' | 'ssh'>('none');
  let authToken = $state('');
  let sshPrivateKey = $state('');
  let deploymentMode = $state<'standalone' | 'swarm'>('standalone');
  let error = $state('');
  let loading = $state(false);

  let savedSources = $state<SavedSource[]>([]);
  let selectedSourceId = $state('');
  let saveSource = $state(false);
  let sourceName = $state('');

  // Git File Browser state
  let selectionMode = $state<'auto' | 'manual'>('auto');
  let gitFiles = $state<string[]>([]);
  let currentPath = $state('');
  let gitBrowseLoading = $state(false);
  let gitBrowseError = $state('');
  let gitBrowseLoaded = $state(false);

  let isGitDetailsValid = $derived.by(() => {
    if (!repoUrl.trim()) return false;
    if (!branch.trim()) return false;
    if (authType === 'token' && !authToken.trim()) return false;
    if (authType === 'ssh' && !sshPrivateKey.trim()) return false;
    return true;
  });

  $effect(() => {
    // track changes to these inputs to reset browse results
    repoUrl;
    branch;
    authToken;
    sshPrivateKey;
    authType;
    selectedSourceId;

    untrack(() => {
      gitBrowseLoaded = false;
      gitFiles = [];
      gitBrowseError = '';
    });
  });

  async function loadRepoFiles() {
    if (!isGitDetailsValid) return;
    gitBrowseLoading = true;
    gitBrowseError = '';
    try {
      const response = await browseGit({
        repo_url: repoUrl.trim(),
        branch: branch.trim(),
        auth_token: authType === 'token' ? authToken.trim() : undefined,
        ssh_private_key: authType === 'ssh' ? sshPrivateKey.trim() : undefined,
        saved_source_id: selectedSourceId || undefined
      });
      gitFiles = response.files;
      gitBrowseLoaded = true;
      currentPath = ''; // start at root
    } catch (e) {
      gitBrowseError = e instanceof Error ? e.message : 'Failed to retrieve repository files';
    } finally {
      gitBrowseLoading = false;
    }
  }

  let currentItems = $derived.by(() => {
    const folders = new Set<string>();
    const files: string[] = [];

    const prefix = currentPath ? currentPath + '/' : '';
    for (const file of gitFiles) {
      if (file.startsWith(prefix)) {
        const relative = file.substring(prefix.length);
        const slashIdx = relative.indexOf('/');
        if (slashIdx === -1) {
          files.push(relative);
        } else {
          folders.add(relative.substring(0, slashIdx));
        }
      }
    }
    return {
      folders: Array.from(folders).sort(),
      files: files.sort()
    };
  });

  function enterFolder(folder: string) {
    currentPath = currentPath ? `${currentPath}/${folder}` : folder;
  }

  function navigateToBreadcrumb(index: number) {
    if (index === -1) {
      currentPath = '';
      return;
    }
    const parts = currentPath.split('/');
    currentPath = parts.slice(0, index + 1).join('/');
  }

  function navigateUp() {
    const parts = currentPath.split('/');
    if (parts.length <= 1) {
      currentPath = '';
    } else {
      currentPath = parts.slice(0, -1).join('/');
    }
  }

  function selectFile(file: string) {
    const fullPath = currentPath ? `${currentPath}/${file}` : file;
    filePath = fullPath;
  }

  onMount(async () => {
    try {
      const all = await listSavedSources();
      savedSources = all.filter((s) => s.type === 'git');
    } catch {
      // ignore — saved sources are optional
    }
  });

  async function handleSourceSelect(id: string) {
    selectedSourceId = id;
    if (id === '') return;
    try {
      const source = await getSavedSource(id);
      if (source?.git_config) {
        repoUrl = source.git_config.repo_url ?? '';
        branch = source.git_config.branch ?? 'main';
        filePath = source.git_config.file_path ?? 'docker-compose.yml';
        authToken = source.git_config.auth_token ?? '';
        sshPrivateKey = source.git_config.ssh_private_key ?? '';
        if (source.git_config.auth_token) {
          authType = 'token';
        } else if (source.git_config.ssh_private_key) {
          authType = 'ssh';
        } else {
          authType = 'none';
        }
      }
    } catch {
      // Fall back to the sanitized data from the list
      const source = savedSources.find((s) => s.id === id);
      if (source?.git_config) {
        repoUrl = source.git_config.repo_url ?? '';
        branch = source.git_config.branch ?? 'main';
        filePath = source.git_config.file_path ?? 'docker-compose.yml';
        authType = 'none';
        authToken = '';
        sshPrivateKey = '';
      }
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
        deployment_mode: deploymentMode,
        auth_token: authType === 'token' ? authToken.trim() : undefined,
        ssh_private_key: authType === 'ssh' ? sshPrivateKey.trim() : undefined,
        saved_source_id: selectedSourceId || undefined
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
    <label for="repo-url" class="block text-sm font-medium text-slate-300 mb-1">
      Repository URL
    </label>
    <input
      id="repo-url"
      type="text"
      bind:value={repoUrl}
      class="w-full px-3 py-2 border border-slate-600 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-slate-950/40"
      placeholder="https://github.com/user/repo.git"
    />
  </div>

  <div class="grid grid-cols-2 gap-4">
    <div>
      <label for="branch" class="block text-sm font-medium text-slate-300 mb-1">
        Branch
      </label>
      <input
        id="branch"
        type="text"
        bind:value={branch}
        class="w-full px-3 py-2 border border-slate-600 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-slate-950/40"
        placeholder="main"
      />
    </div>
    <div>
      <label for="project-name" class="block text-sm font-medium text-slate-300 mb-1">
        Project Name
      </label>
      <input
        id="project-name"
        type="text"
        bind:value={projectName}
        oninput={() => (userEditedProjectName = true)}
        class="w-full px-3 py-2 border border-slate-600 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-slate-950/40"
        placeholder="e.g. homelab, production"
      />
    </div>
  </div>

  {#if !selectedSourceId}
    <div>
      <span class="block text-sm font-medium text-slate-300 mb-2">Authentication</span>
      <div class="flex gap-3">
        <button
          type="button"
          onclick={() => (authType = 'none')}
          class="px-3 py-1.5 text-sm rounded-lg border transition-colors {authType === 'none'
            ? 'bg-indigo-500/10 border-indigo-500/30 text-indigo-400 font-medium'
            : 'border-slate-600 text-slate-400 hover:bg-slate-800/50'}"
        >
          None
        </button>
        <button
          type="button"
          onclick={() => (authType = 'token')}
          class="px-3 py-1.5 text-sm rounded-lg border transition-colors {authType === 'token'
            ? 'bg-indigo-500/10 border-indigo-500/30 text-indigo-400 font-medium'
            : 'border-slate-600 text-slate-400 hover:bg-slate-800/50'}"
        >
          Token
        </button>
        <button
          type="button"
          onclick={() => (authType = 'ssh')}
          class="px-3 py-1.5 text-sm rounded-lg border transition-colors {authType === 'ssh'
            ? 'bg-indigo-500/10 border-indigo-500/30 text-indigo-400 font-medium'
            : 'border-slate-600 text-slate-400 hover:bg-slate-800/50'}"
        >
          SSH Key
        </button>
      </div>
    </div>

    {#if authType === 'token'}
      <div>
        <label for="auth-token" class="block text-sm font-medium text-slate-300 mb-1">
          Access Token
        </label>
        <input
          id="auth-token"
          type="password"
          bind:value={authToken}
          class="w-full px-3 py-2 border border-slate-600 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-slate-950/40"
          placeholder="ghp_... or personal access token"
        />
      </div>
    {/if}

    {#if authType === 'ssh'}
      <div>
        <label for="ssh-key" class="block text-sm font-medium text-slate-300 mb-1">
          SSH Private Key
        </label>
        <textarea
          id="ssh-key"
          bind:value={sshPrivateKey}
          rows="5"
          class="w-full px-3 py-2 border border-slate-600 rounded-lg font-mono text-xs focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-slate-950/40"
          placeholder="-----BEGIN OPENSSH PRIVATE KEY-----&#10;...&#10;-----END OPENSSH PRIVATE KEY-----"
        ></textarea>
      </div>
    {/if}
  {/if}

  <!-- Compose File Path Section -->
  <div class="space-y-2 border border-slate-800 rounded-xl p-4 bg-slate-900/10 backdrop-blur-md">
    <div class="flex items-center justify-between mb-1">
      <span class="text-sm font-medium text-slate-300">Compose File Path</span>
      <div class="flex bg-slate-800/80 rounded-lg p-0.5 text-[11px] border border-slate-700">
        <button
          type="button"
          onclick={() => (selectionMode = 'auto')}
          class="px-2.5 py-0.5 rounded-md transition-all font-medium {selectionMode === 'auto' ? 'bg-indigo-650 text-white shadow' : 'text-slate-400 hover:text-slate-200'}"
        >
          Browse Repo
        </button>
        <button
          type="button"
          onclick={() => (selectionMode = 'manual')}
          class="px-2.5 py-0.5 rounded-md transition-all font-medium {selectionMode === 'manual' ? 'bg-indigo-650 text-white shadow' : 'text-slate-400 hover:text-slate-200'}"
        >
          Manual Override
        </button>
      </div>
    </div>

    {#if selectionMode === 'manual'}
      <div class="space-y-1">
        <input
          id="file-path"
          type="text"
          bind:value={filePath}
          class="w-full px-3 py-2 border border-slate-600 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-slate-950/40"
          placeholder="docker-compose.yml"
        />
        <p class="text-[10px] text-slate-500">
          Enter the relative path to your docker-compose file (e.g. <span class="font-mono">docker-compose.yml</span> or <span class="font-mono">deploy/compose.yaml</span>).
        </p>
      </div>
    {:else}
      {#if !isGitDetailsValid}
        <div class="p-4 bg-slate-900/30 border border-slate-800/40 rounded-lg flex flex-col items-center justify-center text-center space-y-1.5 py-6">
          <svg class="w-7 h-7 text-slate-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
          <div class="text-[11px] text-slate-400 font-semibold">Repository details incomplete</div>
          <p class="text-[10px] text-slate-500 max-w-xs leading-normal">
            Please fill in the Git Repository URL, Branch, and any required Authentication above to enable repository browsing.
          </p>
        </div>
      {:else if gitBrowseLoading}
        <div class="p-6 bg-slate-900/20 border border-slate-800/40 rounded-lg flex flex-col items-center justify-center space-y-2.5">
          <div class="animate-spin rounded-full h-5 w-5 border-2 border-indigo-500 border-t-transparent"></div>
          <span class="text-xs text-slate-400">Connecting to repository and scanning files...</span>
        </div>
      {:else if gitBrowseError}
        <div class="p-4 bg-red-500/10 border border-red-500/25 rounded-lg space-y-2">
          <p class="text-xs text-red-400 leading-normal">{gitBrowseError}</p>
          <button
            type="button"
            onclick={loadRepoFiles}
            class="px-2.5 py-1 bg-red-500/20 hover:bg-red-500/30 text-red-300 text-[11px] rounded font-medium transition-colors"
          >
            Retry Connection
          </button>
        </div>
      {:else if !gitBrowseLoaded}
        <div class="p-4 bg-slate-900/30 border border-slate-800/40 rounded-lg flex flex-col items-center justify-center text-center space-y-2.5 py-6">
          <svg class="w-7 h-7 text-indigo-450/80" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
          </svg>
          <div class="text-xs text-slate-300 font-semibold">Ready to connect</div>
          <p class="text-[10px] text-slate-500 max-w-xs leading-normal">
            Connect to the repository to browse its file structure and locate Docker Compose files.
          </p>
          <button
            type="button"
            onclick={loadRepoFiles}
            class="px-3.5 py-1.5 bg-indigo-600 hover:bg-indigo-500 text-white text-[11px] font-semibold rounded-lg shadow-md transition-all duration-200 active:scale-95 hover:shadow-indigo-500/10"
          >
            Connect & Browse
          </button>
        </div>
      {:else}
        <div class="space-y-2.5 bg-slate-950/40 rounded-xl p-3 border border-slate-800">
          <div class="flex flex-col gap-1.5 border-b border-slate-800 pb-2">
            <div class="flex items-center justify-between text-[11px]">
              <span class="text-slate-400 font-medium">Selected Path:</span>
              <span class="font-mono text-indigo-400 font-semibold bg-indigo-500/10 px-2 py-0.5 rounded truncate max-w-[280px]">
                {filePath || 'None selected'}
              </span>
            </div>

            <div class="flex items-center gap-1.5 text-[11px] text-slate-400 flex-wrap pt-0.5 font-mono">
              <button
                type="button"
                onclick={() => navigateToBreadcrumb(-1)}
                class="hover:text-indigo-400 transition-colors flex items-center gap-1 font-medium"
              >
                📁 root
              </button>
              {#each currentPath.split('/').filter(Boolean) as part, idx}
                <span class="text-slate-750">/</span>
                <button
                  type="button"
                  onclick={() => navigateToBreadcrumb(idx)}
                  class="hover:text-indigo-400 transition-colors"
                >
                  {part}
                </button>
              {/each}
            </div>
          </div>

          <div class="max-h-56 overflow-y-auto space-y-0.5 pr-1">
            {#if currentPath !== ""}
              <button
                type="button"
                onclick={navigateUp}
                class="w-full text-left px-2.5 py-1.5 rounded-lg hover:bg-slate-800/55 text-slate-400 text-xs font-semibold transition-colors flex items-center gap-2"
              >
                <span class="text-slate-500 text-xs">↩</span>
                <span>.. (Go Up)</span>
              </button>
            {/if}

            {#if currentItems.folders.length === 0 && currentItems.files.length === 0}
              <div class="text-center py-6 text-slate-500 text-xs italic">
                No matching compose files in this directory.
              </div>
            {:else}
              {#each currentItems.folders as folder}
                <button
                  type="button"
                  onclick={() => enterFolder(folder)}
                  class="w-full text-left px-2.5 py-1.5 rounded-lg hover:bg-slate-800/40 text-slate-200 text-xs font-medium transition-colors flex items-center justify-between group"
                >
                  <div class="flex items-center gap-2">
                    <span class="text-amber-500/80">📁</span>
                    <span class="truncate">{folder}</span>
                  </div>
                  <span class="text-slate-700 group-hover:text-indigo-400 transition-colors text-[10px] font-mono">&rarr;</span>
                </button>
              {/each}

              {#each currentItems.files as file}
                {@const fullFile = currentPath ? `${currentPath}/${file}` : file}
                {@const isSelected = filePath === fullFile}
                <button
                  type="button"
                  onclick={() => selectFile(file)}
                  class="w-full text-left px-2.5 py-1.5 rounded-lg text-xs transition-colors flex items-center justify-between group {isSelected ? 'bg-indigo-500/25 text-indigo-300 font-semibold border border-indigo-500/30' : 'hover:bg-slate-800/30 text-slate-300'}"
                >
                  <div class="flex items-center gap-2 min-w-0">
                    <span class="text-indigo-400/80 text-sm">📄</span>
                    <span class="truncate">{file}</span>
                  </div>
                  {#if isSelected}
                    <span class="text-indigo-400 text-[9px] font-semibold bg-indigo-500/20 px-1.5 py-0.5 rounded">Selected</span>
                  {/if}
                </button>
              {/each}
            {/if}
          </div>
        </div>
      {/if}
    {/if}
  </div>

  <div>
    <p class="text-sm font-medium text-slate-300 mb-2">Deployment Mode</p>
    <div class="flex gap-3">
      <button
        type="button"
        onclick={() => (deploymentMode = 'standalone')}
        class="flex-1 py-2 px-4 rounded-lg border-2 text-sm font-medium transition-colors {deploymentMode === 'standalone' ? 'border-indigo-500 bg-indigo-500/10 text-indigo-400' : 'border-slate-850 text-slate-400 hover:border-slate-700'}"
      >
        Standalone
      </button>
      <button
        type="button"
        onclick={() => (deploymentMode = 'swarm')}
        class="flex-1 py-2 px-4 rounded-lg border-2 text-sm font-medium transition-colors {deploymentMode === 'swarm' ? 'border-indigo-500 bg-indigo-500/10 text-indigo-400' : 'border-slate-850 text-slate-400 hover:border-slate-700'}"
      >
        Docker Swarm
      </button>
    </div>
    <p class="text-xs text-slate-500 mt-1">
      {deploymentMode === 'swarm' ? 'Volumes are prefixed with the stack name (e.g., mystack_data)' : 'Volumes are prefixed with the project name (e.g., myproject_data)'}
    </p>
  </div>

  {#if !selectedSourceId}
    <div class="flex items-start gap-2">
      <input
        id="save-source"
        type="checkbox"
        bind:checked={saveSource}
        class="mt-0.5 rounded border-slate-600 text-indigo-400 focus:ring-indigo-500"
      />
      <div class="flex-1">
        <label for="save-source" class="text-sm font-medium text-slate-300">
          Save this source for reuse
        </label>
        {#if saveSource}
          <input
            type="text"
            bind:value={sourceName}
            class="mt-1 w-full px-3 py-2 border border-slate-600 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-slate-950/40"
            placeholder="Source name (optional, defaults to repo name)"
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
      onclick={handleImport}
      disabled={loading}
      class="{onBack ? 'flex-1' : 'w-full'} py-2 px-4 bg-gradient-to-r from-indigo-500 to-violet-500 hover:from-indigo-400 hover:to-violet-400 disabled:opacity-50 text-white rounded-lg font-medium transition-all duration-200 active:scale-[0.98]"
    >
      {loading ? 'Importing...' : 'Import'}
    </button>
  </div>
</div>
