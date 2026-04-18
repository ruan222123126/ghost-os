'use client';

import type { FormEvent, ReactNode } from 'react';
import { ignorePromise } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type {
  TaskEditorMode,
  TaskEditorState,
  TaskScheduleMode,
} from '@/lib/configTasks';

interface TaskEditorFormProps {
  editorMode: TaskEditorMode;
  editor: TaskEditorState;
  sessionOptions: string[];
  controlsDisabled: boolean;
  saving: boolean;
  onChangeEditor: (patch: Partial<TaskEditorState>) => void;
  onSubmit: () => Promise<void>;
  onCancelEditing: () => void;
}

interface FormFieldProps {
  label: string;
  description: string;
  children: ReactNode;
}

export function TaskEditorForm(props: TaskEditorFormProps) {
  const { copy } = useWebLocale();
  const {
    editorMode,
    editor,
    sessionOptions,
    controlsDisabled,
    saving,
    onChangeEditor,
    onSubmit,
    onCancelEditing,
  } = props;
  const submitLabel = resolveSubmitLabel(editorMode, saving, copy);
  const scheduleModeOptions: Array<{ value: TaskScheduleMode; label: string }> = [
    { value: 'interval', label: copy.settings.taskEditorModeInterval },
    { value: 'cron', label: copy.settings.taskEditorModeCron },
  ];

  return (
    <form className="space-y-8" onSubmit={(event) => handleTaskSubmit(event, onSubmit)}>
      <div>
        <p className="mb-2 text-[11px] font-bold uppercase tracking-[0.15em] text-[#737373]">
          {editorMode === 'edit' ? copy.settings.taskEditorEditing : copy.settings.taskEditorNew}
        </p>
        <h3 className="text-[20px] font-medium text-[#111111]">
          {editorMode === 'edit' ? copy.settings.taskEditorEditTitle : copy.settings.taskEditorCreateTitle}
        </h3>
      </div>

      <div className="rounded-[16px] border border-[#E5E5E5] bg-white p-6">
        <div className="space-y-5">
          <FormField label={copy.settings.taskEditorMessageLabel} description={copy.settings.taskEditorMessageDescription}>
            <textarea
              value={editor.message}
              disabled={controlsDisabled}
              rows={4}
              onChange={(event) => onChangeEditor({ message: event.target.value })}
              className="w-full rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-2.5 text-[14px] text-[#111111] placeholder-[#A3A3A3] transition-colors focus:border-[#111111] focus:outline-none"
            />
          </FormField>

          <FormField label={copy.settings.taskEditorScheduleModeLabel} description={copy.settings.taskEditorScheduleModeDescription}>
            <select
              value={editor.scheduleMode}
              disabled={controlsDisabled}
              onChange={(event) => onChangeEditor({ scheduleMode: event.target.value as TaskScheduleMode })}
              className="w-full rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-2.5 text-[14px] text-[#111111] transition-colors focus:border-[#111111] focus:outline-none"
            >
              {scheduleModeOptions.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          </FormField>

          {editor.scheduleMode === 'interval' ? (
            <FormField
              label={copy.settings.taskEditorIntervalLabel}
              description={copy.settings.taskEditorIntervalDescription}
            >
              <input
                value={editor.intervalSeconds}
                disabled={controlsDisabled}
                onChange={(event) => onChangeEditor({ intervalSeconds: event.target.value })}
                className="w-full rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-2.5 font-mono text-[14px] text-[#111111] placeholder-[#A3A3A3] transition-colors focus:border-[#111111] focus:outline-none"
              />
            </FormField>
          ) : (
            <FormField label={copy.settings.taskEditorCronLabel} description={copy.settings.taskEditorCronDescription}>
              <input
                value={editor.cronExpr}
                disabled={controlsDisabled}
                onChange={(event) => onChangeEditor({ cronExpr: event.target.value })}
                className="w-full rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-2.5 font-mono text-[14px] text-[#111111] placeholder-[#A3A3A3] transition-colors focus:border-[#111111] focus:outline-none"
              />
            </FormField>
          )}

          <FormField
            label={copy.settings.taskEditorSessionLabel}
            description={copy.settings.taskEditorSessionDescription}
          >
            <input
              list="task-session-options"
              value={editor.sessionId}
              disabled={controlsDisabled}
              onChange={(event) => onChangeEditor({ sessionId: event.target.value })}
              className="w-full rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-2.5 font-mono text-[14px] text-[#111111] placeholder-[#A3A3A3] transition-colors focus:border-[#111111] focus:outline-none"
            />
          </FormField>

          <details className="rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] p-4">
            <summary className="cursor-pointer list-none text-[13px] font-semibold text-[#111111]">
              <span className="mr-1 inline-block text-[#737373]">{'>'}</span>
              {copy.settings.taskEditorRuntimeSection}
            </summary>
            <div className="mt-4 space-y-5">
              <label className="flex items-center gap-2 text-[13px] font-medium text-[#111111]">
                <input
                  type="checkbox"
                  checked={editor.runtimeOverridesEnabled}
                  disabled={controlsDisabled}
                  onChange={(event) => onChangeEditor({ runtimeOverridesEnabled: event.target.checked })}
                  className="h-4 w-4 rounded border border-[#D4D4D4] accent-[#111111]"
                />
                {copy.settings.taskEditorRuntimeEnable}
              </label>

              <FormField
                label={copy.settings.taskEditorRuntimeModelLabel}
                description={copy.settings.taskEditorRuntimeModelDescription}
              >
                <input
                  value={editor.runtimeModel}
                  disabled={controlsDisabled || !editor.runtimeOverridesEnabled}
                  onChange={(event) => onChangeEditor({ runtimeModel: event.target.value })}
                  placeholder={copy.settings.taskEditorRuntimeModelPlaceholder}
                  className="w-full rounded-[12px] border border-[#E5E5E5] bg-white px-4 py-2.5 font-mono text-[14px] text-[#111111] placeholder-[#A3A3A3] transition-colors focus:border-[#111111] focus:outline-none disabled:opacity-60"
                />
              </FormField>

              <FormField
                label={copy.settings.taskEditorRuntimeToolsLabel}
                description={copy.settings.taskEditorRuntimeToolsDescription}
              >
                <textarea
                  value={editor.runtimeToolAllowlist}
                  disabled={controlsDisabled || !editor.runtimeOverridesEnabled}
                  rows={3}
                  onChange={(event) => onChangeEditor({ runtimeToolAllowlist: event.target.value })}
                  placeholder={copy.settings.taskEditorRuntimeToolsPlaceholder}
                  className="w-full rounded-[12px] border border-[#E5E5E5] bg-white px-4 py-2.5 font-mono text-[13px] text-[#111111] placeholder-[#A3A3A3] transition-colors focus:border-[#111111] focus:outline-none disabled:opacity-60"
                />
              </FormField>
            </div>
          </details>

          <datalist id="task-session-options">
            {sessionOptions.map((sessionID) => (
              <option key={sessionID} value={sessionID} />
            ))}
          </datalist>
        </div>
      </div>

      <div className="flex items-center justify-end gap-3">
        <button
          type="button"
          onClick={onCancelEditing}
          className="rounded-full border border-[#E5E5E5] px-6 py-2.5 text-[13px] font-medium text-[#111111] transition-colors hover:bg-[#F5F5F5]"
        >
          {copy.settings.cancel}
        </button>
        <button
          type="submit"
          disabled={controlsDisabled}
          className="rounded-full bg-[#111111] px-8 py-2.5 text-[13px] font-semibold text-white transition-colors hover:bg-[#333333] disabled:cursor-not-allowed disabled:opacity-50"
        >
          {submitLabel}
        </button>
      </div>
    </form>
  );
}

function resolveSubmitLabel(
  mode: TaskEditorMode,
  saving: boolean,
  copy: ReturnType<typeof useWebLocale>['copy'],
): string {
  if (saving) {
    return copy.settings.saving;
  }
  if (mode === 'edit') {
    return copy.settings.taskEditorSubmitSave;
  }

  return copy.settings.taskEditorSubmitCreate;
}

function FormField(props: FormFieldProps) {
  const { label, description, children } = props;

  return (
    <label className="block">
      <span className="mb-1 block text-[13px] font-medium text-[#111111]">{label}</span>
      <span className="mb-2 block text-[12px] text-[#737373]">{description}</span>
      {children}
    </label>
  );
}

function handleTaskSubmit(event: FormEvent<HTMLFormElement>, onSubmit: () => Promise<void>) {
  event.preventDefault();
  ignorePromise(onSubmit());
}
