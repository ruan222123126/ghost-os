import { ComingSoonPanel, type SettingsTab } from '@/components/config/ConfigPanelNavigation';
import { OrchestrationSettingsSection } from '@/components/config/OrchestrationSettingsSection';
import { PresetSettingsSection } from '@/components/config/PresetSettingsSection';
import { PromptsLibrarySettingsSection } from '@/components/config/PromptsLibrarySettingsSection';
import { PromptsPreviewSettingsSection } from '@/components/config/PromptsPreviewSettingsSection';
import { ProviderSettingsSection } from '@/components/config/ProviderSettingsSection';
import { RelaySettingsSection } from '@/components/config/RelaySettingsSection';
import { RuntimeSettingsSection } from '@/components/config/RuntimeSettingsSection';
import { SkillSettingsSection } from '@/components/config/SkillSettingsSection';
import { TaskSettingsSection } from '@/components/config/TaskSettingsSection';
import { ToolSettingsSection } from '@/components/config/ToolSettingsSection';
import type { useConfigProviders } from '@/hooks/useConfigProviders';
import type { useConfigPresets } from '@/hooks/useConfigPresets';
import type { useConfigPrompts } from '@/hooks/useConfigPrompts';
import type { useConfigSkills } from '@/hooks/useConfigSkills';
import type { useConfigTasks } from '@/hooks/useConfigTasks';
import type { useConfigTools } from '@/hooks/useConfigTools';
import type { BridgeConfig, ConfigUpdate, WorkflowTaskPayload } from '@/lib/types';

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
        providers={providersState.providers}
        activeProvider={providersState.activeProvider}
        loading={providersState.providersLoading}
        saving={providersState.providerSaving}
        editorMode={providersState.editorMode}
        editor={providersState.editor}
        onRefresh={providersState.refreshProviders}
        onBeginCreate={providersState.beginCreateProvider}
        onEdit={providersState.editProvider}
        onChangeEditor={providersState.updateEditor}
        onSelectProviderType={providersState.selectProviderType}
        onSubmit={providersState.submitProvider}
        onActivate={providersState.activateProvider}
        onDelete={providersState.deleteProviderByName}
        onCancelEditing={providersState.cancelEditing}
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
        tasks={tasksState.tasks}
        loading={tasksState.tasksLoading}
        saving={tasksState.taskSaving}
        editorMode={tasksState.editorMode}
        editor={tasksState.editor}
        onRefresh={tasksState.refreshTasks}
        onBeginCreateTextTask={tasksState.beginCreateTextTask}
        onEditTextTask={tasksState.editTask}
        onOpenWorkflowCreate={onOpenWorkflowCreate}
        onOpenWorkflowEdit={onOpenWorkflowEdit}
        onChangeEditor={tasksState.updateEditor}
        onSubmit={tasksState.submitTask}
        onSetEnabled={tasksState.setTaskEnabled}
        onRunNow={tasksState.runTaskNowByID}
        onDelete={tasksState.deleteTaskByID}
        onCancelEditing={tasksState.cancelEditing}
      />
    );
  }

  if (activeTab === 'relay') {
    return (
      <RelaySettingsSection
        loading={loading}
        saving={saving}
        config={config}
        onSave={onSave}
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
