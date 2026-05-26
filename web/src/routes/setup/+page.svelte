<script lang="ts">
  import type { VolumeInfo, ProjectSource, InferenceResult, Credentials } from '$lib/types';
  import ImportMethodSelect from '$lib/components/ImportMethodSelect.svelte';
  import WizardStep1 from '$lib/components/WizardStep1.svelte';
  import WizardStep2 from '$lib/components/WizardStep2.svelte';
  import WizardStep3 from '$lib/components/WizardStep3.svelte';
  import InferenceSummary from '$lib/components/InferenceSummary.svelte';
  import GitImportStep1 from '$lib/components/GitImportStep1.svelte';
  import GitImportStep2 from '$lib/components/GitImportStep2.svelte';
  import PortainerStep1 from '$lib/components/PortainerStep1.svelte';
  import PortainerStep2 from '$lib/components/PortainerStep2.svelte';

  // step 0 = method selection, steps 1-N = method-specific sub-flow
  // paste flow: 1 = compose+name, 1.5 = inference, 2 = storage, 3 = review
  // We encode 1.5 as step 15 internally.
  let currentStep = $state(0);
  let selectedMethod = $state<ProjectSource | null>(null);
  let projectId = $state('');
  let projectName = $state('');
  let volumes = $state<VolumeInfo[]>([]);
  let backendType = $state<Credentials['type']>('local');
  let inferenceResult = $state<InferenceResult | undefined>(undefined);

  // Portainer sub-flow state
  let portainerUrl = $state('');
  let portainerApiKey = $state('');
  let portainerEndpointId = $state(0);
  let portainerSavedSourceId = $state('');

  function handleMethodSelect(method: ProjectSource) {
    selectedMethod = method;
    currentStep = 1;
  }

  function handleStep1Complete(id: string, vols: VolumeInfo[], name: string, inference?: InferenceResult) {
    projectId = id;
    volumes = vols;
    projectName = name;
    inferenceResult = inference;
    // If we have an inference result, show the summary step; otherwise skip to storage
    currentStep = inference ? 15 : 2;
  }

  function handleInferenceContinue() {
    currentStep = 2;
  }

  function handleInferenceBack() {
    currentStep = 1;
  }

  // Git sub-flow: step 1 = import, step 2 = show volumes, step 3 = storage, step 4 = review
  function handleGitImportComplete(id: string, vols: VolumeInfo[], name: string) {
    projectId = id;
    volumes = vols;
    projectName = name;
    currentStep = 2;
  }

  function handleGitStep2Next() {
    currentStep = 3;
  }

  function handleGitStep2Back() {
    currentStep = 1;
  }

  // Portainer sub-flow: step 1 = connect, step 2 = select stack, step 3 = storage, step 4 = review
  function handlePortainerStep1Next(url: string, key: string, endpointId: number, savedSourceId?: string) {
    portainerUrl = url;
    portainerApiKey = key;
    portainerEndpointId = endpointId;
    portainerSavedSourceId = savedSourceId || '';
    currentStep = 2;
  }

  function handlePortainerStep2Complete(id: string, vols: VolumeInfo[], name: string) {
    projectId = id;
    volumes = vols;
    projectName = name;
    currentStep = 3;
  }

  function handleStep2Complete(type: Credentials['type']) {
    backendType = type;
    currentStep = selectedMethod === 'git' || selectedMethod === 'portainer' ? 4 : 3;
  }

  function handleStep2Back() {
    if (selectedMethod === 'paste') {
      // Go back to inference summary if we have one, otherwise to compose step
      currentStep = inferenceResult ? 15 : 1;
    } else {
      currentStep = selectedMethod === 'git' || selectedMethod === 'portainer' ? 2 : 1;
    }
  }

  function handleStep1Back() {
    currentStep = 0;
    selectedMethod = null;
  }

  function handleStep3Back() {
    currentStep = selectedMethod === 'git' || selectedMethod === 'portainer' ? 3 : 2;
  }

  // Steps shown in the indicator for each flow
  // paste: 1, 2, 3 (we show inference as part of step 1 visually, progress after step 1 is done)
  const pasteSteps = [1, 2, 3];
  const gitSteps = [1, 2, 3, 4];
  const portainerSteps = [1, 2, 3, 4];

  // Map internal step numbers to visual step indicator numbers
  function visualStep(): number {
    if (currentStep === 15) return 2; // inference summary is visually between 1 and 2
    return currentStep;
  }
</script>

<div class="min-h-screen bg-slate-950 flex flex-col items-center justify-start pt-12 px-4 animate-fade-in">
  <div class="w-full max-w-xl">
    <h1 class="text-2xl font-light text-slate-100 mb-2">
      Setup New Project
      {#if projectName}
        <span class="text-lg font-normal text-slate-500 block sm:inline sm:ml-2">for {projectName}</span>
      {/if}
    </h1>

    {#if currentStep > 0 && (projectName || selectedMethod)}
      <div class="bg-indigo-500/10 border border-indigo-500/20 rounded-xl p-4 mb-6 text-xs flex flex-wrap gap-x-6 gap-y-2 text-slate-400">
        <div class="flex items-center gap-1">
          <span class="font-semibold text-slate-300">Source:</span>
          <span class="capitalize px-1.5 py-0.5 rounded bg-indigo-500/20 text-indigo-300 font-medium">{selectedMethod}</span>
        </div>
        {#if projectName}
          <div><span class="font-semibold text-slate-300">Project Name:</span> <span class="text-slate-200 font-medium">{projectName}</span></div>
        {/if}
        {#if volumes && volumes.length > 0}
          <div><span class="font-semibold text-slate-300">Volumes:</span> <span class="text-slate-200 font-medium">{volumes.length} detected</span></div>
        {/if}
        {#if currentStep >= 3 && backendType}
          <div class="flex items-center gap-1">
            <span class="font-semibold text-slate-300">Storage Backend:</span>
            <span class="uppercase px-1.5 py-0.5 rounded bg-slate-700/50 text-slate-300 font-mono font-medium">{backendType}</span>
          </div>
        {/if}
      </div>
    {/if}

    {#if currentStep > 0 && selectedMethod === 'paste'}
      <!-- Step indicator for the paste sub-flow (steps 1-3) -->
      <div class="flex items-center gap-2 mb-8">
        {#each pasteSteps as step}
          <div class="flex items-center gap-2">
            <div
              class="w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium transition-all duration-500 {visualStep() >= step ? 'bg-gradient-to-r from-indigo-500 to-violet-500 text-white shadow-lg shadow-indigo-500/25' : 'bg-slate-800 text-slate-500'}"
            >
              {step}
            </div>
            {#if step < pasteSteps.length}
              <div class="w-12 h-0.5 transition-all duration-500 {visualStep() > step ? 'bg-indigo-500' : 'bg-slate-700'}"></div>
            {/if}
          </div>
        {/each}
      </div>
    {:else if currentStep > 0 && selectedMethod === 'git'}
      <!-- Step indicator for the git sub-flow (steps 1-4) -->
      <div class="flex items-center gap-2 mb-8">
        {#each gitSteps as step}
          <div class="flex items-center gap-2">
            <div
              class="w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium transition-all duration-500 {currentStep >= step ? 'bg-gradient-to-r from-indigo-500 to-violet-500 text-white shadow-lg shadow-indigo-500/25' : 'bg-slate-800 text-slate-500'}"
            >
              {step}
            </div>
            {#if step < gitSteps.length}
              <div class="w-12 h-0.5 transition-all duration-500 {currentStep > step ? 'bg-indigo-500' : 'bg-slate-700'}"></div>
            {/if}
          </div>
        {/each}
      </div>
    {:else if currentStep > 0 && selectedMethod === 'portainer'}
      <!-- Step indicator for the portainer sub-flow (steps 1-4) -->
      <div class="flex items-center gap-2 mb-8">
        {#each portainerSteps as step}
          <div class="flex items-center gap-2">
            <div
              class="w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium transition-all duration-500 {currentStep >= step ? 'bg-gradient-to-r from-indigo-500 to-violet-500 text-white shadow-lg shadow-indigo-500/25' : 'bg-slate-800 text-slate-500'}"
            >
              {step}
            </div>
            {#if step < portainerSteps.length}
              <div class="w-12 h-0.5 transition-all duration-500 {currentStep > step ? 'bg-indigo-500' : 'bg-slate-700'}"></div>
            {/if}
          </div>
        {/each}
      </div>
    {:else if currentStep === 0}
      <!-- Spacer matching indicator height so the card position stays consistent -->
      <div class="mb-8"></div>
    {/if}

    <div class="bg-slate-800/50 rounded-xl shadow-xl shadow-black/20 border border-slate-700/50 p-6">
      {#if currentStep === 0}
        <ImportMethodSelect onSelect={handleMethodSelect} />
      {:else if currentStep === 1 && selectedMethod === 'paste'}
        <WizardStep1 onComplete={handleStep1Complete} onBack={handleStep1Back} />
      {:else if currentStep === 15 && selectedMethod === 'paste' && inferenceResult}
        <!-- Inference summary step (step 1.5) -->
        <div class="mb-4">
          <h2 class="text-base font-semibold text-slate-100">What we detected</h2>
          <p class="text-sm text-slate-500 mt-0.5">Review auto-detected values before setting up storage.</p>
        </div>
        <InferenceSummary
          inference={inferenceResult}
          onContinue={handleInferenceContinue}
          onBack={handleInferenceBack}
        />
      {:else if currentStep === 1 && selectedMethod === 'git'}
        <GitImportStep1 onComplete={handleGitImportComplete} onBack={handleStep1Back} />
      {:else if currentStep === 2 && selectedMethod === 'git'}
        <GitImportStep2 {volumes} onNext={handleGitStep2Next} onBack={handleGitStep2Back} />
      {:else if currentStep === 1 && selectedMethod === 'portainer'}
        <PortainerStep1 initialUrl={portainerUrl} initialApiKey={portainerApiKey} onNext={handlePortainerStep1Next} onBack={handleStep1Back} />
      {:else if currentStep === 2 && selectedMethod === 'portainer'}
        <PortainerStep2
          portainerUrl={portainerUrl}
          apiKey={portainerApiKey}
          endpointId={portainerEndpointId}
          savedSourceId={portainerSavedSourceId || undefined}
          onComplete={handlePortainerStep2Complete}
          onBack={() => { currentStep = 1; }}
        />
      {:else if (currentStep === 2 && selectedMethod === 'paste') || (currentStep === 3 && (selectedMethod === 'git' || selectedMethod === 'portainer'))}
        <WizardStep2
          {projectId}
          onComplete={handleStep2Complete}
          onBack={handleStep2Back}
        />
      {:else}
        <WizardStep3
          {projectId}
          {projectName}
          {volumes}
          {backendType}
          onBack={handleStep3Back}
        />
      {/if}
    </div>
  </div>
</div>
