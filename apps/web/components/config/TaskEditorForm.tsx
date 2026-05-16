'use client';

import type { FormEvent } from 'react';
import { SoftDropdownSelect } from '@/components/SoftDropdownSelect';
import { TaskFormField } from '@/components/config/TaskFormField';
import { TaskRuntimeSection } from '@/components/config/TaskRuntimeSection';
import { ignorePromise } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { TaskEditorMode, TaskEditorState, TaskScheduleMode } from '@/lib/configTasks';
import type { PresetPayload } from '@/lib/types';

interface TaskEditorFormProps {
  editorMode: TaskEditorMode;
  editor: TaskEditorState;
  presets: PresetPayload[];
  sessionOptions: string[];
  controlsDisabled: boolean;
  saving: boolean;
  onChangeEditor: (patch: Partial<TaskEditorState>) => void;
  onSubmit: () => Promise<void>;
  onCancelEditing: () => void;
}

export function TaskEditorForm(props: TaskEditorFormProps) {
  const { copy } = useWebLocale();
  const {
    editorMode,
    editor,
    presets,
    sessionOptions,
    controlsDisabled,
    saving,
    onChangeEditor,
    onSubmit,
    onCancelEditing,
  } = props;

  return (
    <form className="space-y-8" onSubmit={(event) => handleTaskSubmit(event, onSubmit)}>
      <TaskEditorHeader editorMode={editorMode} />
      <TaskEditorFields
        editor={editor}
        presets={presets}
        sessionOptions={sessionOptions}
        controlsDisabled={controlsDisabled}
        onChangeEditor={onChangeEditor}
      />
      <TaskEditorActions
        controlsDisabled={controlsDisabled}
        submitLabel={resolveSubmitLabel(editorMode, saving, copy)}
        onCancelEditing={onCancelEditing}
      />
    </form>
  );
}

function TaskEditorHeader(props: { editorMode: TaskEditorMode }) {
  const { copy } = useWebLocale();
  const { editorMode } = props;

  return (
    <div>
      <p className="mb-2 text-[11px] font-bold uppercase tracking-[0.15em] text-[#737373]">
        {editorMode === 'edit' ? copy.settings.taskEditorEditing : copy.settings.taskEditorNew}
      </p>
      <h3 className="text-[20px] font-medium text-[#111111]">
        {editorMode === 'edit' ? copy.settings.taskEditorEditTitle : copy.settings.taskEditorCreateTitle}
      </h3>
    </div>
  );
}

function TaskEditorFields(props: {
  editor: TaskEditorState;
  presets: PresetPayload[];
  sessionOptions: string[];
  controlsDisabled: boolean;
  onChangeEditor: (patch: Partial<TaskEditorState>) => void;
}) {
  const { editor, presets, sessionOptions, controlsDisabled, onChangeEditor } = props;

  return (
    <div className="rounded-[16px] border border-[#E5E5E5] bg-white p-6">
      <div className="space-y-5">
        <TaskMessageField editor={editor} controlsDisabled={controlsDisabled} onChangeEditor={onChangeEditor} />
        <TaskScheduleFields editor={editor} controlsDisabled={controlsDisabled} onChangeEditor={onChangeEditor} />
        <TaskSessionField editor={editor} controlsDisabled={controlsDisabled} onChangeEditor={onChangeEditor} />
        <TaskRuntimeSection
          editor={editor}
          presets={presets}
          controlsDisabled={controlsDisabled}
          onChangeEditor={onChangeEditor}
        />
        <TaskSessionOptions sessionOptions={sessionOptions} />
      </div>
    </div>
  );
}

function TaskMessageField(props: {
  editor: TaskEditorState;
  controlsDisabled: boolean;
  onChangeEditor: (patch: Partial<TaskEditorState>) => void;
}) {
  const { copy } = useWebLocale();
  const { editor, controlsDisabled, onChangeEditor } = props;

  return (
    <TaskFormField label={copy.settings.taskEditorMessageLabel} description={copy.settings.taskEditorMessageDescription}>
      <textarea
        value={editor.message}
        disabled={controlsDisabled}
        rows={4}
        onChange={(event) => onChangeEditor({ message: event.target.value })}
        className="w-full rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-2.5 text-[14px] text-[#111111] placeholder-[#A3A3A3] transition-colors focus:border-[#111111] focus:outline-none"
      />
    </TaskFormField>
  );
}

function TaskScheduleFields(props: {
  editor: TaskEditorState;
  controlsDisabled: boolean;
  onChangeEditor: (patch: Partial<TaskEditorState>) => void;
}) {
  const { copy } = useWebLocale();
  const { editor, controlsDisabled, onChangeEditor } = props;
  const options: Array<{ value: TaskScheduleMode; label: string }> = [
    { value: 'interval', label: copy.settings.taskEditorModeInterval },
    { value: 'cron', label: copy.settings.taskEditorModeCron },
  ];

  return (
    <>
      <TaskFormField label={copy.settings.taskEditorScheduleModeLabel} description={copy.settings.taskEditorScheduleModeDescription}>
        <SoftDropdownSelect
          value={editor.scheduleMode}
          disabled={controlsDisabled}
          options={options}
          onChange={(value) => onChangeEditor({ scheduleMode: value as TaskScheduleMode })}
        />
      </TaskFormField>
      <TaskScheduleValueField editor={editor} controlsDisabled={controlsDisabled} onChangeEditor={onChangeEditor} />
    </>
  );
}

function TaskScheduleValueField(props: {
  editor: TaskEditorState;
  controlsDisabled: boolean;
  onChangeEditor: (patch: Partial<TaskEditorState>) => void;
}) {
  const { copy } = useWebLocale();
  const { editor, controlsDisabled, onChangeEditor } = props;
  const cronMode = editor.scheduleMode === 'cron';

  return (
    <TaskFormField
      label={cronMode ? copy.settings.taskEditorCronLabel : copy.settings.taskEditorIntervalLabel}
      description={cronMode ? copy.settings.taskEditorCronDescription : copy.settings.taskEditorIntervalDescription}
    >
      <input
        value={cronMode ? editor.cronExpr : editor.intervalSeconds}
        disabled={controlsDisabled}
        onChange={(event) => onChangeEditor(cronMode ? { cronExpr: event.target.value } : { intervalSeconds: event.target.value })}
        className="w-full rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-2.5 font-mono text-[14px] text-[#111111] placeholder-[#A3A3A3] transition-colors focus:border-[#111111] focus:outline-none"
      />
    </TaskFormField>
  );
}

function TaskSessionField(props: {
  editor: TaskEditorState;
  controlsDisabled: boolean;
  onChangeEditor: (patch: Partial<TaskEditorState>) => void;
}) {
  const { copy } = useWebLocale();
  const { editor, controlsDisabled, onChangeEditor } = props;

  return (
    <TaskFormField label={copy.settings.taskEditorSessionLabel} description={copy.settings.taskEditorSessionDescription}>
      <input
        list="task-session-options"
        value={editor.sessionId}
        disabled={controlsDisabled}
        onChange={(event) => onChangeEditor({ sessionId: event.target.value })}
        className="w-full rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-2.5 font-mono text-[14px] text-[#111111] placeholder-[#A3A3A3] transition-colors focus:border-[#111111] focus:outline-none"
      />
    </TaskFormField>
  );
}

function TaskSessionOptions(props: { sessionOptions: string[] }) {
  return (
    <datalist id="task-session-options">
      {props.sessionOptions.map((sessionID) => (
        <option key={sessionID} value={sessionID} />
      ))}
    </datalist>
  );
}

function TaskEditorActions(props: {
  controlsDisabled: boolean;
  submitLabel: string;
  onCancelEditing: () => void;
}) {
  const { copy } = useWebLocale();
  const { controlsDisabled, submitLabel, onCancelEditing } = props;

  return (
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

function handleTaskSubmit(event: FormEvent<HTMLFormElement>, onSubmit: () => Promise<void>) {
  event.preventDefault();
  ignorePromise(onSubmit());
}
