'use client';

import { LoopEditorForm } from '@/components/config/LoopEditorForm';
import { LoopTaskList } from '@/components/config/LoopTaskList';
import { useTaskLogs } from '@/hooks/config/useTaskLogs';
import { useLoopSettingsState } from '@/hooks/config/useLoopSettingsState';
import { ignorePromise } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { useConfigPresets } from '@/hooks/useConfigPresets';
import type { useConfigTasks } from '@/hooks/useConfigTasks';
import type { BridgeConfig } from '@/lib/types';

interface LoopSettingsSectionProps {
  config: BridgeConfig | null;
  tasksState: ReturnType<typeof useConfigTasks>;
  presetsState: ReturnType<typeof useConfigPresets>;
}

export function LoopSettingsSection(props: LoopSettingsSectionProps) {
  const { copy } = useWebLocale();
  const logs = useTaskLogs(copy.settings.tasksLogsEmpty);
  const state = useLoopSettingsState(toLoopStateOptions(props, copy));

  const handleRun = async (id: string) => {
    await state.runByID(id);
    if (logs.taskID === id) {
      await logs.open(id);
    }
  };

  if (state.editorOpen) {
    return (
      <LoopEditorForm
        mode={state.editorMode}
        editor={state.editor}
        presets={state.presets}
        controlsDisabled={state.controlsDisabled}
        saving={state.submitting}
        onChangeEditor={state.updateEditor}
        onSubmit={state.submitEditor}
        onCancel={state.cancelEditor}
      />
    );
  }

  return (
    <section data-testid="loop-settings-section">
      <LoopSettingsHeader state={state} />
      <LoopSettingsError error={state.error} />
      <LoopTaskList
        loops={state.loops}
        presets={state.presets}
        loading={state.loading}
        controlsDisabled={state.controlsDisabled}
        runningLoopID={state.runningLoopID}
        onEdit={state.editLoop}
        onOpenLogs={logs.open}
        onRun={handleRun}
        onSetEnabled={state.setEnabledByID}
        onDelete={state.deleteByID}
        logsTaskID={logs.taskID}
        logsData={logs.entries}
        logsLoading={logs.loading}
        logsError={logs.error}
        onCloseLogs={logs.close}
      />
    </section>
  );
}

function toLoopStateOptions(
  props: LoopSettingsSectionProps,
  copy: ReturnType<typeof useWebLocale>['copy'],
) {
  return {
    tasks: props.tasksState.tasks,
    tasksLoading: props.tasksState.tasksLoading,
    taskSaving: props.tasksState.taskSaving,
    taskError: props.tasksState.taskError,
    presets: props.presetsState.presets,
    presetsLoading: props.presetsState.presetsLoading,
    presetError: props.presetsState.presetError,
    config: props.config,
    copy,
    refreshTasks: props.tasksState.refreshTasks,
    setTaskEnabled: props.tasksState.setTaskEnabled,
    runTaskNowByID: props.tasksState.runTaskNowByID,
    deleteTaskByID: props.tasksState.deleteTaskByID,
  };
}

function LoopSettingsHeader(props: { state: ReturnType<typeof useLoopSettingsState> }) {
  const { copy } = useWebLocale();
  const { state } = props;

  return (
    <header className="mb-10 flex flex-wrap items-end justify-between gap-3">
      <div>
        <h1 className="mb-2 text-[28px] font-semibold tracking-tight text-[#111111]">{copy.settings.loopTitle}</h1>
        <p className="text-[14px] text-[#737373]">{copy.settings.loopDescription}</p>
      </div>
      <LoopSettingsHeaderActions state={state} />
    </header>
  );
}

function LoopSettingsHeaderActions(props: { state: ReturnType<typeof useLoopSettingsState> }) {
  const { copy } = useWebLocale();
  const { state } = props;

  return (
    <div className="flex items-center gap-2">
      <button type="button" disabled={state.controlsDisabled} onClick={() => ignorePromise(state.refresh())} className="rounded-full border border-[#E5E5E5] px-4 py-2 text-[13px] font-medium text-[#111111] transition-colors hover:bg-[#F5F5F5] disabled:cursor-not-allowed disabled:opacity-50">
        {copy.settings.refresh}
      </button>
      <button type="button" disabled={state.controlsDisabled} onClick={state.startCreate} className="inline-flex items-center rounded-full bg-[#111111] px-4 py-2 text-[13px] font-medium text-white transition-colors hover:bg-[#333333] disabled:cursor-not-allowed disabled:opacity-50">
        {copy.settings.loopNew}
      </button>
    </div>
  );
}

function LoopSettingsError(props: { error: string }) {
  if (!props.error) {
    return null;
  }

  return (
    <div className="mb-4 rounded-[16px] border border-red-200 bg-red-50 px-6 py-4 text-[13px] text-red-700">
      {props.error}
    </div>
  );
}
