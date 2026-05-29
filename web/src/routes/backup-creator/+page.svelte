<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import * as projectState from '$lib/state/projects.svelte';
  import type { VolumeInfo, SavedBackend, SavedSource, PortainerEndpoint, PortainerStack, Credentials } from '$lib/types';
  import {
    listSavedBackends,
    listSavedSources,
    getSavedSource,
    portainerConnect,
    portainerStacks as apiPortainerStacks,
    parseBCCompose,
    fetchBCGit,
    fetchBCPortainer,
    generateBCCompose,
    importCompose,
    importGit,
    importPortainer,
    updateCompose,
    saveCredentials,
    createProject,
    listProjects
  } from '$lib/api';

  // Wizard state
  let step = $state(1);
  let loading = $state(false);
  let error = $state('');

  // Step 1: Starting Point
  let sourceType = $state<'paste' | 'git' | 'portainer' | 'project'>('paste');
  let composeContent = $state('');
  let projectName = $state('');
  let userEditedProjectName = $state(false);

  $effect(() => {
    if (userEditedProjectName) return;

    if (sourceType === 'git') {
      let inferred = '';
      const cleanRepo = gitRepoUrl.trim().replace(/\.git$/, '');
      const repoName = cleanRepo.split('/').pop() || '';

      if (gitFilePath) {
        const parts = gitFilePath.split('/');
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
    } else if (sourceType === 'paste') {
      const match = composeContent.match(/^name:\s*["']?([^"'\s#\n]+)["']?/m);
      if (match && match[1]) {
        projectName = match[1];
      }
    }
  });

  // Existing Projects
  import type { Project } from '$lib/types';
  let projectsList = $state<Project[]>([]);
  let selectedProjectId = $state('');

  // Saved Sources
  let savedSources = $state<SavedSource[]>([]);
  let selectedSavedSourceId = $state('');

  // Git specific settings
  let gitRepoUrl = $state('');
  let gitBranch = $state('main');
  let gitFilePath = $state('docker-compose.yml');
  let gitAuthToken = $state('');
  let gitSshKey = $state('');

  // Portainer specific settings
  let portainerUrl = $state('');
  let portainerApiKey = $state('');
  let portainerConnected = $state(false);
  let endpointsList = $state<PortainerEndpoint[]>([]);
  let selectedPortainerEndpointId = $state<number | null>(null);
  let stacksList = $state<PortainerStack[]>([]);
  let selectedPortainerStackId = $state<number | null>(null);
  let stacksLoading = $state(false);

  // Step 2 & 3: Parsed elements
  let parsedVolumes = $state<VolumeInfo[]>([]);
  let parsedServices = $state<string[]>([]);
  let selectedVolumes = $state<string[]>([]);
  let stopServices = $state<string[]>([]);
  let volumeFilenameFormats = $state<Record<string, string>>({});

  // Step 4: Backup Options
  let backupServiceName = $state('backup');
  let backupImage = $state('offen/docker-volume-backup:v2');
  let cronPreset = $state<'daily' | 'hourly' | 'weekly' | 'monthly' | 'custom'>('daily');
  let cronCustom = $state('0 2 * * *');
  let filenameFormat = $state('backup-%Y-%m-%dT%H-%M-%S.tar.gz');
  let enableGpg = $state(false);
  let gpgPassphrase = $state('');
  let retentionDays = $state(14);

  // Step 5: Destination Storage
  let backendType = $state<'local' | 'smb' | 's3' | 'webdav' | 'azure' | 'dropbox' | 'gdrive' | 'sftp' | 'saved'>('local');
  let savedBackends = $state<SavedBackend[]>([]);
  let selectedSavedBackendId = $state('');

  // Local fields
  let localPath = $state('./backups');
  let localUseEnvVars = $state(false);

  // SMB fields
  let smbHost = $state('');
  let smbShare = $state('');
  let smbPath = $state('');
  let smbUsername = $state('');
  let smbPassword = $state('');
  let smbPort = $state(445);
  let smbUseEnvVars = $state(false);

  // Use env vars settings for other backends
  let s3UseEnvVars = $state(false);
  let webdavUseEnvVars = $state(false);
  let azureUseEnvVars = $state(false);
  let dropboxUseEnvVars = $state(false);
  let gdriveUseEnvVars = $state(false);
  let sftpUseEnvVars = $state(false);

  // S3 fields
  let s3Bucket = $state('');
  let s3AccessKey = $state('');
  let s3SecretKey = $state('');
  let s3Endpoint = $state('');
  let s3Region = $state('');
  let s3StorageClass = $state('');

  // WebDAV fields
  let webdavUrl = $state('');
  let webdavUser = $state('');
  let webdavPass = $state('');
  let webdavPath = $state('');
  let webdavInsecure = $state(false);

  // Azure fields
  let azureConnString = $state('');
  let azureContainer = $state('');

  // Dropbox fields
  let dropboxToken = $state('');
  let dropboxAppKey = $state('');
  let dropboxAppSecret = $state('');
  let dropboxPath = $state('');

  // Google Drive fields
  let gdriveFolderId = $state('');
  let gdriveCredentials = $state('');
  let gdriveImpersonate = $state('');

  // SSH/SFTP fields
  let sftpHost = $state('');
  let sftpPort = $state(22);
  let sftpUser = $state('');
  let sftpPass = $state('');
  let sftpKey = $state('');
  let sftpPath = $state('');

  // Step 6: Output
  let generatedCompose = $state('');
  let copySuccess = $state(false);

  // Derived lists
  let gitSavedSources = $derived(savedSources.filter(s => s.type === 'git'));
  let portainerSavedSources = $derived(savedSources.filter(s => s.type === 'portainer'));

  onMount(async () => {
    try {
      savedBackends = await listSavedBackends();
      savedSources = await listSavedSources();
      projectsList = await listProjects();
    } catch (e) {
      console.error('Failed to load saved configuration presets:', e);
    }
  });

  async function handleGitSourceSelect(id: string) {
    selectedSavedSourceId = id;
    if (id === '') return;
    try {
      const source = await getSavedSource(id);
      if (source?.git_config) {
        gitRepoUrl = source.git_config.repo_url;
        gitBranch = source.git_config.branch || 'main';
        gitFilePath = source.git_config.file_path || 'docker-compose.yml';
        gitAuthToken = source.git_config.auth_token || '';
        gitSshKey = source.git_config.ssh_private_key || '';
      }
    } catch {
      const source = savedSources.find(s => s.id === id);
      if (source?.git_config) {
        gitRepoUrl = source.git_config.repo_url;
        gitBranch = source.git_config.branch || 'main';
        gitFilePath = source.git_config.file_path || 'docker-compose.yml';
      }
    }
  }

  async function handlePortainerSourceSelect(id: string) {
    selectedSavedSourceId = id;
    if (id === '') return;
    try {
      const source = await getSavedSource(id);
      if (source?.portainer_config) {
        portainerUrl = source.portainer_config.portainer_url;
        portainerApiKey = source.portainer_config.api_key || '';
        if (portainerApiKey) {
          await connectPortainer();
        }
      }
    } catch {
      const source = savedSources.find(s => s.id === id);
      if (source?.portainer_config) {
        portainerUrl = source.portainer_config.portainer_url;
      }
    }
  }

  async function connectPortainer() {
    error = '';
    if (!portainerUrl.trim()) {
      error = 'Portainer URL is required';
      return;
    }
    if (!selectedSavedSourceId && !portainerApiKey.trim()) {
      error = 'API key is required';
      return;
    }

    loading = true;
    try {
      const result = await portainerConnect(portainerUrl.trim(), portainerApiKey.trim(), selectedSavedSourceId || undefined);
      endpointsList = result.endpoints;
      portainerConnected = true;
      if (endpointsList.length === 1) {
        selectedPortainerEndpointId = endpointsList[0].Id;
        await loadPortainerStacks();
      }
    } catch (e) {
      error = e instanceof Error ? e.message : 'Connection to Portainer failed';
    } finally {
      loading = false;
    }
  }

  async function loadPortainerStacks() {
    if (selectedPortainerEndpointId === null) return;
    stacksLoading = true;
    error = '';
    try {
      stacksList = await apiPortainerStacks(portainerUrl.trim(), portainerApiKey.trim(), selectedPortainerEndpointId, selectedSavedSourceId || undefined);
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to list stacks';
    } finally {
      stacksLoading = false;
    }
  }

  const finalCron = $derived.by(() => {
    switch (cronPreset) {
      case 'hourly': return '0 * * * *';
      case 'weekly': return '0 3 * * 0';
      case 'monthly': return '0 3 1 * *';
      case 'custom': return cronCustom;
      case 'daily':
      default:
        return '0 2 * * *';
    }
  });

  async function handleStep1Next() {
    error = '';
    if (sourceType === 'project' && !selectedProjectId) {
      error = 'Please select an existing project';
      return;
    }
    if (!projectName.trim() && sourceType !== 'portainer' && sourceType !== 'project') {
      error = 'Project name is required';
      return;
    }

    loading = true;
    try {
      let content = composeContent;

      if (sourceType === 'project') {
        const proj = projectsList.find(p => p.id === selectedProjectId);
        if (!proj) {
          error = 'Selected project not found';
          loading = false;
          return;
        }
        content = proj.compose_content;
        composeContent = content;
        projectName = proj.name;
      } else if (sourceType === 'git') {
        if (!gitRepoUrl.trim()) {
          error = 'Git repository URL is required';
          loading = false;
          return;
        }
        const res = await fetchBCGit({
          repo_url: gitRepoUrl.trim(),
          branch: gitBranch,
          file_path: gitFilePath,
          auth_token: gitAuthToken,
          ssh_private_key: gitSshKey,
          saved_source_id: selectedSavedSourceId || undefined
        });
        content = res.compose_content;
        composeContent = content;
      } else if (sourceType === 'portainer') {
        if (!selectedPortainerStackId) {
          error = 'Please select a Portainer stack';
          loading = false;
          return;
        }
        const res = await fetchBCPortainer({
          portainer_url: portainerUrl.trim(),
          api_key: portainerApiKey.trim(),
          stack_id: selectedPortainerStackId,
          saved_source_id: selectedSavedSourceId || undefined
        });
        content = res.compose_content;
        composeContent = content;

        // Auto-assign project name from stack name if not set
        if (!projectName.trim()) {
          const selectedStack = stacksList.find(s => s.Id === selectedPortainerStackId);
          if (selectedStack) {
            projectName = selectedStack.Name;
          }
        }
      }

      if (!content.trim()) {
        error = 'Compose content is empty';
        loading = false;
        return;
      }

      const parseResult = await parseBCCompose(content);
      parsedVolumes = parseResult.volumes;
      parsedServices = parseResult.services;

      if (!projectName.trim() && parseResult.project_name) {
        projectName = parseResult.project_name;
      }

      // Inferred settings configuration:
      
      // 1. Pre-select only volumes that already have backups set up
      const volumesWithPatterns = parsedVolumes.filter(v => v.backup_pattern);
      if (volumesWithPatterns.length > 0) {
        selectedVolumes = volumesWithPatterns.map(v => v.name);
      } else {
        selectedVolumes = parsedVolumes.map(v => v.name);
      }

      // 2. Pre-populate filename formats
      volumeFilenameFormats = {};
      for (const vol of parsedVolumes) {
        if (vol.backup_pattern) {
          volumeFilenameFormats[vol.name] = vol.backup_pattern;
        } else {
          const suffix = vol.name.toLowerCase().replace(/[^a-z0-9_-]/g, '_');
          volumeFilenameFormats[vol.name] = `backup-${suffix}-%Y-%m-%dT%H-%M-%S.tar.gz`;
        }
      }

      // 3. Cron expression
      if (parseResult.cron_expression) {
        const expr = parseResult.cron_expression;
        if (expr === '0 * * * *') {
          cronPreset = 'hourly';
        } else if (expr === '0 3 * * 0') {
          cronPreset = 'weekly';
        } else if (expr === '0 3 1 * *') {
          cronPreset = 'monthly';
        } else if (expr === '0 2 * * *') {
          cronPreset = 'daily';
        } else {
          cronPreset = 'custom';
          cronCustom = expr;
        }
      }

      // 4. GPG Passphrase
      if (parseResult.gpg_passphrase) {
        gpgPassphrase = parseResult.gpg_passphrase;
        enableGpg = true;
      } else {
        gpgPassphrase = '';
        enableGpg = false;
      }

      // 5. Retention Days
      if (parseResult.retention_days !== undefined) {
        retentionDays = parseResult.retention_days;
      } else {
        retentionDays = 0;
      }

      // 6. Backup Image
      if (parseResult.backup_image) {
        backupImage = parseResult.backup_image;
      }

      // 7. Stop Services
      if (parseResult.stop_services) {
        stopServices = parseResult.stop_services;
      } else {
        stopServices = [];
      }

      // 8. Storage backend and custom variables
      if (parseResult.smb_config) {
        backendType = 'smb';
        smbHost = parseResult.smb_config.host;
        smbShare = parseResult.smb_config.share;
        smbPath = parseResult.smb_config.path || '';
        smbPort = parseResult.smb_config.port || 445;
        smbUsername = parseResult.smb_config.username || '';
        smbPassword = parseResult.smb_config.password || '';
        smbUseEnvVars = parseResult.smb_config.use_env_vars || false;
      } else if (parseResult.backup_volumes && parseResult.backup_volumes.length > 0) {
        const firstVol = parseResult.backup_volumes[0];
        const [hostPath, containerPath] = firstVol.split(':');
        if (containerPath === '/archive' && !hostPath.includes('smb_backup')) {
          backendType = 'local';
          localPath = hostPath;
        }
      }

      // 9. Environment variables / credentials mapping
      if (parseResult.env_vars) {
        const ev = parseResult.env_vars;
        if (ev.AWS_S3_BUCKET_NAME) {
          backendType = 's3';
          s3Bucket = ev.AWS_S3_BUCKET_NAME;
          s3AccessKey = ev.AWS_ACCESS_KEY_ID || '';
          s3SecretKey = ev.AWS_SECRET_ACCESS_KEY || '';
          s3Endpoint = ev.AWS_ENDPOINT || '';
          s3Region = ev.AWS_DEFAULT_REGION || '';
          s3StorageClass = ev.AWS_STORAGE_CLASS || '';
        } else if (ev.WEBDAV_URL) {
          backendType = 'webdav';
          webdavUrl = ev.WEBDAV_URL;
          webdavUser = ev.WEBDAV_USERNAME || '';
          webdavPass = ev.WEBDAV_PASSWORD || '';
          webdavPath = ev.WEBDAV_PATH || '';
          webdavInsecure = ev.WEBDAV_URL_INSECURE === 'true';
        } else if (ev.AZURE_STORAGE_CONNECTION_STRING) {
          backendType = 'azure';
          azureConnString = ev.AZURE_STORAGE_CONNECTION_STRING;
          azureContainer = ev.AZURE_STORAGE_CONTAINER || '';
        } else if (ev.DROPBOX_ACCESS_TOKEN) {
          backendType = 'dropbox';
          dropboxToken = ev.DROPBOX_ACCESS_TOKEN;
          dropboxAppKey = ev.DROPBOX_APP_KEY || '';
          dropboxAppSecret = ev.DROPBOX_APP_SECRET || '';
          dropboxPath = ev.DROPBOX_REMOTE_PATH || '';
        } else if (ev.GOOGLE_DRIVE_FOLDER_ID) {
          backendType = 'gdrive';
          gdriveFolderId = ev.GOOGLE_DRIVE_FOLDER_ID;
          gdriveCredentials = ev.GOOGLE_DRIVE_CREDENTIALS || '';
          gdriveImpersonate = ev.GOOGLE_DRIVE_IMPERSONATE_USER || '';
        } else if (ev.SSH_HOST) {
          backendType = 'sftp';
          sftpHost = ev.SSH_HOST;
          sftpPort = Number(ev.SSH_PORT) || 22;
          sftpUser = ev.SSH_USER || '';
          sftpPass = ev.SSH_PASSWORD || '';
          sftpKey = ev.SSH_KEY || '';
          sftpPath = ev.SSH_REMOTE_PATH || '';
        }
      }

      step = 2;
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to retrieve or parse compose file';
    } finally {
      loading = false;
    }
  }

  function handleStep2Next() {
    error = '';
    if (selectedVolumes.length === 0) {
      error = 'Please select at least one volume to back up';
      return;
    }
    for (const vol of selectedVolumes) {
      if (!volumeFilenameFormats[vol]) {
        const suffix = vol.toLowerCase().replace(/[^a-z0-9_-]/g, '_');
        volumeFilenameFormats[vol] = `backup-${suffix}-%Y-%m-%dT%H-%M-%S.tar.gz`;
      }
    }
    step = 3;
  }

  function handleStep3Next() {
    step = 4;
  }

  function handleStep4Next() {
    error = '';
    if (enableGpg && !gpgPassphrase) {
      error = 'GPG passphrase is required when GPG encryption is enabled';
      return;
    }
    step = 5;
  }

  async function handleStep5Generate() {
    error = '';
    loading = true;

    try {
      const envVars: Record<string, string> = {};
      const extraVolumes: string[] = [];
      let smbConfig: any = undefined;

      if (backendType === 'local') {
        if (!localUseEnvVars && !localPath.trim()) {
          error = 'Backup directory path is required';
          loading = false;
          return;
        }
        const finalPath = localUseEnvVars ? '${LOCAL_BACKUP_PATH}' : localPath.trim();
        extraVolumes.push(`${finalPath}:/archive`);
      } else if (backendType === 'smb') {
        if (!smbUseEnvVars && (!smbHost.trim() || !smbShare.trim())) {
          error = 'SMB Host and Share are required';
          loading = false;
          return;
        }
        extraVolumes.push('smb_backup:/archive');
        smbConfig = {
          host: smbHost.trim() || "SMB_HOST",
          share: smbShare.trim() || "SMB_SHARE",
          path: smbPath.trim() || (smbUseEnvVars ? "SMB_PATH" : undefined),
          username: smbUsername.trim() || (smbUseEnvVars ? "SMB_USERNAME" : undefined),
          password: smbPassword || (smbUseEnvVars ? "SMB_PASSWORD" : undefined),
          port: smbPort || (smbUseEnvVars ? 445 : undefined),
          use_env_vars: smbUseEnvVars
        };
      } else if (backendType === 's3') {
        if (!s3UseEnvVars && (!s3Bucket.trim() || !s3AccessKey.trim() || !s3SecretKey.trim())) {
          error = 'S3 Bucket Name, Access Key ID, and Secret Access Key are required';
          loading = false;
          return;
        }
        if (s3UseEnvVars) {
          envVars['AWS_S3_BUCKET_NAME'] = '${AWS_S3_BUCKET_NAME}';
          envVars['AWS_ACCESS_KEY_ID'] = '${AWS_ACCESS_KEY_ID}';
          envVars['AWS_SECRET_ACCESS_KEY'] = '${AWS_SECRET_ACCESS_KEY}';
          if (s3Endpoint.trim() || s3UseEnvVars) envVars['AWS_ENDPOINT'] = '${AWS_ENDPOINT}';
          if (s3Region.trim() || s3UseEnvVars) envVars['AWS_DEFAULT_REGION'] = '${AWS_DEFAULT_REGION}';
          if (s3StorageClass.trim() || s3UseEnvVars) envVars['AWS_STORAGE_CLASS'] = '${AWS_STORAGE_CLASS}';
        } else {
          envVars['AWS_S3_BUCKET_NAME'] = s3Bucket.trim();
          envVars['AWS_ACCESS_KEY_ID'] = s3AccessKey.trim();
          envVars['AWS_SECRET_ACCESS_KEY'] = s3SecretKey.trim();
          if (s3Endpoint.trim()) envVars['AWS_ENDPOINT'] = s3Endpoint.trim();
          if (s3Region.trim()) envVars['AWS_DEFAULT_REGION'] = s3Region.trim();
          if (s3StorageClass.trim()) envVars['AWS_STORAGE_CLASS'] = s3StorageClass.trim();
        }
      } else if (backendType === 'webdav') {
        if (!webdavUseEnvVars && (!webdavUrl.trim() || !webdavUser.trim() || !webdavPass)) {
          error = 'WebDAV URL, Username, and Password are required';
          loading = false;
          return;
        }
        if (webdavUseEnvVars) {
          envVars['WEBDAV_URL'] = '${WEBDAV_URL}';
          envVars['WEBDAV_USERNAME'] = '${WEBDAV_USERNAME}';
          envVars['WEBDAV_PASSWORD'] = '${WEBDAV_PASSWORD}';
          if (webdavPath.trim() || webdavUseEnvVars) envVars['WEBDAV_PATH'] = '${WEBDAV_PATH}';
          if (webdavInsecure || webdavUseEnvVars) envVars['WEBDAV_URL_INSECURE'] = '${WEBDAV_URL_INSECURE}';
        } else {
          envVars['WEBDAV_URL'] = webdavUrl.trim();
          envVars['WEBDAV_USERNAME'] = webdavUser.trim();
          envVars['WEBDAV_PASSWORD'] = webdavPass;
          if (webdavPath.trim()) envVars['WEBDAV_PATH'] = webdavPath.trim();
          if (webdavInsecure) envVars['WEBDAV_URL_INSECURE'] = 'true';
        }
      } else if (backendType === 'azure') {
        if (!azureUseEnvVars && (!azureConnString.trim() || !azureContainer.trim())) {
          error = 'Azure Storage Connection String and Container name are required';
          loading = false;
          return;
        }
        if (azureUseEnvVars) {
          envVars['AZURE_STORAGE_CONNECTION_STRING'] = '${AZURE_STORAGE_CONNECTION_STRING}';
          envVars['AZURE_STORAGE_CONTAINER'] = '${AZURE_STORAGE_CONTAINER}';
        } else {
          envVars['AZURE_STORAGE_CONNECTION_STRING'] = azureConnString.trim();
          envVars['AZURE_STORAGE_CONTAINER'] = azureContainer.trim();
        }
      } else if (backendType === 'dropbox') {
        if (!dropboxUseEnvVars && !dropboxToken.trim()) {
          error = 'Dropbox Access Token is required';
          loading = false;
          return;
        }
        if (dropboxUseEnvVars) {
          envVars['DROPBOX_ACCESS_TOKEN'] = '${DROPBOX_ACCESS_TOKEN}';
          if (dropboxAppKey.trim() || dropboxUseEnvVars) envVars['DROPBOX_APP_KEY'] = '${DROPBOX_APP_KEY}';
          if (dropboxAppSecret.trim() || dropboxUseEnvVars) envVars['DROPBOX_APP_SECRET'] = '${DROPBOX_APP_SECRET}';
          if (dropboxPath.trim() || dropboxUseEnvVars) envVars['DROPBOX_REMOTE_PATH'] = '${DROPBOX_REMOTE_PATH}';
        } else {
          envVars['DROPBOX_ACCESS_TOKEN'] = dropboxToken.trim();
          if (dropboxAppKey.trim()) envVars['DROPBOX_APP_KEY'] = dropboxAppKey.trim();
          if (dropboxAppSecret.trim()) envVars['DROPBOX_APP_SECRET'] = dropboxAppSecret.trim();
          if (dropboxPath.trim()) envVars['DROPBOX_REMOTE_PATH'] = dropboxPath.trim();
        }
      } else if (backendType === 'gdrive') {
        if (!gdriveUseEnvVars && (!gdriveFolderId.trim() || !gdriveCredentials.trim())) {
          error = 'Google Drive Folder ID and Service Account Credentials are required';
          loading = false;
          return;
        }
        if (gdriveUseEnvVars) {
          envVars['GOOGLE_DRIVE_FOLDER_ID'] = '${GOOGLE_DRIVE_FOLDER_ID}';
          envVars['GOOGLE_DRIVE_CREDENTIALS'] = '${GOOGLE_DRIVE_CREDENTIALS}';
          if (gdriveImpersonate.trim() || gdriveUseEnvVars) envVars['GOOGLE_DRIVE_IMPERSONATE_USER'] = '${GOOGLE_DRIVE_IMPERSONATE_USER}';
        } else {
          envVars['GOOGLE_DRIVE_FOLDER_ID'] = gdriveFolderId.trim();
          envVars['GOOGLE_DRIVE_CREDENTIALS'] = gdriveCredentials.trim();
          if (gdriveImpersonate.trim()) envVars['GOOGLE_DRIVE_IMPERSONATE_USER'] = gdriveImpersonate.trim();
        }
      } else if (backendType === 'sftp') {
        if (!sftpUseEnvVars && (!sftpHost.trim() || !sftpUser.trim())) {
          error = 'SSH/SFTP Host and User are required';
          loading = false;
          return;
        }
        if (sftpUseEnvVars) {
          envVars['SSH_HOST'] = '${SSH_HOST}';
          envVars['SSH_USER'] = '${SSH_USER}';
          if (sftpPort || sftpUseEnvVars) envVars['SSH_PORT'] = '${SSH_PORT}';
          if (sftpPass || sftpUseEnvVars) envVars['SSH_PASSWORD'] = '${SSH_PASSWORD}';
          if (sftpKey.trim() || sftpUseEnvVars) envVars['SSH_KEY'] = '${SSH_KEY}';
          if (sftpPath.trim() || sftpUseEnvVars) envVars['SSH_REMOTE_PATH'] = '${SSH_REMOTE_PATH}';
        } else {
          envVars['SSH_HOST'] = sftpHost.trim();
          envVars['SSH_USER'] = sftpUser.trim();
          if (sftpPort && sftpPort !== 22) envVars['SSH_PORT'] = String(sftpPort);
          if (sftpPass) envVars['SSH_PASSWORD'] = sftpPass;
          if (sftpKey.trim()) envVars['SSH_KEY'] = sftpKey.trim();
          if (sftpPath.trim()) envVars['SSH_REMOTE_PATH'] = sftpPath.trim();
        }
      } else if (backendType === 'saved') {
        const backend = savedBackends.find(b => b.id === selectedSavedBackendId);
        if (!backend) {
          error = 'Please select a saved backend';
          loading = false;
          return;
        }
        if (backend.credentials.type === 'local' && backend.credentials.local) {
          const finalPath = localUseEnvVars ? '${LOCAL_BACKUP_PATH}' : backend.credentials.local.path;
          extraVolumes.push(`${finalPath}:/archive`);
        } else if (backend.credentials.type === 'smb' && backend.credentials.smb) {
          extraVolumes.push('smb_backup:/archive');
          smbConfig = {
            host: backend.credentials.smb.host,
            share: backend.credentials.smb.share,
            path: backend.credentials.smb.path || undefined,
            username: backend.credentials.smb.username || undefined,
            password: backend.credentials.smb.password || undefined,
            port: backend.credentials.smb.port || undefined,
            use_env_vars: smbUseEnvVars
          };
        }
      }

      let currentCompose = composeContent;

      for (const vol of selectedVolumes) {
        const pv = parsedVolumes.find(v => v.name === vol);
        const suffix = vol.toLowerCase().replace(/[^a-z0-9_-]/g, '_');
        const serviceName = `${backupServiceName.trim()}_${suffix}`;

        const currentFormat = (volumeFilenameFormats[vol] || `backup-${suffix}-%Y-%m-%dT%H-%M-%S.tar.gz`).trim();

        const volStopServices: string[] = [];
        if (pv && stopServices.includes(pv.compose_service)) {
          volStopServices.push(pv.compose_service);
        }

        const res = await generateBCCompose({
          compose_content: currentCompose,
          service_name: serviceName,
          image: backupImage.trim(),
          cron_expression: finalCron,
          filename_format: currentFormat,
          gpg_passphrase: enableGpg ? gpgPassphrase : undefined,
          retention_days: retentionDays > 0 ? retentionDays : undefined,
          selected_volumes: [vol],
          stop_services: volStopServices,
          env_vars: envVars,
          volumes: extraVolumes,
          smb_config: smbConfig
        });

        currentCompose = res.compose_content;
      }

      generatedCompose = currentCompose;
      step = 6;
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to generate compose file';
    } finally {
      loading = false;
    }
  }

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(generatedCompose);
      copySuccess = true;
      setTimeout(() => (copySuccess = false), 2000);
    } catch {
      // ignore
    }
  }

  function handleDownload() {
    const blob = new Blob([generatedCompose], { type: 'text/yaml' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'docker-compose.yml';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  }

  async function handleSaveProject() {
    error = '';
    loading = true;
    try {
      let projectId = '';
      if (sourceType === 'paste') {
        const importRes = await importCompose(generatedCompose, 'standalone');
        projectId = importRes.id;
      } else if (sourceType === 'git') {
        const importRes = await importGit({
          repo_url: gitRepoUrl.trim(),
          branch: gitBranch,
          file_path: gitFilePath,
          auth_token: gitAuthToken,
          ssh_private_key: gitSshKey,
          project_name: projectName.trim(),
          deployment_mode: 'standalone',
          saved_source_id: selectedSavedSourceId || undefined
        });
        projectId = importRes.id;
        await updateCompose(projectId, generatedCompose);
      } else if (sourceType === 'portainer') {
        if (!selectedPortainerStackId || !selectedPortainerEndpointId) {
          error = 'Endpoint and Stack selection required';
          loading = false;
          return;
        }
        const importRes = await importPortainer({
          portainer_url: portainerUrl.trim(),
          api_key: portainerApiKey.trim(),
          stack_id: selectedPortainerStackId,
          endpoint_id: selectedPortainerEndpointId,
          project_name: projectName.trim(),
          deployment_mode: 'standalone',
          saved_source_id: selectedSavedSourceId || undefined
        });
        projectId = importRes.id;
        await updateCompose(projectId, generatedCompose);
      }

      // Save credentials if local or smb backend is configured
      let credsToSave: Credentials | null = null;
      const isPlaceholder = (val: string) => !val || val.includes('$') || val.includes('{');

      if (backendType === 'local' && !isPlaceholder(localPath.trim())) {
        credsToSave = { type: 'local', local: { path: localPath.trim() } };
      } else if (backendType === 'smb') {
        if (!isPlaceholder(smbHost.trim()) && !isPlaceholder(smbShare.trim())) {
          credsToSave = {
            type: 'smb',
            smb: {
              host: smbHost.trim(),
              share: smbShare.trim(),
              path: smbPath.trim(),
              username: smbUsername.trim(),
              password: smbPassword,
              port: smbPort
            }
          };
        }
      } else if (backendType === 'saved') {
        const backend = savedBackends.find(b => b.id === selectedSavedBackendId);
        if (backend) {
          credsToSave = {
            ...backend.credentials,
            saved_backend_id: selectedSavedBackendId
          };
        }
      }

      if (credsToSave) {
        try {
          await saveCredentials(projectId, credsToSave);
        } catch (err) {
          console.warn('Could not save credentials to database:', err);
        }
      }

      await createProject(projectId, projectName.trim());
      await projectState.load();
      goto(`/projects/${projectId}`);
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to save project';
    } finally {
      loading = false;
    }
  }

  function handleBack() {
    error = '';
    step = Math.max(1, step - 1);
  }
</script>

<div class="min-h-screen bg-slate-950 flex flex-col items-center justify-start pt-12 pb-24 px-4 animate-fade-in">
  <div class="w-full max-w-2xl">
    <div class="flex items-center justify-between mb-2">
      <h1 class="text-2xl font-light text-slate-100">Backup Creator Wizard</h1>
      <a href="/" class="text-sm text-slate-500 hover:text-slate-200 transition-colors">
        ← Back to Projects
      </a>
    </div>
    <p class="text-sm text-slate-500 mb-8">
      Generate a customized Docker Compose configuration with integrated Offen backup sidecars.
    </p>

    <!-- Step Indicator -->
    <div class="flex items-center justify-between w-full mb-8 bg-slate-800/50 border border-slate-700/50 rounded-xl px-6 py-4 overflow-x-auto gap-4">
      {#each [1, 2, 3, 4, 5, 6] as i}
        <div class="flex items-center gap-2 shrink-0">
          <div
            class="w-8 h-8 rounded-full flex items-center justify-center text-sm font-semibold transition-all duration-500 {step >= i ? 'bg-gradient-to-r from-emerald-500 to-emerald-400 text-white shadow-lg shadow-emerald-500/25' : 'bg-slate-700 text-slate-500'}"
          >
            {i}
          </div>
          <span class="text-xs font-medium {step === i ? 'text-slate-100 font-semibold' : 'text-slate-500'}">
            {#if i === 1}Source{:else if i === 2}Volumes{:else if i === 3}Stopping{:else if i === 4}Options{:else if i === 5}Destination{:else}Output{/if}
          </span>
          {#if i < 6}
            <div class="w-6 h-0.5 transition-all duration-500 {step > i ? 'bg-emerald-500' : 'bg-slate-700'}"></div>
          {/if}
        </div>
      {/each}
    </div>

    <!-- Error Banner -->
    {#if error}
      <div class="mb-6 p-4 bg-red-500/10 border border-red-500/30 rounded-xl text-red-400 text-sm flex items-start gap-3">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 shrink-0 mt-0.5 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
        </svg>
        <span>{error}</span>
      </div>
    {/if}

    <!-- Main Card -->
    <div class="bg-slate-800/50 rounded-2xl border border-slate-700/50 shadow-xl shadow-black/20 p-8 transition-all">

      <!-- STEP 1: STARTING POINT -->
      {#if step === 1}
        <div class="space-y-6">
          <div>
            <h2 class="text-lg font-semibold text-slate-100 mb-1">Step 1: Choose Import Source</h2>
            <p class="text-sm text-slate-400">Pasted compose content, remote Git repo, or Portainer stacks.</p>
          </div>

          <div>
            <label class="block text-sm font-medium text-slate-300 mb-2">Import Method</label>
            <div class="grid grid-cols-2 md:flex gap-2">
              <button
                onclick={() => { sourceType = 'paste'; selectedSavedSourceId = ''; }}
                class="flex-1 py-3 px-2 border-2 rounded-xl text-xs font-semibold transition-all {sourceType === 'paste' ? 'border-emerald-500 bg-emerald-500/10 text-emerald-400' : 'border-slate-600 text-slate-400 hover:border-slate-500'}"
              >
                📝 Paste Compose
              </button>
              <button
                onclick={() => { sourceType = 'git'; selectedSavedSourceId = ''; }}
                class="flex-1 py-3 px-2 border-2 rounded-xl text-xs font-semibold transition-all {sourceType === 'git' ? 'border-emerald-500 bg-emerald-500/10 text-emerald-400' : 'border-slate-600 text-slate-400 hover:border-slate-500'}"
              >
                🐙 Git Repository
              </button>
              <button
                onclick={() => { sourceType = 'portainer'; selectedSavedSourceId = ''; }}
                class="flex-1 py-3 px-2 border-2 rounded-xl text-xs font-semibold transition-all {sourceType === 'portainer' ? 'border-emerald-500 bg-emerald-500/10 text-emerald-400' : 'border-slate-600 text-slate-400 hover:border-slate-500'}"
              >
                🐳 Portainer Stacks
              </button>
              <button
                onclick={() => { sourceType = 'project'; selectedSavedSourceId = ''; }}
                class="flex-1 py-3 px-2 border-2 rounded-xl text-xs font-semibold transition-all {sourceType === 'project' ? 'border-emerald-500 bg-emerald-500/10 text-emerald-400' : 'border-slate-600 text-slate-400 hover:border-slate-500'}"
              >
                📁 Existing Project
              </button>
            </div>
          </div>

          {#if sourceType !== 'portainer' && sourceType !== 'project'}
            <div>
              <label for="p-name" class="block text-sm font-medium text-slate-300 mb-2">Project Name</label>
              <input
                id="p-name"
                type="text"
                bind:value={projectName}
                oninput={() => (userEditedProjectName = true)}
                placeholder="e.g. database-stack, dev-env"
                class="w-full px-4 py-2.5 bg-slate-900 border border-slate-600 text-slate-100 rounded-xl focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500 placeholder:text-slate-500"
              />
            </div>
          {/if}

          <!-- EXISTING PROJECT METHOD FORM -->
          {#if sourceType === 'project'}
            <div class="space-y-4">
              <div>
                <label for="project-select" class="block text-sm font-medium text-slate-300 mb-2">Select Project</label>
                <select
                  id="project-select"
                  bind:value={selectedProjectId}
                  class="w-full px-4 py-2.5 border border-slate-600 rounded-xl focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 bg-slate-900"
                >
                  <option value="">Choose an existing project...</option>
                  {#each projectsList as proj}
                    <option value={proj.id}>{proj.name} ({proj.deployment_mode === 'git' ? 'Git' : proj.deployment_mode === 'portainer' ? 'Portainer' : 'Standalone'})</option>
                  {/each}
                </select>
              </div>
            </div>

          <!-- GIT METHOD FORM -->
          {:else if sourceType === 'git'}
            <div class="space-y-4">
              {#if gitSavedSources.length > 0}
                <div>
                  <label for="saved-git-source" class="block text-sm font-medium text-slate-300 mb-2">Use saved Git source</label>
                  <select
                    id="saved-git-source"
                    value={selectedSavedSourceId}
                    onchange={(e) => handleGitSourceSelect(e.currentTarget.value)}
                    class="w-full px-4 py-2.5 border border-slate-600 rounded-xl focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 bg-slate-900"
                  >
                    <option value="">Choose saved source...</option>
                    {#each gitSavedSources as src}
                      <option value={src.id}>{src.name}</option>
                    {/each}
                  </select>
                </div>
              {/if}
              <div class="grid grid-cols-2 gap-4">
                <div class="col-span-2">
                  <label for="git-url" class="block text-sm font-medium text-slate-300 mb-2">Repo URL</label>
                  <input
                    id="git-url"
                    type="text"
                    bind:value={gitRepoUrl}
                    placeholder="https://github.com/user/repo.git"
                    class="w-full px-4 py-2.5 bg-slate-900 border border-slate-600 text-slate-100 rounded-xl focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500 placeholder:text-slate-500"
                  />
                </div>
                <div>
                  <label for="git-branch" class="block text-sm font-medium text-slate-300 mb-2">Branch</label>
                  <input
                    id="git-branch"
                    type="text"
                    bind:value={gitBranch}
                    placeholder="main"
                    class="w-full px-4 py-2.5 bg-slate-900 border border-slate-600 text-slate-100 rounded-xl focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500 placeholder:text-slate-500"
                  />
                </div>
                <div>
                  <label for="git-path" class="block text-sm font-medium text-slate-300 mb-2">Compose Path</label>
                  <input
                    id="git-path"
                    type="text"
                    bind:value={gitFilePath}
                    placeholder="docker-compose.yml"
                    class="w-full px-4 py-2.5 bg-slate-900 border border-slate-600 text-slate-100 rounded-xl focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500 placeholder:text-slate-500"
                  />
                </div>
                {#if !selectedSavedSourceId}
                  <div class="col-span-2">
                    <label for="git-token" class="block text-sm font-medium text-slate-300 mb-2">Auth Token (HTTPS, Optional)</label>
                    <input
                      id="git-token"
                      type="password"
                      bind:value={gitAuthToken}
                      placeholder="ghp_xxxxxxxxxxxxxxxxxxxxx"
                      class="w-full px-4 py-2.5 bg-slate-900 border border-slate-600 text-slate-100 rounded-xl focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500 placeholder:text-slate-500"
                    />
                  </div>
                  <div class="col-span-2">
                    <label for="git-ssh" class="block text-sm font-medium text-slate-300 mb-2">SSH Private Key (Optional)</label>
                    <textarea
                      id="git-ssh"
                      bind:value={gitSshKey}
                      rows="3"
                      placeholder="-----BEGIN OPENSSH PRIVATE KEY-----..."
                      class="w-full px-4 py-3 border border-slate-600 rounded-xl font-mono text-xs focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500"
                    ></textarea>
                  </div>
                {/if}
              </div>
            </div>

          <!-- PORTAINER METHOD FORM -->
          {:else if sourceType === 'portainer'}
            <div class="space-y-4">
              {#if portainerSavedSources.length > 0}
                <div>
                  <label for="saved-port-source" class="block text-sm font-medium text-slate-300 mb-2">Use saved Portainer source</label>
                  <select
                    id="saved-port-source"
                    value={selectedSavedSourceId}
                    onchange={(e) => handlePortainerSourceSelect(e.currentTarget.value)}
                    disabled={portainerConnected}
                    class="w-full px-4 py-2.5 border border-slate-600 rounded-xl focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 bg-slate-900 disabled:bg-slate-800 disabled:text-slate-500"
                  >
                    <option value="">Choose saved source...</option>
                    {#each portainerSavedSources as src}
                      <option value={src.id}>{src.name}</option>
                    {/each}
                  </select>
                </div>
              {/if}

              <div>
                <label for="port-url" class="block text-sm font-medium text-slate-300 mb-2">Portainer URL</label>
                <input
                  id="port-url"
                  type="url"
                  bind:value={portainerUrl}
                  disabled={portainerConnected}
                  placeholder="https://portainer.local"
                  class="w-full px-4 py-2.5 border border-slate-600 rounded-xl focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 disabled:bg-slate-800 disabled:text-slate-500"
                />
              </div>
               {#if !selectedSavedSourceId}
                 <div>
                   <label for="port-key" class="block text-sm font-medium text-slate-300 mb-2">API Key</label>
                   <input
                     id="port-key"
                     type="password"
                     bind:value={portainerApiKey}
                     disabled={portainerConnected}
                     placeholder="ptr_..."
                     class="w-full px-4 py-2.5 border border-slate-600 rounded-xl focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 disabled:bg-slate-800 disabled:text-slate-500"
                   />
                 </div>
               {/if}

              {#if !portainerConnected}
                <button
                  onclick={connectPortainer}
                  disabled={loading}
                  class="w-full py-2 px-4 border border-emerald-600 text-emerald-400 hover:bg-emerald-500/10 disabled:opacity-50 rounded-xl font-semibold transition-colors"
                >
                  {loading ? 'Connecting...' : 'Connect Portainer'}
                </button>
              {:else}
                <div class="p-3 bg-green-50 border border-green-200 rounded-xl text-green-700 text-sm flex items-center gap-2">
                  <svg class="w-4 h-4 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                  </svg>
                  Connected to Portainer
                  <button
                    onclick={() => { portainerConnected = false; endpointsList = []; selectedPortainerEndpointId = null; stacksList = []; selectedPortainerStackId = null; }}
                    class="ml-auto text-emerald-400 underline text-xs font-semibold"
                  >
                    Disconnect
                  </button>
                </div>

                <!-- Endpoint Selection -->
                {#if endpointsList.length > 1}
                  <div>
                    <label for="port-ep" class="block text-sm font-medium text-slate-300 mb-2">Select Endpoint</label>
                    <select
                      id="port-ep"
                      bind:value={selectedPortainerEndpointId}
                      onchange={loadPortainerStacks}
                      class="w-full px-4 py-2.5 border border-slate-600 rounded-xl focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 bg-slate-900"
                    >
                      <option value={null}>Choose endpoint...</option>
                      {#each endpointsList as ep}
                        <option value={ep.Id}>{ep.Name} (ID: {ep.Id})</option>
                      {/each}
                    </select>
                  </div>
                {/if}

                <!-- Stack Selection -->
                {#if selectedPortainerEndpointId !== null}
                  <div>
                    <label for="port-stack" class="block text-sm font-medium text-slate-300 mb-2">Select Stack</label>
                    {#if stacksLoading}
                      <p class="text-sm text-slate-500">Loading stacks...</p>
                    {:else if stacksList.length === 0}
                      <p class="text-sm text-amber-700">No stacks found on this endpoint.</p>
                    {:else}
                      <select
                        id="port-stack"
                        bind:value={selectedPortainerStackId}
                        class="w-full px-4 py-2.5 border border-slate-600 rounded-xl focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 bg-slate-900"
                      >
                        <option value={null}>Choose stack...</option>
                        {#each stacksList as st}
                          <option value={st.Id}>{st.Name} {st.is_dvb_backed ? '(backed up)' : ''}</option>
                        {/each}
                      </select>
                    {/if}
                  </div>
                {/if}

                <!-- Project Name -->
                <div>
                  <label for="port-pname" class="block text-sm font-medium text-slate-300 mb-2">Project Name (Optional)</label>
                  <input
                    id="port-pname"
                    type="text"
                    bind:value={projectName}
                    placeholder="Defaults to stack name"
                    class="w-full px-4 py-2.5 bg-slate-900 border border-slate-600 text-slate-100 rounded-xl focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500 placeholder:text-slate-500"
                  />
                </div>
              {/if}
            </div>

          <!-- PASTE METHOD FORM -->
          {:else}
            <div>
              <label for="compose-paste" class="block text-sm font-medium text-slate-300 mb-2">docker-compose.yml</label>
              <textarea
                id="compose-paste"
                bind:value={composeContent}
                rows="10"
                placeholder="Paste your compose YAML content here..."
                class="w-full px-4 py-3 border border-slate-600 rounded-xl font-mono text-sm focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500"
              ></textarea>
            </div>
          {/if}

          <button
            onclick={handleStep1Next}
            disabled={loading || (sourceType === 'portainer' && !selectedPortainerStackId) || (sourceType === 'project' && !selectedProjectId)}
            class="w-full py-3 px-6 bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white rounded-xl font-semibold shadow-md transition-colors"
          >
            {loading ? 'Fetching & Parsing...' : 'Next'}
          </button>
        </div>

      <!-- STEP 2: VOLUME SELECTION -->
      {:else if step === 2}
        <div class="space-y-6">
          <div>
            <h2 class="text-lg font-semibold text-slate-100 mb-1">Step 2: Select Volumes to Back Up</h2>
            <p class="text-sm text-slate-400">Pick the named volumes from your compose file that need backing up.</p>
          </div>

          {#if parsedVolumes.length === 0}
            <div class="p-6 bg-amber-500/10 border border-amber-500/20 rounded-xl text-amber-400 text-sm">
              No top-level named volumes were detected in the compose file. You must define named volumes in your compose structure to back them up.
            </div>
          {:else}
            <div class="space-y-3 max-h-80 overflow-y-auto pr-2 border border-slate-700/50 p-2 rounded-xl">
              {#each parsedVolumes as vol}
                <label class="flex items-center gap-3 p-3 bg-slate-800/50 hover:bg-slate-700/50 rounded-xl cursor-pointer transition-colors border border-slate-700/50">
                  <input
                    type="checkbox"
                    value={vol.name}
                    bind:group={selectedVolumes}
                    class="h-5 w-5 rounded border-slate-600 text-emerald-600 focus:ring-emerald-500"
                  />
                  <div class="flex-1 min-w-0">
                    <div class="font-semibold text-slate-100 truncate">{vol.name}</div>
                    <div class="text-xs text-slate-500">Mounted on: <code class="bg-slate-700 px-1 rounded">{vol.compose_service}</code> at <code class="bg-slate-700 px-1 rounded">{vol.compose_mount_path}</code></div>
                  </div>
                </label>
              {/each}
            </div>
          {/if}

          <div class="flex gap-4">
            <button
              onclick={handleBack}
              class="flex-1 py-3 px-6 border border-slate-600 hover:bg-slate-800/50 text-slate-300 rounded-xl font-semibold transition-colors"
            >
              Back
            </button>
            <button
              onclick={handleStep2Next}
              class="flex-1 py-3 px-6 bg-emerald-600 hover:bg-emerald-500 text-white rounded-xl font-semibold shadow-md transition-colors"
            >
              Next
            </button>
          </div>
        </div>

      <!-- STEP 3: CONTAINER STOPPING -->
      {:else if step === 3}
        <div class="space-y-6">
          <div>
            <h2 class="text-lg font-semibold text-slate-100 mb-1">Step 3: Safe Backup Stopping (Optional)</h2>
            <p class="text-sm text-slate-400">Select services that should be scaled down/stopped during backups to avoid database corruption.</p>
          </div>

          {#if parsedServices.length === 0}
            <div class="p-6 bg-slate-800/50 border border-slate-700/50 rounded-xl text-slate-400 text-sm">
              No services detected.
            </div>
          {:else}
            <div class="space-y-3 max-h-80 overflow-y-auto pr-2 border border-slate-700/50 p-2 rounded-xl">
              {#each parsedServices as svc}
                {#if svc !== backupServiceName}
                  <label class="flex items-center gap-3 p-3 bg-slate-800/50 hover:bg-slate-700/50 rounded-xl cursor-pointer transition-colors border border-slate-700/50">
                    <input
                      type="checkbox"
                      value={svc}
                      bind:group={stopServices}
                      class="h-5 w-5 rounded border-slate-600 text-emerald-600 focus:ring-emerald-500"
                    />
                    <div class="flex-1 min-w-0">
                      <div class="font-semibold text-slate-100 truncate">{svc}</div>
                      <p class="text-xs text-slate-500">Will be stopped when backup begins, and restarted when complete.</p>
                    </div>
                  </label>
                {/if}
              {/each}
            </div>
          {/if}

          <div class="flex gap-4">
            <button
              onclick={handleBack}
              class="flex-1 py-3 px-6 border border-slate-600 hover:bg-slate-800/50 text-slate-300 rounded-xl font-semibold transition-colors"
            >
              Back
            </button>
            <button
              onclick={handleStep3Next}
              class="flex-1 py-3 px-6 bg-emerald-600 hover:bg-emerald-500 text-white rounded-xl font-semibold shadow-md transition-colors"
            >
              Next
            </button>
          </div>
        </div>

      <!-- STEP 4: BACKUP OPTIONS -->
      {:else if step === 4}
        <div class="space-y-6">
          <div>
            <h2 class="text-lg font-semibold text-slate-100 mb-1">Step 4: Backup Container Configurations</h2>
            <p class="text-sm text-slate-400">Configure naming convention, cron schedule, and encryption.</p>
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div>
              <label for="svc-name" class="block text-sm font-medium text-slate-300 mb-2">Service Name</label>
              <input
                id="svc-name"
                type="text"
                bind:value={backupServiceName}
                class="w-full px-4 py-2.5 bg-slate-900 border border-slate-600 text-slate-100 rounded-xl focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500 placeholder:text-slate-500"
              />
            </div>
            <div>
              <label for="svc-image" class="block text-sm font-medium text-slate-300 mb-2">Offen Backup Image</label>
              <input
                id="svc-image"
                type="text"
                bind:value={backupImage}
                class="w-full px-4 py-2.5 bg-slate-900 border border-slate-600 text-slate-100 rounded-xl focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500 placeholder:text-slate-500"
              />
            </div>
          </div>

          <div>
            <label class="block text-sm font-medium text-slate-300 mb-2">Backup Schedule (Frequency)</label>
            <div class="flex flex-wrap gap-2 mb-3">
              {#each ['daily', 'hourly', 'weekly', 'monthly', 'custom'] as p}
                <button
                  onclick={() => (cronPreset = p as any)}
                  class="py-1.5 px-3 border rounded-lg text-sm font-semibold transition-all {cronPreset === p ? 'border-emerald-600 bg-emerald-500/10 text-emerald-400' : 'border-slate-700/50 text-slate-400 hover:bg-slate-800/50'}"
                >
                  {p.charAt(0).toUpperCase() + p.slice(1)}
                </button>
              {/each}
            </div>
            {#if cronPreset === 'custom'}
              <input
                type="text"
                bind:value={cronCustom}
                placeholder="* * * * *"
                class="w-full px-4 py-2.5 border border-slate-600 rounded-xl font-mono text-sm focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500"
              />
              <p class="text-xs text-slate-500 mt-1">Provide a standard 5-field crontab pattern.</p>
            {:else}
              <div class="p-3 bg-slate-800/50 border border-slate-700/50 rounded-xl text-sm font-semibold text-slate-300">
                Pattern: <code class="bg-slate-700 px-1 rounded">{finalCron}</code>
              </div>
            {/if}
          </div>

          <div>
            <label class="block text-sm font-medium text-slate-300 mb-2">Filename Conventions per Volume</label>
            <div class="space-y-3">
              {#each selectedVolumes as vol}
                <div class="flex items-center gap-3">
                  <span class="text-xs font-semibold text-slate-500 w-24 truncate" title={vol}>{vol}</span>
                  <input
                    type="text"
                    bind:value={volumeFilenameFormats[vol]}
                    class="flex-1 px-4 py-2 border border-slate-600 rounded-xl font-mono text-sm focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500"
                  />
                </div>
              {/each}
            </div>
            <p class="text-xs text-slate-500 mt-2">Supports standard strftime placeholders (e.g. %Y, %m, %d, %H, %M).</p>
          </div>

          <div class="border-t border-slate-700/50 pt-4">
            <div class="flex items-center justify-between mb-4">
              <div>
                <label for="gpg-toggle" class="text-sm font-semibold text-slate-100">GPG Encryption</label>
                <p class="text-xs text-slate-500">Encrypt backup archives using symmetric AES-256 (GPG).</p>
              </div>
              <input
                id="gpg-toggle"
                type="checkbox"
                bind:checked={enableGpg}
                class="h-6 w-11 rounded-full border-slate-600 text-emerald-600 focus:ring-emerald-500 cursor-pointer"
              />
            </div>
            {#if enableGpg}
              <div>
                <label for="gpg-pass" class="block text-sm font-medium text-slate-300 mb-2">GPG Passphrase</label>
                <input
                  id="gpg-pass"
                  type="password"
                  bind:value={gpgPassphrase}
                  placeholder="Enter encryption passphrase"
                  class="w-full px-4 py-2.5 bg-slate-900 border border-slate-600 text-slate-100 rounded-xl focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500 placeholder:text-slate-500"
                />
              </div>
            {/if}
          </div>

          <div>
            <label for="retention" class="block text-sm font-medium text-slate-300 mb-2">Backup Retention Days (0 for unlimited)</label>
            <input
              id="retention"
              type="number"
              bind:value={retentionDays}
              min="0"
              class="w-full px-4 py-2.5 bg-slate-900 border border-slate-600 text-slate-100 rounded-xl focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500 placeholder:text-slate-500"
            />
          </div>

          <div class="flex gap-4">
            <button
              onclick={handleBack}
              class="flex-1 py-3 px-6 border border-slate-600 hover:bg-slate-800/50 text-slate-300 rounded-xl font-semibold transition-colors"
            >
              Back
            </button>
            <button
              onclick={handleStep4Next}
              class="flex-1 py-3 px-6 bg-emerald-600 hover:bg-emerald-500 text-white rounded-xl font-semibold shadow-md transition-colors"
            >
              Next
            </button>
          </div>
        </div>

      <!-- STEP 5: DESTINATION STORAGE -->
      {:else if step === 5}
        <div class="space-y-6">
          <div>
            <h2 class="text-lg font-semibold text-slate-100 mb-1">Step 5: Configure Storage Destination</h2>
            <p class="text-sm text-slate-400">Pick where your backups will be uploaded.</p>
          </div>

          <div>
            <label for="dest-type" class="block text-sm font-medium text-slate-300 mb-2">Storage Backend Type</label>
            <select
              id="dest-type"
              bind:value={backendType}
              class="w-full px-4 py-2.5 border border-slate-600 rounded-xl focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 bg-slate-900"
            >
              <option value="local">📁 Local Host Directory</option>
              <option value="smb">🌐 SMB / CIFS Network Share</option>
              <option value="s3">☁️ Amazon S3 / S3-Compatible</option>
              <option value="webdav">🕸️ WebDAV Storage</option>
              <option value="azure">🟦 Azure Blob Storage</option>
              <option value="dropbox">📦 Dropbox</option>
              <option value="gdrive">🤖 Google Drive</option>
              <option value="sftp">🔑 SSH / SFTP</option>
              {#if savedBackends.length > 0}
                <option value="saved">⭐ Saved App Backend Preset</option>
              {/if}
            </select>
          </div>

          <!-- DYNAMIC FIELDS -->
          {#if backendType === 'local'}
            <div class="space-y-4">
              <div>
                <label for="l-path" class="block text-sm font-medium text-slate-300 mb-2">Host Path</label>
                <input
                  id="l-path"
                  type="text"
                  bind:value={localPath}
                  placeholder="./backups"
                  class="w-full px-4 py-2.5 bg-slate-900 border border-slate-600 text-slate-100 rounded-xl focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500 placeholder:text-slate-500"
                />
                <p class="text-xs text-slate-500 mt-1">This directory on the host will mount to `/archive` inside the backup container.</p>
              </div>

              <div>
                <label class="inline-flex items-center text-sm text-slate-400 cursor-pointer">
                  <input type="checkbox" bind:checked={localUseEnvVars} class="rounded text-emerald-500 focus:ring-emerald-500 mr-2" />
                  Use environment variables instead of hardcoded values in generated Compose file
                </label>
                <p class="text-xs text-slate-500 mt-1 pl-6">
                  Replaces host path with <code>${"{"}LOCAL_BACKUP_PATH{"}"}</code>.
                </p>
              </div>

              <!-- Setup Guide & Example -->
              <div class="bg-indigo-500/10 border border-indigo-500/20 rounded-xl p-4 text-sm text-slate-300">
                <div class="flex items-center gap-2 font-semibold text-slate-200 mb-1">
                  <svg class="w-4 h-4 text-indigo-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>
                  Setup Guide & Example
                </div>
                <p class="text-xs text-slate-400 mb-2">Mount a folder on your host machine to save backups locally. Specify an absolute host path.</p>
                <div class="bg-slate-800/80 p-2.5 rounded-lg border border-indigo-500/20/50 text-xs font-mono">
                  <div class="grid grid-cols-2 gap-y-1">
                    <span class="text-slate-200 font-semibold">Linux Path:</span>
                    <span>/opt/docker-backups</span>
                    <span class="text-slate-200 font-semibold">Windows Path:</span>
                    <span>C:\backups</span>
                  </div>
                </div>
              </div>
            </div>

          {:else if backendType === 'smb'}
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label for="s-host" class="block text-sm font-medium text-slate-300 mb-1">Host / IP</label>
                <input id="s-host" type="text" bind:value={smbHost} placeholder="192.168.1.10" class="w-full px-3 py-2 bg-slate-900 border border-slate-600 text-slate-100 rounded-lg placeholder:text-slate-500 focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500" />
              </div>
              <div>
                <label for="s-share" class="block text-sm font-medium text-slate-300 mb-1">Share Name</label>
                <input id="s-share" type="text" bind:value={smbShare} placeholder="backups" class="w-full px-3 py-2 bg-slate-900 border border-slate-600 text-slate-100 rounded-lg placeholder:text-slate-500 focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500" />
              </div>
              <div>
                <label for="s-path" class="block text-sm font-medium text-slate-300 mb-1">Path inside Share</label>
                <input id="s-path" type="text" bind:value={smbPath} placeholder="subfolder" class="w-full px-3 py-2 bg-slate-900 border border-slate-600 text-slate-100 rounded-lg placeholder:text-slate-500 focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500" />
              </div>
              <div>
                <label for="s-port" class="block text-sm font-medium text-slate-300 mb-1">Port</label>
                <input id="s-port" type="number" bind:value={smbPort} class="w-full px-3 py-2 bg-slate-900 border border-slate-600 text-slate-100 rounded-lg placeholder:text-slate-500 focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500" />
              </div>
              <div>
                <label for="s-user" class="block text-sm font-medium text-slate-300 mb-1">Username</label>
                <input id="s-user" type="text" bind:value={smbUsername} placeholder="backup-user" class="w-full px-3 py-2 bg-slate-900 border border-slate-600 text-slate-100 rounded-lg placeholder:text-slate-500 focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500" />
              </div>
              <div>
                <label for="s-pass" class="block text-sm font-medium text-slate-300 mb-1">Password</label>
                <input id="s-pass" type="password" bind:value={smbPassword} placeholder="••••••••" class="w-full px-3 py-2 bg-slate-900 border border-slate-600 text-slate-100 rounded-lg placeholder:text-slate-500 focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500" />
              </div>
              <div class="col-span-2 mt-2">
                <label class="inline-flex items-center text-sm text-slate-400 cursor-pointer">
                  <input type="checkbox" bind:checked={smbUseEnvVars} class="rounded text-emerald-500 focus:ring-emerald-500 mr-2" />
                  Use environment variables instead of hardcoded values in generated Compose file
                </label>
                <p class="text-xs text-slate-500 mt-1 pl-6">
                  Replaces sensitive fields with <code>${"{"}SMB_BACKUP_ADDR{"}"}</code>, <code>${"{"}SMB_BACKUP_USERNAME{"}"}</code>, and <code>${"{"}SMB_BACKUP_PASSWORD{"}"}</code> for commits. Actual values will still be saved to the database to run backup jobs.
                </p>
              </div>

              <!-- Setup Guide & Example -->
              <div class="col-span-2 bg-indigo-500/10 border border-indigo-500/20 rounded-xl p-4 text-sm text-slate-300 mt-2">
                <div class="flex items-center gap-2 font-semibold text-slate-200 mb-1">
                  <svg class="w-4 h-4 text-indigo-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>
                  Setup Guide & Example
                </div>
                <p class="text-xs text-slate-400 mb-2">Connect to a Windows Share, TrueNAS, Synology NAS, or other network storage via SMB/CIFS.</p>
                <div class="bg-slate-800/80 p-2.5 rounded-lg border border-indigo-500/20/50 text-xs font-mono">
                  <div class="grid grid-cols-2 gap-y-1">
                    <span class="text-slate-200 font-semibold">Host / IP:</span>
                    <span>192.168.1.100 <span class="text-slate-500">(or nas.local)</span></span>
                    <span class="text-slate-200 font-semibold">Share Name:</span>
                    <span>backups</span>
                    <span class="text-slate-200 font-semibold">Path inside Share:</span>
                    <span>docker/postgres <span class="text-slate-500">(optional)</span></span>
                    <span class="text-slate-200 font-semibold">Port:</span>
                    <span>445 <span class="text-slate-500">(default)</span></span>
                  </div>
                </div>
              </div>
            </div>

          {:else if backendType === 's3'}
            <div class="grid grid-cols-2 gap-4">
              <div class="col-span-2">
                <label for="s3-b" class="block text-sm font-medium text-slate-300 mb-1">S3 Bucket Name</label>
                <input id="s3-b" type="text" bind:value={s3Bucket} placeholder="my-backups" class="w-full px-3 py-2 bg-slate-900 border border-slate-600 text-slate-100 rounded-lg placeholder:text-slate-500 focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500" />
              </div>
              <div>
                <label for="s3-key" class="block text-sm font-medium text-slate-300 mb-1">AWS Access Key ID</label>
                <input id="s3-key" type="text" bind:value={s3AccessKey} placeholder="AKIA..." class="w-full px-3 py-2 bg-slate-900 border border-slate-600 text-slate-100 rounded-lg placeholder:text-slate-500 focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500" />
              </div>
              <div>
                <label for="s3-sec" class="block text-sm font-medium text-slate-300 mb-1">AWS Secret Access Key</label>
                <input id="s3-sec" type="password" bind:value={s3SecretKey} placeholder="••••••••" class="w-full px-3 py-2 bg-slate-900 border border-slate-600 text-slate-100 rounded-lg placeholder:text-slate-500 focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500" />
              </div>
              <div class="col-span-2">
                <label for="s3-end" class="block text-sm font-medium text-slate-300 mb-1">Custom Endpoint (for MinIO, Backblaze, etc.)</label>
                <input id="s3-end" type="text" bind:value={s3Endpoint} placeholder="https://minio.local:9000" class="w-full px-3 py-2 bg-slate-900 border border-slate-600 text-slate-100 rounded-lg placeholder:text-slate-500 focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500" />
              </div>
              <div>
                <label for="s3-reg" class="block text-sm font-medium text-slate-300 mb-1">Region</label>
                <input id="s3-reg" type="text" bind:value={s3Region} placeholder="us-east-1" class="w-full px-3 py-2 bg-slate-900 border border-slate-600 text-slate-100 rounded-lg placeholder:text-slate-500 focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500" />
              </div>
              <div>
                <label for="s3-class" class="block text-sm font-medium text-slate-300 mb-1">Storage Class</label>
                <input id="s3-class" type="text" bind:value={s3StorageClass} placeholder="STANDARD" class="w-full px-3 py-2 bg-slate-900 border border-slate-600 text-slate-100 rounded-lg placeholder:text-slate-500 focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500" />
              </div>
              <div class="col-span-2 mt-2">
                <label class="inline-flex items-center text-sm text-slate-400 cursor-pointer">
                  <input type="checkbox" bind:checked={s3UseEnvVars} class="rounded text-emerald-500 focus:ring-emerald-500 mr-2" />
                  Use environment variables instead of hardcoded values in generated Compose file
                </label>
                <p class="text-xs text-slate-500 mt-1 pl-6">
                  Replaces sensitive fields with <code>${"{"}AWS_S3_BUCKET_NAME{"}"}</code>, <code>${"{"}AWS_ACCESS_KEY_ID{"}"}</code>, and <code>${"{"}AWS_SECRET_ACCESS_KEY{"}"}</code> for commits.
                </p>
              </div>

              <!-- Setup Guide & Example -->
              <div class="col-span-2 bg-indigo-500/10 border border-indigo-500/20 rounded-xl p-4 text-sm text-slate-300 mt-2">
                <div class="flex items-center gap-2 font-semibold text-slate-200 mb-1">
                  <svg class="w-4 h-4 text-indigo-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>
                  Setup Guide & Example
                </div>
                <p class="text-xs text-slate-400 mb-2">Store backups in Amazon S3, or any S3-compatible service (MinIO, Backblaze B2, Cloudflare R2, DigitalOcean Spaces).</p>
                <div class="bg-slate-800/80 p-2.5 rounded-lg border border-indigo-500/20/50 text-xs font-mono space-y-2">
                  <div>
                    <span class="text-slate-200 font-semibold block">Example (Amazon S3):</span>
                    <div class="grid grid-cols-2 gap-y-0.5 pl-2 border-l border-indigo-500/20">
                      <span>Bucket Name:</span> <span>my-company-backups</span>
                      <span>Region:</span> <span>us-east-1</span>
                      <span>Access Key:</span> <span>AKIAIOSFODNN7EXAMPLE</span>
                    </div>
                  </div>
                  <div>
                    <span class="text-slate-200 font-semibold block">Example (MinIO / Custom S3):</span>
                    <div class="grid grid-cols-2 gap-y-0.5 pl-2 border-l border-indigo-500/20">
                      <span>Endpoint:</span> <span>https://minio.company.com</span>
                      <span>Region:</span> <span>us-east-1 <span class="text-slate-500">(dummy region needed)</span></span>
                    </div>
                  </div>
                </div>
              </div>
            </div>

          {:else if backendType === 'webdav'}
            <div class="grid grid-cols-2 gap-4">
              <div class="col-span-2">
                <label for="wd-url" class="block text-sm font-medium text-slate-300 mb-1">WebDAV URL</label>
                <input id="wd-url" type="text" bind:value={webdavUrl} placeholder="https://nextcloud.local/remote.php/dav/files/user/" class="w-full px-3 py-2 bg-slate-900 border border-slate-600 text-slate-100 rounded-lg placeholder:text-slate-500 focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500" />
              </div>
              <div>
                <label for="wd-user" class="block text-sm font-medium text-slate-300 mb-1">Username</label>
                <input id="wd-user" type="text" bind:value={webdavUser} class="w-full px-3 py-2 bg-slate-900 border border-slate-600 text-slate-100 rounded-lg placeholder:text-slate-500 focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500" />
              </div>
              <div>
                <label for="wd-pass" class="block text-sm font-medium text-slate-300 mb-1">Password</label>
                <input id="wd-pass" type="password" bind:value={webdavPass} placeholder="••••••••" class="w-full px-3 py-2 bg-slate-900 border border-slate-600 text-slate-100 rounded-lg placeholder:text-slate-500 focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500" />
              </div>
              <div class="col-span-2">
                <label for="wd-path" class="block text-sm font-medium text-slate-300 mb-1">WebDAV Path</label>
                <input id="wd-path" type="text" bind:value={webdavPath} placeholder="/backups" class="w-full px-3 py-2 bg-slate-900 border border-slate-600 text-slate-100 rounded-lg placeholder:text-slate-500 focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500" />
              </div>
              <div class="col-span-2 flex items-center gap-2">
                <input id="wd-insecure" type="checkbox" bind:checked={webdavInsecure} class="rounded text-emerald-600 focus:ring-emerald-500" />
                <label for="wd-insecure" class="text-sm font-semibold text-slate-300">Disable SSL verification (insecure)</label>
              </div>
              <div class="col-span-2 mt-2">
                <label class="inline-flex items-center text-sm text-slate-400 cursor-pointer">
                  <input type="checkbox" bind:checked={webdavUseEnvVars} class="rounded text-emerald-500 focus:ring-emerald-500 mr-2" />
                  Use environment variables instead of hardcoded values in generated Compose file
                </label>
                <p class="text-xs text-slate-500 mt-1 pl-6">
                  Replaces sensitive fields with <code>${"{"}WEBDAV_URL{"}"}</code>, <code>${"{"}WEBDAV_USERNAME{"}"}</code>, and <code>${"{"}WEBDAV_PASSWORD{"}"}</code> for commits.
                </p>
              </div>

              <!-- Setup Guide & Example -->
              <div class="col-span-2 bg-indigo-500/10 border border-indigo-500/20 rounded-xl p-4 text-sm text-slate-300 mt-2">
                <div class="flex items-center gap-2 font-semibold text-slate-200 mb-1">
                  <svg class="w-4 h-4 text-indigo-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>
                  Setup Guide & Example
                </div>
                <p class="text-xs text-slate-400 mb-2">Upload backups to a Nextcloud, ownCloud, or generic WebDAV instance.</p>
                <div class="bg-slate-800/80 p-2.5 rounded-lg border border-indigo-500/20/50 text-xs font-mono">
                  <div class="grid grid-cols-2 gap-y-1">
                    <span class="text-slate-200 font-semibold">WebDAV URL:</span>
                    <span>https://nextcloud.example.com/remote.php/dav/files/backup_user/</span>
                    <span class="text-slate-200 font-semibold">WebDAV Path:</span>
                    <span>/backups/production</span>
                    <span class="text-slate-200 font-semibold">Password:</span>
                    <span><span class="text-slate-500">(Use a generated App Password for Nextcloud)</span></span>
                  </div>
                </div>
              </div>
            </div>

          {:else if backendType === 'azure'}
            <div class="space-y-4">
              <div>
                <label for="az-conn" class="block text-sm font-medium text-slate-300 mb-2">Connection String</label>
                <textarea id="az-conn" bind:value={azureConnString} rows="3" placeholder="DefaultEndpointsProtocol=https;AccountName=..." class="w-full px-3 py-2 bg-slate-900 border border-slate-600 text-slate-100 rounded-lg placeholder:text-slate-500 focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500"></textarea>
              </div>
              <div>
                <label for="az-cont" class="block text-sm font-medium text-slate-300 mb-2">Container Name</label>
                <input id="az-cont" type="text" bind:value={azureContainer} placeholder="backups" class="w-full px-3 py-2 bg-slate-900 border border-slate-600 text-slate-100 rounded-lg placeholder:text-slate-500 focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500" />
              </div>
              <div>
                <label class="inline-flex items-center text-sm text-slate-400 cursor-pointer">
                  <input type="checkbox" bind:checked={azureUseEnvVars} class="rounded text-emerald-500 focus:ring-emerald-500 mr-2" />
                  Use environment variables instead of hardcoded values in generated Compose file
                </label>
                <p class="text-xs text-slate-500 mt-1 pl-6">
                  Replaces sensitive fields with <code>${"{"}AZURE_STORAGE_CONNECTION_STRING{"}"}</code> and <code>${"{"}AZURE_STORAGE_CONTAINER{"}"}</code> for commits.
                </p>
              </div>

              <!-- Setup Guide & Example -->
              <div class="bg-indigo-500/10 border border-indigo-500/20 rounded-xl p-4 text-sm text-slate-300">
                <div class="flex items-center gap-2 font-semibold text-slate-200 mb-1">
                  <svg class="w-4 h-4 text-indigo-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>
                  Setup Guide & Example
                </div>
                <p class="text-xs text-slate-400 mb-2">Save backups directly inside an Azure Blob Storage Container.</p>
                <div class="bg-slate-800/80 p-2.5 rounded-lg border border-indigo-500/20/50 text-xs font-mono">
                  <div class="grid grid-cols-2 gap-y-1">
                    <span class="text-slate-200 font-semibold">Container Name:</span>
                    <span>database-backups</span>
                    <span class="text-slate-200 font-semibold block">Connection String:</span>
                    <span class="break-all whitespace-normal">DefaultEndpointsProtocol=https;AccountName=mybackupstore;AccountKey=...</span>
                  </div>
                </div>
              </div>
            </div>

          {:else if backendType === 'dropbox'}
            <div class="grid grid-cols-2 gap-4">
              <div class="col-span-2">
                <label for="db-tok" class="block text-sm font-medium text-slate-300 mb-1">Access Token</label>
                <input id="db-tok" type="password" bind:value={dropboxToken} placeholder="Token" class="w-full px-3 py-2 bg-slate-900 border border-slate-600 text-slate-100 rounded-lg placeholder:text-slate-500 focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500" />
              </div>
              <div>
                <label for="db-key" class="block text-sm font-medium text-slate-300 mb-1">App Key (Optional)</label>
                <input id="db-key" type="text" bind:value={dropboxAppKey} class="w-full px-3 py-2 bg-slate-900 border border-slate-600 text-slate-100 rounded-lg placeholder:text-slate-500 focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500" />
              </div>
              <div>
                <label for="db-sec" class="block text-sm font-medium text-slate-300 mb-1">App Secret (Optional)</label>
                <input id="db-sec" type="password" bind:value={dropboxAppSecret} class="w-full px-3 py-2 bg-slate-900 border border-slate-600 text-slate-100 rounded-lg placeholder:text-slate-500 focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500" />
              </div>
              <div class="col-span-2">
                <label for="db-path" class="block text-sm font-medium text-slate-300 mb-1">Dropbox Target Path</label>
                <input id="db-path" type="text" bind:value={dropboxPath} placeholder="/backups" class="w-full px-3 py-2 bg-slate-900 border border-slate-600 text-slate-100 rounded-lg placeholder:text-slate-500 focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500" />
              </div>
              <div class="col-span-2 mt-2">
                <label class="inline-flex items-center text-sm text-slate-400 cursor-pointer">
                  <input type="checkbox" bind:checked={dropboxUseEnvVars} class="rounded text-emerald-500 focus:ring-emerald-500 mr-2" />
                  Use environment variables instead of hardcoded values in generated Compose file
                </label>
                <p class="text-xs text-slate-500 mt-1 pl-6">
                  Replaces sensitive fields with <code>${"{"}DROPBOX_ACCESS_TOKEN{"}"}</code> for commits.
                </p>
              </div>

              <!-- Setup Guide & Example -->
              <div class="col-span-2 bg-indigo-500/10 border border-indigo-500/20 rounded-xl p-4 text-sm text-slate-300 mt-2">
                <div class="flex items-center gap-2 font-semibold text-slate-200 mb-1">
                  <svg class="w-4 h-4 text-indigo-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>
                  Setup Guide & Example
                </div>
                <p class="text-xs text-slate-400 mb-2">Upload backups straight to your Dropbox account. Create an app in Dropbox Developer Portal to get a token.</p>
                <div class="bg-slate-800/80 p-2.5 rounded-lg border border-indigo-500/20/50 text-xs font-mono">
                  <div class="grid grid-cols-2 gap-y-1">
                    <span class="text-slate-200 font-semibold">Access Token:</span>
                    <span>sl.B1a2c3d4... <span class="text-slate-500">(Short-lived or Scoped token)</span></span>
                    <span class="text-slate-200 font-semibold">Target Path:</span>
                    <span>/apps/myapp/backups</span>
                  </div>
                </div>
              </div>
            </div>

          {:else if backendType === 'gdrive'}
            <div class="space-y-4">
              <div>
                <label for="gd-folder" class="block text-sm font-medium text-slate-300 mb-2">Google Drive Folder ID</label>
                <input id="gd-folder" type="text" bind:value={gdriveFolderId} placeholder="1a2b3c4d5e..." class="w-full px-3 py-2 bg-slate-900 border border-slate-600 text-slate-100 rounded-lg placeholder:text-slate-500 focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500" />
              </div>
              <div>
                <label for="gd-creds" class="block text-sm font-medium text-slate-300 mb-2">Service Account Credentials JSON</label>
                <textarea id="gd-creds" bind:value={gdriveCredentials} rows="5" placeholder="Copy-paste your Google Cloud service account key .json file contents here..." class="w-full px-3 py-2 border border-slate-600 rounded-lg font-mono text-xs"></textarea>
              </div>
              <div>
                <label for="gd-imp" class="block text-sm font-medium text-slate-300 mb-2">Impersonated User Email (Optional)</label>
                <input id="gd-imp" type="email" bind:value={gdriveImpersonate} placeholder="user@company.com" class="w-full px-3 py-2 bg-slate-900 border border-slate-600 text-slate-100 rounded-lg placeholder:text-slate-500 focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500" />
              </div>
              <div>
                <label class="inline-flex items-center text-sm text-slate-400 cursor-pointer">
                  <input type="checkbox" bind:checked={gdriveUseEnvVars} class="rounded text-emerald-500 focus:ring-emerald-500 mr-2" />
                  Use environment variables instead of hardcoded values in generated Compose file
                </label>
                <p class="text-xs text-slate-500 mt-1 pl-6">
                  Replaces sensitive fields with <code>${"{"}GOOGLE_DRIVE_FOLDER_ID{"}"}</code> and <code>${"{"}GOOGLE_DRIVE_CREDENTIALS{"}"}</code> for commits.
                </p>
              </div>

              <!-- Setup Guide & Example -->
              <div class="bg-indigo-500/10 border border-indigo-500/20 rounded-xl p-4 text-sm text-slate-300">
                <div class="flex items-center gap-2 font-semibold text-slate-200 mb-1">
                  <svg class="w-4 h-4 text-indigo-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>
                  Setup Guide & Example
                </div>
                <p class="text-xs text-slate-400 mb-2">Upload backups to a shared Google Drive Folder using a Google Cloud Service Account credentials JSON key.</p>
                <div class="bg-slate-800/80 p-2.5 rounded-lg border border-indigo-500/20/50 text-xs font-mono space-y-1">
                  <div><span class="text-slate-200 font-semibold">Folder ID:</span> <span class="break-all">1Z9Y8X7W6V5U4T3S2R1Q...</span></div>
                  <div class="text-slate-500 pl-4 border-l border-indigo-500/20">
                    Get ID from folder URL: `https://drive.google.com/drive/folders/&lt;FOLDER_ID&gt;`
                  </div>
                  <div><span class="text-slate-200 font-semibold block">Service Account Credentials JSON:</span></div>
                  <pre class="bg-slate-800 p-2 rounded text-[10px] text-slate-400">{`{\n  "type": "service_account",\n  "project_id": "my-project",\n  "private_key_id": "abcd...",\n  "private_key": "-----BEGIN PRIVATE KEY-----\\n...",\n  "client_email": "backup-sa@my-project.iam.gserviceaccount.com"\n}`}</pre>
                </div>
              </div>
            </div>

          {:else if backendType === 'sftp'}
            <div class="grid grid-cols-2 gap-4">
              <div class="col-span-2">
                <label for="sf-host" class="block text-sm font-medium text-slate-300 mb-1">SSH/SFTP Host</label>
                <input id="sf-host" type="text" bind:value={sftpHost} placeholder="sftp.example.com" class="w-full px-3 py-2 bg-slate-900 border border-slate-600 text-slate-100 rounded-lg placeholder:text-slate-500 focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500" />
              </div>
              <div>
                <label for="sf-port" class="block text-sm font-medium text-slate-300 mb-1">Port</label>
                <input id="sf-port" type="number" bind:value={sftpPort} class="w-full px-3 py-2 bg-slate-900 border border-slate-600 text-slate-100 rounded-lg placeholder:text-slate-500 focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500" />
              </div>
              <div>
                <label for="sf-user" class="block text-sm font-medium text-slate-300 mb-1">User</label>
                <input id="sf-user" type="text" bind:value={sftpUser} placeholder="root" class="w-full px-3 py-2 bg-slate-900 border border-slate-600 text-slate-100 rounded-lg placeholder:text-slate-500 focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500" />
              </div>
              <div>
                <label for="sf-pass" class="block text-sm font-medium text-slate-300 mb-1">Password</label>
                <input id="sf-pass" type="password" bind:value={sftpPass} class="w-full px-3 py-2 bg-slate-900 border border-slate-600 text-slate-100 rounded-lg placeholder:text-slate-500 focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500" />
              </div>
              <div class="col-span-2">
                <label for="sf-key" class="block text-sm font-medium text-slate-300 mb-1">Private SSH Key (Optional)</label>
                <textarea id="sf-key" bind:value={sftpKey} rows="3" placeholder="-----BEGIN OPENSSH PRIVATE KEY-----..." class="w-full px-3 py-2 border border-slate-600 rounded-lg font-mono text-xs"></textarea>
              </div>
              <div class="col-span-2">
                <label for="sf-path" class="block text-sm font-medium text-slate-300 mb-1">Remote Target Path</label>
                <input id="sf-path" type="text" bind:value={sftpPath} placeholder="/home/user/backups" class="w-full px-3 py-2 bg-slate-900 border border-slate-600 text-slate-100 rounded-lg placeholder:text-slate-500 focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500" />
              </div>
              <div class="col-span-2 mt-2">
                <label class="inline-flex items-center text-sm text-slate-400 cursor-pointer">
                  <input type="checkbox" bind:checked={sftpUseEnvVars} class="rounded text-emerald-500 focus:ring-emerald-500 mr-2" />
                  Use environment variables instead of hardcoded values in generated Compose file
                </label>
                <p class="text-xs text-slate-500 mt-1 pl-6">
                  Replaces sensitive fields with <code>${"{"}SSH_HOST{"}"}</code>, <code>${"{"}SSH_USER{"}"}</code>, and <code>${"{"}SSH_PASSWORD{"}"}</code> for commits.
                </p>
              </div>

              <!-- Setup Guide & Example -->
              <div class="col-span-2 bg-indigo-500/10 border border-indigo-500/20 rounded-xl p-4 text-sm text-slate-300 mt-2">
                <div class="flex items-center gap-2 font-semibold text-slate-200 mb-1">
                  <svg class="w-4 h-4 text-indigo-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>
                  Setup Guide & Example
                </div>
                <p class="text-xs text-slate-400 mb-2">Upload backups to a remote server or NAS using SFTP.</p>
                <div class="bg-slate-800/80 p-2.5 rounded-lg border border-indigo-500/20/50 text-xs font-mono">
                  <div class="grid grid-cols-2 gap-y-1">
                    <span class="text-slate-200 font-semibold">Host:</span>
                    <span>nas.local <span class="text-slate-500">(or 192.168.1.50)</span></span>
                    <span class="text-slate-200 font-semibold">User:</span>
                    <span>backupuser</span>
                    <span class="text-slate-200 font-semibold">Port:</span>
                    <span>22 <span class="text-slate-500">(default SSH port)</span></span>
                    <span class="text-slate-200 font-semibold">Path:</span>
                    <span>/share/backups/databases</span>
                  </div>
                </div>
              </div>
            </div>

          {:else if backendType === 'saved'}
            <div class="space-y-4">
              <div>
                <label for="s-back" class="block text-sm font-medium text-slate-300 mb-2">Choose saved backend preset</label>
                <select
                  id="s-back"
                  bind:value={selectedSavedBackendId}
                  class="w-full px-4 py-2.5 border border-slate-600 rounded-xl focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 bg-slate-900"
                >
                  <option value="">Select preset...</option>
                  {#each savedBackends as backend}
                    <option value={backend.id}>{backend.name} ({backend.credentials.type})</option>
                  {/each}
                </select>
              </div>

              {#if savedBackends.find(b => b.id === selectedSavedBackendId)?.credentials.type === 'smb'}
                <div class="mt-4">
                  <label class="inline-flex items-center text-sm text-slate-400 cursor-pointer">
                    <input type="checkbox" bind:checked={smbUseEnvVars} class="rounded text-emerald-500 focus:ring-emerald-500 mr-2" />
                    Use environment variables instead of hardcoded values in generated Compose file
                  </label>
                  <p class="text-xs text-slate-500 mt-1 pl-6">
                    Replaces sensitive fields with <code>${"{"}SMB_BACKUP_ADDR{"}"}</code>, <code>${"{"}SMB_BACKUP_USERNAME{"}"}</code>, and <code>${"{"}SMB_BACKUP_PASSWORD{"}"}</code> for commits. Actual values will still be saved to the database to run backup jobs.
                  </p>
                </div>
              {/if}
            </div>
          {/if}

          <div class="flex gap-4">
            <button
              onclick={handleBack}
              class="flex-1 py-3 px-6 border border-slate-600 hover:bg-slate-800/50 text-slate-300 rounded-xl font-semibold transition-colors"
            >
              Back
            </button>
            <button
              onclick={handleStep5Generate}
              disabled={loading}
              class="flex-1 py-3 px-6 bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white rounded-xl font-semibold shadow-md transition-colors"
            >
              {loading ? 'Generating...' : 'Generate Compose'}
            </button>
          </div>
        </div>

      <!-- STEP 6: OUTPUT & COMPLETE -->
      {:else}
        <div class="space-y-6">
          <div>
            <h2 class="text-lg font-semibold text-slate-100 mb-1">Step 6: Completed Compose File</h2>
            <p class="text-sm text-slate-400">Your docker-compose.yml is ready! You can copy, download, or save it to this manager.</p>
          </div>

          <div class="relative">
            <pre class="bg-slate-900 text-slate-100 p-5 rounded-xl text-sm font-mono overflow-auto max-h-96 border border-slate-700 shadow-inner select-all">{generatedCompose}</pre>
            <div class="absolute right-4 top-4 flex gap-2">
              <button
                onclick={handleCopy}
                class="py-1.5 px-3 bg-slate-800 hover:bg-slate-700 border border-slate-700 text-slate-200 text-xs font-semibold rounded-lg shadow transition-colors flex items-center gap-1.5"
              >
                {#if copySuccess}
                  <span>Copied!</span>
                {:else}
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 5H6a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2v-1M8 5a2 2 0 002 2h2a2 2 0 002-2M8 5a2 2 0 002 2h2a2 2 0 002-2M8 5a2 2 0 012-2h2a2 2 0 012 2m0 0h2a2 2 0 012 2v3m2 4H10m0 0l3-3m-3 3l3 3" />
                  </svg>
                  <span>Copy</span>
                {/if}
              </button>
              <button
                onclick={handleDownload}
                class="py-1.5 px-3 bg-slate-800 hover:bg-slate-700 border border-slate-700 text-slate-200 text-xs font-semibold rounded-lg shadow transition-colors flex items-center gap-1.5"
              >
                <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                </svg>
                <span>Download</span>
              </button>
            </div>
          </div>

          <div class="p-4 bg-emerald-500/10 border border-emerald-500/20 rounded-xl text-emerald-400 text-sm">
            <strong>Next Step:</strong> Replace the `docker-compose.yml` file in your stack or Git repository with this new content.
          </div>

          <div class="flex gap-4">
            <button
              onclick={handleBack}
              class="flex-1 py-3 px-6 border border-slate-600 hover:bg-slate-800/50 text-slate-300 rounded-xl font-semibold transition-colors"
            >
              Back
            </button>
            {#if sourceType === 'project'}
              <a
                href="/projects/{selectedProjectId}"
                class="flex-1 py-3 px-6 bg-emerald-600 hover:bg-emerald-500 text-white rounded-xl font-semibold shadow-md transition-colors text-center flex items-center justify-center"
              >
                Done
              </a>
            {:else}
              <button
                onclick={handleSaveProject}
                disabled={loading}
                class="flex-1 py-3 px-6 bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white rounded-xl font-semibold shadow-md transition-colors"
              >
                {loading ? 'Saving Project...' : 'Save as Project'}
              </button>
            {/if}
          </div>
        </div>
      {/if}

    </div>
  </div>
</div>
