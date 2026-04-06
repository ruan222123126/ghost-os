'use client';

import type { FC } from 'react';
import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  CloseIcon,
  ComingSoonPanel,
  SettingsNavigation,
  type SettingsTab,
} from '@/components/config/ConfigPanelNavigation';
import { ProviderSettingsSection } from '@/components/config/ProviderSettingsSection';
import { RuntimeSettingsSection } from '@/components/config/RuntimeSettingsSection';
import { TaskSettingsSection } from '@/components/config/TaskSettingsSection';
import { useConfigProviders } from '@/hooks/useConfigProviders';
import { useConfigTasks } from '@/hooks/useConfigTasks';
import type { BridgeConfig, ConfigUpdate, WorkflowTaskPayload } from '@/lib/types';

interface ConfigPanelProps {
  open: boolean;
  initialTab?: SettingsTab;
  loading: boolean;
  saving: boolean;
  config: BridgeConfig | null;
  error: string;
  onClose: () => void;
  onOpenWorkflowCreate: () => void;
  onOpenWorkflowEdit: (task: WorkflowTaskPayload) => void;
  onSave: (update: ConfigUpdate) => Promise<boolean>;
  onReload: () => Promise<void>;
}

export type { SettingsTab };

export const ConfigPanel: FC<ConfigPanelProps> = ({
  open,
  initialTab = 'provider',
  loading,
  saving,
  config,
  error,
  onClose,
  onOpenWorkflowCreate,
  onOpenWorkflowEdit,
  onSave,
  onReload,
}) => {
  const [activeTab, setActiveTab] = useState<SettingsTab>('provider');
  const {
    providers,
    activeProvider,
    providersLoading,
    providerSaving,
    providerError,
    editorMode,
    editor,
    refreshProviders,
    beginCreateProvider,
    editProvider,
    updateEditor,
    selectProviderType,
    submitProvider,
    activateProvider,
    deleteProviderByName,
    cancelEditing,
  } = useConfigProviders({
    open,
    onReloadConfig: onReload,
  });
  const {
    tasks,
    tasksLoading,
    taskSaving,
    taskError,
    editorMode: taskEditorMode,
    editor: taskEditor,
    refreshTasks,
    beginCreateTextTask,
    editTask: editTextTask,
    updateEditor: updateTaskEditor,
    submitTask,
    setTaskEnabled,
    runTaskNowByID,
    deleteTaskByID,
    cancelEditing: cancelTaskEditing,
  } = useConfigTasks({
    open,
  });

  useEffect(() => {
    if (!open) {
      return;
    }
    setActiveTab(initialTab);
  }, [initialTab, open]);

  const tabError = useMemo(() => {
    if (activeTab === 'provider') {
      return providerError;
    }
    if (activeTab === 'tasks') {
      return taskError;
    }
    if (activeTab === 'general') {
      return error;
    }
    return '';
  }, [activeTab, error, providerError, taskError]);

  const handleSelectTab = useCallback((tab: SettingsTab) => {
    setActiveTab(tab);
    cancelEditing();
    cancelTaskEditing();
  }, [cancelEditing, cancelTaskEditing]);

  if (!open) {
    return null;
  }

  return (
    <div className="fixed inset-0 z-40 flex items-center justify-center p-4 sm:p-6 md:p-12" role="dialog" aria-modal="true" aria-labelledby="settings-title">
      <button type="button" className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={onClose} aria-label="Close settings" />

      <section className="relative z-10 flex h-[85vh] max-h-[800px] w-full max-w-[1000px] overflow-hidden rounded-[24px] border border-[#E5E5E5] bg-white shadow-2xl">
        <button
          type="button"
          onClick={onClose}
          className="absolute right-6 top-6 z-20 rounded-full bg-[#F5F5F5] p-2 text-[#737373] transition-colors hover:text-[#111111]"
          aria-label="Close settings"
        >
          <CloseIcon />
        </button>

        <SettingsNavigation activeTab={activeTab} onSelectTab={handleSelectTab} />

        <div className="relative flex-1 overflow-y-auto bg-white">
          <div className="mx-auto max-w-2xl px-12 pb-24 pt-12">
            {tabError ? (
              <div className="mb-4 rounded-[12px] border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
                {tabError}
              </div>
            ) : null}

            {activeTab === 'provider' ? (
              <ProviderSettingsSection
                providers={providers}
                activeProvider={activeProvider}
                loading={providersLoading}
                saving={providerSaving}
                editorMode={editorMode}
                editor={editor}
                onRefresh={refreshProviders}
                onBeginCreate={beginCreateProvider}
                onEdit={editProvider}
                onChangeEditor={updateEditor}
                onSelectProviderType={selectProviderType}
                onSubmit={submitProvider}
                onActivate={activateProvider}
                onDelete={deleteProviderByName}
                onCancelEditing={cancelEditing}
              />
            ) : null}

            {activeTab === 'general' ? (
              <RuntimeSettingsSection
                loading={loading}
                saving={saving}
                config={config}
                onSave={onSave}
                onReload={onReload}
              />
            ) : null}

            {activeTab === 'tasks' ? (
              <TaskSettingsSection
                tasks={tasks}
                loading={tasksLoading}
                saving={taskSaving}
                editorMode={taskEditorMode}
                editor={taskEditor}
                onRefresh={refreshTasks}
                onBeginCreateTextTask={beginCreateTextTask}
                onEditTextTask={editTextTask}
                onOpenWorkflowCreate={onOpenWorkflowCreate}
                onOpenWorkflowEdit={onOpenWorkflowEdit}
                onChangeEditor={updateTaskEditor}
                onSubmit={submitTask}
                onSetEnabled={setTaskEnabled}
                onRunNow={runTaskNowByID}
                onDelete={deleteTaskByID}
                onCancelEditing={cancelTaskEditing}
              />
            ) : null}

            {activeTab !== 'provider' && activeTab !== 'general' && activeTab !== 'tasks' ? (
              <ComingSoonPanel tab={activeTab} />
            ) : null}
          </div>
        </div>
      </section>
    </div>
  );
};
