'use client';

import type { FC } from 'react';
import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  SettingsNavigation,
  type SettingsTab,
} from '@/components/config/ConfigPanelNavigation';
import { CloseButton } from '@/components/CloseButton';
import { ConfigPanelSectionContent } from '@/components/config/ConfigPanelSectionContent';
import { resolveConfigPanelTabError } from '@/components/config/configPanelTabError';
import { useConfigProviders } from '@/hooks/useConfigProviders';
import { useConfigPresets } from '@/hooks/useConfigPresets';
import { useConfigPrompts } from '@/hooks/useConfigPrompts';
import { useConfigSkills } from '@/hooks/useConfigSkills';
import { useConfigTasks } from '@/hooks/useConfigTasks';
import { useConfigTools } from '@/hooks/useConfigTools';
import { useWebLocale } from '@/lib/i18n/provider';
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
  const { copy } = useWebLocale();
  const [activeTab, setActiveTab] = useState<SettingsTab>('provider');
  const providersState = useConfigProviders({
    open,
    onReloadConfig: onReload,
    onActivateRuntimeConfig: onSave,
    modelSelectionEnabled: config?.model_selection_enabled ?? true,
  });
  const presetsState = useConfigPresets({ open });
  const promptsState = useConfigPrompts({ open });
  const skillsState = useConfigSkills({ open });
  const tasksState = useConfigTasks({ open, config });
  const toolsState = useConfigTools({ open });

  useEffect(() => {
    if (open) {
      setActiveTab(initialTab);
    }
  }, [initialTab, open]);

  const tabError = useMemo(() => {
    return resolveConfigPanelTabError({
      activeTab,
      generalError: error,
      providerError: providersState.providerError,
      presetError: presetsState.presetError,
      promptError: promptsState.promptError,
      taskError: tasksState.taskError,
      skillError: skillsState.skillError,
      toolError: toolsState.toolError,
    });
  }, [
    activeTab,
    error,
    providersState.providerError,
    presetsState.presetError,
    promptsState.promptError,
    tasksState.taskError,
    skillsState.skillError,
    toolsState.toolError,
  ]);

  const handleSelectTab = useCallback((tab: SettingsTab) => {
    setActiveTab(tab);
    providersState.cancelEditing();
    tasksState.cancelEditing();
  }, [providersState, tasksState]);

  if (!open) {
    return null;
  }

  return (
    <div className="fixed inset-0 z-40 flex items-center justify-center p-4 sm:p-6 md:p-12" role="dialog" aria-modal="true" aria-labelledby="settings-title">
      <button type="button" className="absolute inset-0 bg-black/40 backdrop-blur-[2px]" onClick={onClose} aria-label={copy.settings.closeSettingsAria} />

      <section
        data-testid="config-panel-shell"
        className="relative z-10 flex h-[85vh] max-h-[800px] w-full max-w-[1000px] overflow-hidden rounded-[24px] border border-[#E5E5E5] bg-white shadow-2xl"
      >
        <CloseButton
          onClick={onClose}
          className="absolute right-6 top-6 z-20"
          aria-label={copy.settings.closeSettingsAria}
        />

        <SettingsNavigation activeTab={activeTab} onSelectTab={handleSelectTab} />

        <div className="relative flex-1 overflow-y-auto overscroll-contain touch-pan-y bg-white">
          <div
            data-testid="config-panel-content"
            className="mx-auto w-full max-w-2xl px-8 pb-24 pt-12 md:px-12"
          >
            {tabError ? (
              <div className="mb-4 rounded-[12px] border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
                {tabError}
              </div>
            ) : null}

            <ConfigPanelSectionContent
              activeTab={activeTab}
              loading={loading}
              saving={saving}
              config={config}
              onSave={onSave}
              onRefreshConfig={onReload}
              onOpenWorkflowCreate={onOpenWorkflowCreate}
              onOpenWorkflowEdit={onOpenWorkflowEdit}
              providersState={providersState}
              presetsState={presetsState}
              promptsState={promptsState}
              skillsState={skillsState}
              tasksState={tasksState}
              toolsState={toolsState}
            />
          </div>
        </div>
      </section>
    </div>
  );
};
