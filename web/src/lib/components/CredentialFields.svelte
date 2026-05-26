<script lang="ts">
  import type { Credentials } from '$lib/types';

  interface Props {
    credentials: Credentials;
    isEdit?: boolean;
    theme?: 'indigo' | 'emerald';
  }

  let { credentials = $bindable(), isEdit = false, theme = 'indigo' }: Props = $props();

  // Ensure appropriate sub-object is instantiated when type changes
  $effect(() => {
    const type = credentials.type;
    if (type === 'local' && !credentials.local) {
      credentials.local = { path: '' };
    } else if (type === 'smb' && !credentials.smb) {
      credentials.smb = { host: '', share: '', path: '', username: '', password: '', port: 445 };
    } else if (type === 's3' && !credentials.s3) {
      credentials.s3 = { bucket: '', access_key: '', secret_key: '', endpoint: '', region: 'us-east-1', storage_class: 'STANDARD' };
    } else if (type === 'webdav' && !credentials.webdav) {
      credentials.webdav = { url: '', username: '', password: '', path: '', insecure: false };
    } else if (type === 'azure' && !credentials.azure) {
      credentials.azure = { connection_string: '', container: '' };
    } else if (type === 'dropbox' && !credentials.dropbox) {
      credentials.dropbox = { access_token: '', app_key: '', app_secret: '', remote_path: '' };
    } else if (type === 'gdrive' && !credentials.gdrive) {
      credentials.gdrive = { folder_id: '', credentials: '', impersonate: '' };
    } else if (type === 'sftp' && !credentials.sftp) {
      credentials.sftp = { host: '', user: '', port: 22, password: '', private_key: '', remote_path: '' };
    }
  });

  const focusRing = $derived(
    theme === 'emerald'
      ? 'focus:ring-emerald-500/20 focus:border-emerald-500'
      : 'focus:ring-indigo-500/20 focus:border-indigo-500'
  );
  
  const checkboxColor = $derived(
    theme === 'emerald'
      ? 'text-emerald-500 focus:ring-emerald-500/20'
      : 'text-indigo-500 focus:ring-indigo-500/20'
  );
</script>

{#if credentials.type === 'local' && credentials.local}
  <div>
    <label for="local-path" class="block text-sm font-medium text-slate-300 mb-1">Backup Path</label>
    <input
      id="local-path"
      type="text"
      bind:value={credentials.local.path}
      class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 {focusRing} transition-all"
      placeholder="/mnt/backups"
    />
  </div>
{:else if credentials.type === 'smb' && credentials.smb}
  <div class="grid grid-cols-2 gap-4">
    <div>
      <label for="smb-host" class="block text-sm font-medium text-slate-300 mb-1">Host</label>
      <input
        id="smb-host"
        type="text"
        bind:value={credentials.smb.host}
        class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 {focusRing} transition-all"
        placeholder="192.168.1.10"
      />
    </div>
    <div>
      <label for="smb-share" class="block text-sm font-medium text-slate-300 mb-1">Share</label>
      <input
        id="smb-share"
        type="text"
        bind:value={credentials.smb.share}
        class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 {focusRing} transition-all"
        placeholder="backups"
      />
    </div>
    <div>
      <label for="smb-path" class="block text-sm font-medium text-slate-300 mb-1">Path (optional)</label>
      <input
        id="smb-path"
        type="text"
        bind:value={credentials.smb.path}
        class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 {focusRing} transition-all"
        placeholder="subfolder"
      />
    </div>
    <div>
      <label for="smb-port" class="block text-sm font-medium text-slate-300 mb-1">Port</label>
      <input
        id="smb-port"
        type="number"
        bind:value={credentials.smb.port}
        class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 {focusRing} transition-all"
        placeholder="445"
      />
    </div>
    <div>
      <label for="smb-username" class="block text-sm font-medium text-slate-300 mb-1">Username</label>
      <input
        id="smb-username"
        type="text"
        bind:value={credentials.smb.username}
        class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 {focusRing} transition-all"
        placeholder="user"
      />
    </div>
    <div>
      <label for="smb-password" class="block text-sm font-medium text-slate-300 mb-1">Password</label>
      <input
        id="smb-password"
        type="password"
        bind:value={credentials.smb.password}
        class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 {focusRing} transition-all"
        placeholder={isEdit ? 'Enter to change' : 'password'}
      />
    </div>
  </div>
{:else}
  <div class="p-2.5 bg-amber-500/10 border border-amber-500/20 rounded-lg text-amber-400 text-xs flex items-start gap-2 mb-2">
    <svg class="w-4 h-4 text-amber-400 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
    </svg>
    <span>This backend is in Beta. Things may not work as expected — please file a GitHub issue with the repository if you encounter any bugs.</span>
  </div>

  {#if credentials.type === 's3' && credentials.s3}
    <div class="grid grid-cols-2 gap-4">
      <div>
        <label for="s3-bucket" class="block text-sm font-medium text-slate-300 mb-1">Bucket Name</label>
        <input
          id="s3-bucket"
          type="text"
          bind:value={credentials.s3.bucket}
          class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 {focusRing} transition-all"
          placeholder="my-backups"
        />
      </div>
      <div>
        <label for="s3-region" class="block text-sm font-medium text-slate-300 mb-1">Region</label>
        <input
          id="s3-region"
          type="text"
          bind:value={credentials.s3.region}
          class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 {focusRing} transition-all"
          placeholder="us-east-1"
        />
      </div>
      <div>
        <label for="s3-access" class="block text-sm font-medium text-slate-300 mb-1">Access Key</label>
        <input
          id="s3-access"
          type="text"
          bind:value={credentials.s3.access_key}
          class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 {focusRing} transition-all"
          placeholder="Access key"
        />
      </div>
      <div>
        <label for="s3-secret" class="block text-sm font-medium text-slate-300 mb-1">Secret Key</label>
        <input
          id="s3-secret"
          type="password"
          bind:value={credentials.s3.secret_key}
          class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 {focusRing} transition-all"
          placeholder={isEdit ? 'Enter to change' : 'password'}
        />
      </div>
      <div class="col-span-2">
        <label for="s3-endpoint" class="block text-sm font-medium text-slate-300 mb-1">Custom Endpoint (optional)</label>
        <input
          id="s3-endpoint"
          type="text"
          bind:value={credentials.s3.endpoint}
          class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 {focusRing} transition-all"
          placeholder="https://minio.example.com"
        />
      </div>
      <div class="col-span-2">
        <label for="s3-class" class="block text-sm font-medium text-slate-300 mb-1">Storage Class</label>
        <input
          id="s3-class"
          type="text"
          bind:value={credentials.s3.storage_class}
          class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 {focusRing} transition-all"
          placeholder="STANDARD"
        />
      </div>
    </div>
  {:else if credentials.type === 'webdav' && credentials.webdav}
    <div class="grid grid-cols-2 gap-4">
      <div class="col-span-2">
        <label for="webdav-url" class="block text-sm font-medium text-slate-300 mb-1">WebDAV Server URL</label>
        <input
          id="webdav-url"
          type="text"
          bind:value={credentials.webdav.url}
          class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 {focusRing} transition-all"
          placeholder="https://nextcloud.example.com/remote.php/webdav"
        />
      </div>
      <div>
        <label for="webdav-user" class="block text-sm font-medium text-slate-300 mb-1">Username</label>
        <input
          id="webdav-user"
          type="text"
          bind:value={credentials.webdav.username}
          class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 {focusRing} transition-all"
          placeholder="username"
        />
      </div>
      <div>
        <label for="webdav-pass" class="block text-sm font-medium text-slate-300 mb-1">Password</label>
        <input
          id="webdav-pass"
          type="password"
          bind:value={credentials.webdav.password}
          class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 {focusRing} transition-all"
          placeholder={isEdit ? 'Enter to change' : 'password'}
        />
      </div>
      <div class="col-span-2">
        <label for="webdav-path" class="block text-sm font-medium text-slate-300 mb-1">Remote Path (optional)</label>
        <input
          id="webdav-path"
          type="text"
          bind:value={credentials.webdav.path}
          class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 {focusRing} transition-all"
          placeholder="/backups"
        />
      </div>
      <div class="col-span-2 flex items-center gap-2 py-1">
        <input
          id="webdav-insecure"
          type="checkbox"
          bind:checked={credentials.webdav.insecure}
          class="h-4 w-4 rounded border-slate-700 {checkboxColor} bg-slate-900"
        />
        <label for="webdav-insecure" class="text-sm font-medium text-slate-300">Skip TLS verification (Insecure)</label>
      </div>
    </div>
  {:else if credentials.type === 'azure' && credentials.azure}
    <div class="space-y-4">
      <div>
        <label for="azure-conn" class="block text-sm font-medium text-slate-300 mb-1">Connection String</label>
        <input
          id="azure-conn"
          type="password"
          bind:value={credentials.azure.connection_string}
          class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 {focusRing} transition-all"
          placeholder="DefaultEndpointsProtocol=..."
        />
      </div>
      <div>
        <label for="azure-container" class="block text-sm font-medium text-slate-300 mb-1">Container Name</label>
        <input
          id="azure-container"
          type="text"
          bind:value={credentials.azure.container}
          class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 {focusRing} transition-all"
          placeholder="my-backups"
        />
      </div>
    </div>
  {:else if credentials.type === 'dropbox' && credentials.dropbox}
    <div class="grid grid-cols-2 gap-4">
      <div class="col-span-2">
        <label for="db-token" class="block text-sm font-medium text-slate-300 mb-1">Access Token</label>
        <input
          id="db-token"
          type="password"
          bind:value={credentials.dropbox.access_token}
          class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 {focusRing} transition-all"
          placeholder="Access Token"
        />
      </div>
      <div>
        <label for="db-key" class="block text-sm font-medium text-slate-300 mb-1">App Key</label>
        <input
          id="db-key"
          type="text"
          bind:value={credentials.dropbox.app_key}
          class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 {focusRing} transition-all"
          placeholder="App key"
        />
      </div>
      <div>
        <label for="db-secret" class="block text-sm font-medium text-slate-300 mb-1">App Secret</label>
        <input
          id="db-secret"
          type="password"
          bind:value={credentials.dropbox.app_secret}
          class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 {focusRing} transition-all"
          placeholder={isEdit ? 'Enter to change' : 'password'}
        />
      </div>
      <div class="col-span-2">
        <label for="db-path" class="block text-sm font-medium text-slate-300 mb-1">Remote Path (optional)</label>
        <input
          id="db-path"
          type="text"
          bind:value={credentials.dropbox.remote_path}
          class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 {focusRing} transition-all"
          placeholder="/backups"
        />
      </div>
    </div>
  {:else if credentials.type === 'gdrive' && credentials.gdrive}
    <div class="space-y-4">
      <div>
        <label for="gd-folder" class="block text-sm font-medium text-slate-300 mb-1">Folder ID</label>
        <input
          id="gd-folder"
          type="text"
          bind:value={credentials.gdrive.folder_id}
          class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 {focusRing} transition-all"
          placeholder="Google Drive Folder ID"
        />
      </div>
      <div>
        <label for="gd-creds" class="block text-sm font-medium text-slate-300 mb-1">Service Account JSON Credentials</label>
        <textarea
          id="gd-creds"
          rows="4"
          bind:value={credentials.gdrive.credentials}
          class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg font-mono text-xs text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 {focusRing} transition-all"
          placeholder="Paste service account JSON block..."
        ></textarea>
      </div>
      <div>
        <label for="gd-impersonate" class="block text-sm font-medium text-slate-300 mb-1">Impersonate Email (optional)</label>
        <input
          id="gd-impersonate"
          type="email"
          bind:value={credentials.gdrive.impersonate}
          class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 {focusRing} transition-all"
          placeholder="user@domain.com"
        />
      </div>
    </div>
  {:else if credentials.type === 'sftp' && credentials.sftp}
    <div class="grid grid-cols-2 gap-4">
      <div class="col-span-2">
        <label for="sftp-host" class="block text-sm font-medium text-slate-300 mb-1">SFTP Host / IP</label>
        <input
          id="sftp-host"
          type="text"
          bind:value={credentials.sftp.host}
          class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 {focusRing} transition-all"
          placeholder="sftp.example.com"
        />
      </div>
      <div>
        <label for="sftp-user" class="block text-sm font-medium text-slate-300 mb-1">SFTP Username</label>
        <input
          id="sftp-user"
          type="text"
          bind:value={credentials.sftp.user}
          class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 {focusRing} transition-all"
          placeholder="backup-user"
        />
      </div>
      <div>
        <label for="sftp-port" class="block text-sm font-medium text-slate-300 mb-1">Port</label>
        <input
          id="sftp-port"
          type="number"
          bind:value={credentials.sftp.port}
          class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 {focusRing} transition-all"
          placeholder="22"
        />
      </div>
      <div class="col-span-2">
        <label for="sftp-pass" class="block text-sm font-medium text-slate-300 mb-1">Password Auth (optional)</label>
        <input
          id="sftp-pass"
          type="password"
          bind:value={credentials.sftp.password}
          class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 {focusRing} transition-all"
          placeholder={isEdit ? 'Enter to change' : 'password'}
        />
      </div>
      <div class="col-span-2">
        <label for="sftp-key" class="block text-sm font-medium text-slate-300 mb-1">Private Key Auth (optional)</label>
        <textarea
          id="sftp-key"
          rows="4"
          bind:value={credentials.sftp.private_key}
          class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg font-mono text-xs text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 {focusRing} transition-all"
          placeholder="Paste OpenSSH Private Key block..."
        ></textarea>
      </div>
      <div class="col-span-2">
        <label for="sftp-path" class="block text-sm font-medium text-slate-300 mb-1">Remote Path (optional)</label>
        <input
          id="sftp-path"
          type="text"
          bind:value={credentials.sftp.remote_path}
          class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 {focusRing} transition-all"
          placeholder="/var/backups"
        />
      </div>
    </div>
  {/if}
{/if}
