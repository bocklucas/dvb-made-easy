<script lang="ts">
  import type { ProjectSource } from '$lib/types';

  interface Method {
    id: ProjectSource;
    label: string;
    description: string;
    icon: string;
  }

  interface Props {
    onSelect: (method: ProjectSource) => void;
  }

  let { onSelect }: Props = $props();

  // Data-driven: add new methods here when their tickets are implemented
  const methods: Method[] = [
    {
      id: 'paste',
      label: 'Paste Compose',
      description: 'Paste your docker-compose.yml content directly',
      icon: 'M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2'
    },
    {
      id: 'git',
      label: 'Import from Git',
      description: 'Clone a Git repository containing your compose file',
      icon: 'M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2z'
    },
    {
      id: 'portainer',
      label: 'Connect Portainer',
      description: 'Import stacks from a running Portainer instance',
      icon: 'M5 12h14M12 5l7 7-7 7'
    }
  ];
</script>

<div class="space-y-4">
  <div>
    <h2 class="text-lg font-semibold text-gray-900">How do you want to import your project?</h2>
    <p class="text-sm text-gray-500 mt-1">Choose an import method to get started.</p>
  </div>

  <div class="space-y-3">
    {#each methods as method}
      <button
        onclick={() => onSelect(method.id)}
        class="w-full flex items-start gap-4 p-4 border border-gray-200 rounded-lg hover:border-blue-400 hover:bg-blue-50 text-left transition-colors group"
      >
        <div class="flex-shrink-0 w-10 h-10 rounded-lg bg-blue-100 group-hover:bg-blue-200 flex items-center justify-center transition-colors">
          <svg class="w-5 h-5 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d={method.icon} />
          </svg>
        </div>
        <div>
          <div class="font-medium text-gray-900">{method.label}</div>
          <div class="text-sm text-gray-500 mt-0.5">{method.description}</div>
        </div>
        <svg class="w-5 h-5 text-gray-300 group-hover:text-blue-400 ml-auto flex-shrink-0 mt-2.5 transition-colors" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
        </svg>
      </button>
    {/each}
  </div>
</div>
