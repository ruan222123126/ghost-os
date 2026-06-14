import nextDynamic from 'next/dynamic';
import { ComingSoonPanel, type SettingsTab } from '@/components/config/ConfigPanelNavigation';
import type { useConfigProviders } from '@/hooks/useConfigProviders';
import type { useConfigPresets } from '@/hooks/useConfigPresets';
import type { useConfigPrompts } from '@/hooks/useConfigPrompts';
import type { useConfigSkills } from '@/hooks/useConfigSkills';
import type { useConfigTasks } from '@/hooks/useConfigTasks';
import type { useConfigTools } from '@/hooks/useConfigTools';
import type { BridgeConfig, ConfigUpdate, WorkflowTaskPayload } from '@/lib/types';

const LoopSettingsSection = nextDynamic(
  () => import('@/components/config/LoopSettingsSection').then((mod) => mod.LoopSettingsSection),
  { loading: ConfigSectionLoadingBar, ssr: false },
);
const OrchestrationSettingsSection = nextDynamic(
  () => import('@/components/config/OrchestrationSettingsSection').then((mod) => mod.OrchestrationSettingsSection),
  { loading: ConfigSectionLoadingBar, ssr: false },
);
const PresetSettingsSection = nextDynamic(
  () => import('@/components/config/PresetSettingsSection').then((mod) => mod.PresetSettingsSection),
  { loading: ConfigSectionLoadingBar, ssr: false },
);
const PromptsLibrarySettingsSection = nextDynamic(
  () => import('@/components/config/PromptsLibrarySettingsSection').then((mod) => mod.PromptsLibrarySettingsSection),
  { loading: ConfigSectionLoadingBar, ssr: false },
);
const PromptsPreviewSettingsSection = nextDynamic(
  () => import('@/components/config/PromptsPreviewSettingsSection').then((mod) => mod.PromptsPreviewSettingsSection),
  { loading: ConfigSectionLoadingBar, ssr: false },
);
const ProviderSettingsSection = nextDynamic(
  () => import('@/components/config/ProviderSettingsSection').then((mod) => mod.ProviderSettingsSection),
  { loading: ConfigSectionLoadingBar, ssr: false },
);
const RuntimeSettingsSection = nextDynamic(
  () => import('@/components/config/RuntimeSettingsSection').then((mod) => mod.RuntimeSettingsSection),
  { loading: ConfigSectionLoadingBar, ssr: false },
);
const SkillSettingsSection = nextDynamic(
  () => import('@/components/config/SkillSettingsSection').then((mod) => mod.SkillSettingsSection),
  { loading: ConfigSectionLoadingBar, ssr: false },
);
const TaskSettingsSection = nextDynamic(
  () => import('@/components/config/TaskSettingsSection').then((mod) => mod.TaskSettingsSection),
  { loading: ConfigSectionLoadingBar, ssr: false },
);
const ToolSettingsSection = nextDynamic(
  () => import('@/components/config/ToolSettingsSection').then((mod) => mod.ToolSettingsSection),
  { loading: ConfigSectionLoadingBar, ssr: false },
);

function ConfigSectionLoadingBar() {
  return <TopLoadingBar isVisible />;
}

function TopLoadingBar({ isVisible = false }: { isVisible?: boolean }) {
  return (
    <>
      <style>{`
        @keyframes high-perf-slide {
          0% {
            transform: translateX(-10%) scaleX(0.1);
          }
          50% {
            transform: translateX(30%) scaleX(0.5);
          }
          100% {
            transform: translateX(100%) scaleX(0.1);
          }
        }

        .animate-top-loading-bar {
          animation: high-perf-slide 1.2s linear infinite;
          transform-origin: left center;
        }
      `}</style>

      <div
        data-testid="config-section-top-loading-bar"
        className={`fixed top-0 left-0 z-[9999] h-[3px] w-full pointer-events-none transition-opacity duration-500 ease-in-out ${
          isVisible ? 'opacity-100' : 'opacity-0'
        }`}
      >
        <div className="h-full w-full animate-top-loading-bar bg-black shadow-[0_0_8px_rgba(0,0,0,0.3)]" />
      </div>
    </>
  );
}

interface ConfigPanelSectionContentProps {
  activeTab: SettingsTab;
  loading: boolean;
  saving: boolean;
  config: BridgeConfig | null;
  onSave: (update: ConfigUpdate) => Promise<boolean>;
  onRefreshConfig: () => Promise<void>;
  onOpenWorkflowCreate: () => void;
  onOpenWorkflowEdit: (task: WorkflowTaskPayload) => void;
  providersState: ReturnType<typeof useConfigProviders>;
  presetsState: ReturnType<typeof useConfigPresets>;
  promptsState: ReturnType<typeof useConfigPrompts>;
  skillsState: ReturnType<typeof useConfigSkills>;
  tasksState: ReturnType<typeof useConfigTasks>;
  toolsState: ReturnType<typeof useConfigTools>;
}

export function ConfigPanelSectionContent(props: ConfigPanelSectionContentProps) {
  const {
    activeTab,
    loading,
    saving,
    config,
    onSave,
    onRefreshConfig,
    onOpenWorkflowCreate,
    onOpenWorkflowEdit,
    providersState,
    presetsState,
    promptsState,
    skillsState,
    tasksState,
    toolsState,
  } = props;

  if (activeTab === 'provider') {
    return (
      <ProviderSettingsSection
        machine={providersState}
      />
    );
  }

  if (activeTab === 'general') {
    return (
      <RuntimeSettingsSection
        loading={loading}
        saving={saving}
        config={config}
        onSave={onSave}
      />
    );
  }

  if (activeTab === 'tasks') {
    return (
      <TaskSettingsSection
        machine={tasksState}
        presets={presetsState.presets ?? []}
        onOpenWorkflowCreate={onOpenWorkflowCreate}
        onOpenWorkflowEdit={onOpenWorkflowEdit}
      />
    );
  }

  if (activeTab === 'relay') {
    return (
      <LoopSettingsSection
        config={config}
        tasksState={tasksState}
        presetsState={presetsState}
      />
    );
  }

  if (activeTab === 'orchestration') {
    return <OrchestrationSettingsSection />;
  }

  if (activeTab === 'skills') {
    return (
      <SkillSettingsSection
        skills={skillsState.skills}
        loading={skillsState.skillsLoading}
        saving={skillsState.skillSaving}
        onRefresh={skillsState.refreshSkills}
        onUpdate={skillsState.updateSkillByID}
        onDelete={skillsState.deleteSkillByID}
      />
    );
  }

  if (activeTab === 'tools') {
    return (
      <ToolSettingsSection
        tools={toolsState.tools}
        loading={toolsState.toolsLoading}
        saving={toolsState.toolSaving}
        onRefresh={toolsState.refreshTools}
        onUpdate={toolsState.updateToolByName}
      />
    );
  }

  if (activeTab === 'presets') {
    return (
      <PresetSettingsSection
        config={config}
        presets={presetsState.presets}
        loading={presetsState.presetsLoading}
        saving={presetsState.presetSaving}
        tools={toolsState.tools}
        toolsLoading={toolsState.toolsLoading}
        prompts={promptsState.prompts}
        promptsLoading={promptsState.promptsLoading}
        onRefresh={presetsState.refreshPresets}
        onCreatePreset={presetsState.createPreset}
        onUpdatePreset={presetsState.updatePresetByID}
        onDeletePreset={presetsState.deletePresetByID}
        onActivatePreset={async (id: string) => {
          await presetsState.activatePresetByID(id);
          await Promise.all([
            onRefreshConfig(),
            toolsState.refreshTools(),
            promptsState.refreshPrompts(),
          ]);
        }}
      />
    );
  }

  if (activeTab === 'prompts_library') {
    return (
      <PromptsLibrarySettingsSection
        prompts={promptsState.prompts}
        loading={promptsState.promptsLoading}
        saving={promptsState.promptSaving}
        onRefresh={promptsState.refreshPrompts}
        onSavePromptLibrary={promptsState.savePromptLibrary}
      />
    );
  }

  if (activeTab === 'prompts_preview') {
    return (
      <PromptsPreviewSettingsSection
        prompts={promptsState.prompts}
        loading={promptsState.promptsLoading}
        saving={promptsState.promptSaving}
        onRefresh={promptsState.refreshPrompts}
      />
    );
  }

  return <ComingSoonPanel tab={activeTab} />;
}
