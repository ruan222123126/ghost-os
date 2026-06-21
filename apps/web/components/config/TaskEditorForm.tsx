'use client';

import type { FormEvent } from 'react';
import { CloseButton } from '@/components/CloseButton';
import { SoftDropdownSelect } from '@/components/SoftDropdownSelect';
import { TaskFormField } from '@/components/config/TaskFormField';
import { TaskRuntimeSection } from '@/components/config/TaskRuntimeSection';
import { ignorePromise } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type {
  TaskEditorMode,
  TaskEditorState,
  TaskEditorType,
  TaskRelayStopPolicy,
  TaskScheduleMode,
} from '@/lib/configTasks';
import type { PresetPayload } from '@/lib/types';

interface TaskEditorFormProps {
  editorMode: TaskEditorMode;
  editor: TaskEditorState;
  presets: PresetPayload[];
  sessionOptions: string[];
  controlsDisabled: boolean;
  saving: boolean;
  onChangeEditor: (patch: Partial<TaskEditorState>) => void;
  onSelectTaskType: (taskType: TaskEditorType) => void;
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
    onSelectTaskType,
    onSubmit,
    onCancelEditing,
  } = props;

  return (
    <form className="space-y-8" onSubmit={(event) => handleTaskSubmit(event, onSubmit)}>
      <TaskEditorHeader editorMode={editorMode} editor={editor} />
      <TaskEditorFields
        editorMode={editorMode}
        editor={editor}
        presets={presets}
        sessionOptions={sessionOptions}
        controlsDisabled={controlsDisabled}
        onChangeEditor={onChangeEditor}
        onSelectTaskType={onSelectTaskType}
      />
      <TaskEditorActions
        controlsDisabled={controlsDisabled}
        submitLabel={resolveSubmitLabel(editorMode, saving, copy)}
        onCancelEditing={onCancelEditing}
      />
    </form>
  );
}

function TaskEditorHeader(props: {
  editorMode: TaskEditorMode;
  editor: TaskEditorState;
}) {
  const { copy } = useWebLocale();
  const { editorMode, editor } = props;

  return (
    <div>
      <p className="mb-2 text-[11px] font-bold uppercase tracking-[0.15em] text-[#737373]">
        {editorMode === 'edit' ? copy.settings.taskEditorEditing : copy.settings.taskEditorNew}
      </p>
      <h3 className="text-[20px] font-medium text-[#111111]">
        {resolveEditorTitle(editorMode, editor.taskType, copy)}
      </h3>
    </div>
  );
}

function resolveEditorTitle(
  editorMode: TaskEditorMode,
  taskType: TaskEditorType,
  copy: ReturnType<typeof useWebLocale>['copy'],
): string {
  if (editorMode === 'create') {
    return copy.settings.taskEditorCreateTitle;
  }
  if (taskType === 'loop') {
    return copy.settings.loopEditorEditTitle;
  }
  return copy.settings.taskEditorEditTitle;
}

function TaskEditorFields(props: {
  editorMode: TaskEditorMode;
  editor: TaskEditorState;
  presets: PresetPayload[];
  sessionOptions: string[];
  controlsDisabled: boolean;
  onChangeEditor: (patch: Partial<TaskEditorState>) => void;
  onSelectTaskType: (taskType: TaskEditorType) => void;
}) {
  const {
    editorMode,
    editor,
    presets,
    sessionOptions,
    controlsDisabled,
    onChangeEditor,
    onSelectTaskType,
  } = props;

  return (
    <div className="rounded-[16px] border border-[#E5E5E5] bg-white p-6">
      <div className="space-y-5">
        {editorMode === 'create' ? (
          <TaskTypeField
            editor={editor}
            controlsDisabled={controlsDisabled}
            onSelectTaskType={onSelectTaskType}
          />
        ) : null}
        <TaskMessageField editor={editor} controlsDisabled={controlsDisabled} onChangeEditor={onChangeEditor} />
        <TaskScheduleFields editor={editor} controlsDisabled={controlsDisabled} onChangeEditor={onChangeEditor} />
        {editor.taskType === 'loop' ? (
          <LoopTaskFields
            editor={editor}
            presets={presets}
            controlsDisabled={controlsDisabled}
            onChangeEditor={onChangeEditor}
          />
        ) : (
          <TextTaskFields
            editor={editor}
            presets={presets}
            sessionOptions={sessionOptions}
            controlsDisabled={controlsDisabled}
            onChangeEditor={onChangeEditor}
          />
        )}
      </div>
    </div>
  );
}

function TaskTypeField(props: {
  editor: TaskEditorState;
  controlsDisabled: boolean;
  onSelectTaskType: (taskType: TaskEditorType) => void;
}) {
  const { copy } = useWebLocale();
  const options: Array<{ value: TaskEditorType; label: string }> = [
    { value: 'text', label: copy.settings.taskEditorTypeText },
    { value: 'loop', label: copy.settings.taskEditorTypeLoop },
  ];

  return (
    <TaskFormField label={copy.settings.taskEditorTypeLabel} description={copy.settings.taskEditorTypeDescription}>
      <SoftDropdownSelect
        value={props.editor.taskType}
        disabled={props.controlsDisabled}
        options={options}
        testId="task-editor-type-select"
        onChange={(value) => props.onSelectTaskType(value as TaskEditorType)}
      />
    </TaskFormField>
  );
}

function TaskMessageField(props: {
  editor: TaskEditorState;
  controlsDisabled: boolean;
  onChangeEditor: (patch: Partial<TaskEditorState>) => void;
}) {
  const { copy } = useWebLocale();
  const { editor, controlsDisabled, onChangeEditor } = props;
  const loopTask = editor.taskType === 'loop';

  return (
    <TaskFormField
      label={loopTask ? copy.settings.loopEditorMessageLabel : copy.settings.taskEditorMessageLabel}
      description={loopTask ? copy.settings.loopEditorMessageDescription : copy.settings.taskEditorMessageDescription}
    >
      <textarea
        value={editor.message}
        disabled={controlsDisabled}
        rows={loopTask ? 5 : 4}
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
      <TaskFormField
        label={editor.taskType === 'loop' ? copy.settings.loopEditorScheduleModeLabel : copy.settings.taskEditorScheduleModeLabel}
        description={editor.taskType === 'loop' ? copy.settings.loopEditorScheduleModeDescription : copy.settings.taskEditorScheduleModeDescription}
      >
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

function TextTaskFields(props: {
  editor: TaskEditorState;
  presets: PresetPayload[];
  sessionOptions: string[];
  controlsDisabled: boolean;
  onChangeEditor: (patch: Partial<TaskEditorState>) => void;
}) {
  const { editor, presets, sessionOptions, controlsDisabled, onChangeEditor } = props;

  return (
    <>
      <TaskSessionField editor={editor} controlsDisabled={controlsDisabled} onChangeEditor={onChangeEditor} />
      <TaskRuntimeSection
        editor={editor}
        presets={presets}
        controlsDisabled={controlsDisabled}
        onChangeEditor={onChangeEditor}
      />
      <TaskSessionOptions sessionOptions={sessionOptions} />
    </>
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

function LoopTaskFields(props: {
  editor: TaskEditorState;
  presets: PresetPayload[];
  controlsDisabled: boolean;
  onChangeEditor: (patch: Partial<TaskEditorState>) => void;
}) {
  const { editor, presets, controlsDisabled, onChangeEditor } = props;

  return (
    <>
      <LoopStopPolicyField editor={editor} controlsDisabled={controlsDisabled} onChangeEditor={onChangeEditor} />
      <LoopNumberField
        labelKey="maxRounds"
        value={editor.relayMaxRounds}
        min={1}
        disabled={controlsDisabled}
        onChange={(relayMaxRounds) => onChangeEditor({ relayMaxRounds })}
      />
      <LoopPresetField
        editor={editor}
        presets={presets}
        controlsDisabled={controlsDisabled}
        onChangeEditor={onChangeEditor}
      />
      <LoopNumberField
        labelKey="timeout"
        value={editor.relayExecutionTimeoutMS}
        min={0}
        disabled={controlsDisabled}
        onChange={(relayExecutionTimeoutMS) => onChangeEditor({ relayExecutionTimeoutMS })}
      />
    </>
  );
}

function LoopStopPolicyField(props: {
  editor: TaskEditorState;
  controlsDisabled: boolean;
  onChangeEditor: (patch: Partial<TaskEditorState>) => void;
}) {
  const { copy } = useWebLocale();
  const { editor, controlsDisabled, onChangeEditor } = props;
  const options: Array<{ value: TaskRelayStopPolicy; label: string }> = [
    { value: 'ai_decides', label: copy.settings.loopStopAIDecides },
    { value: 'max_rounds', label: copy.settings.loopStopMaxRounds },
  ];

  return (
    <TaskFormField label={copy.settings.loopEditorStopPolicyLabel} description={copy.settings.loopEditorStopPolicyDescription}>
      <SoftDropdownSelect
        value={editor.relayStopPolicy}
        disabled={controlsDisabled}
        options={options}
        onChange={(value) => onChangeEditor({ relayStopPolicy: value as TaskRelayStopPolicy })}
      />
    </TaskFormField>
  );
}

function LoopPresetField(props: {
  editor: TaskEditorState;
  presets: PresetPayload[];
  controlsDisabled: boolean;
  onChangeEditor: (patch: Partial<TaskEditorState>) => void;
}) {
  const { copy } = useWebLocale();
  const { editor, presets, controlsDisabled, onChangeEditor } = props;

  return (
    <TaskFormField label={copy.settings.loopEditorPresetLabel} description={copy.settings.loopEditorPresetDescription}>
      <SoftDropdownSelect
        value={editor.runtimePresetId}
        disabled={controlsDisabled}
        options={buildLoopPresetOptions(editor.runtimePresetId, presets, copy)}
        onChange={(runtimePresetId) => onChangeEditor({ runtimePresetId })}
      />
    </TaskFormField>
  );
}

function buildLoopPresetOptions(
  currentValue: string,
  presets: PresetPayload[],
  copy: ReturnType<typeof useWebLocale>['copy'],
) {
  const options = [
    { value: '', label: copy.settings.loopEditorPresetNone },
    ...presets
      .slice()
      .sort((left, right) => left.name.localeCompare(right.name))
      .map((preset) => ({ value: preset.id, label: preset.name })),
  ];
  if (currentValue === '' || options.some((option) => option.value === currentValue)) {
    return options;
  }
  return [
    {
      value: currentValue,
      label: copy.settings.taskEditorRuntimePresetUnavailable(currentValue),
      disabled: true,
    },
    ...options,
  ];
}

function LoopNumberField(props: {
  labelKey: 'maxRounds' | 'timeout';
  value: string;
  min: number;
  disabled: boolean;
  onChange: (value: string) => void;
}) {
  const { copy } = useWebLocale();
  const maxRounds = props.labelKey === 'maxRounds';

  return (
    <TaskFormField
      label={maxRounds ? copy.settings.loopEditorMaxRoundsLabel : copy.settings.loopEditorTimeoutLabel}
      description={maxRounds ? copy.settings.loopEditorMaxRoundsDescription : copy.settings.loopEditorTimeoutDescription}
    >
      <input
        value={props.value}
        disabled={props.disabled}
        type="number"
        min={props.min}
        onChange={(event) => props.onChange(event.target.value)}
        className="w-full rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-2.5 font-mono text-[14px] text-[#111111] transition-colors focus:border-[#111111] focus:outline-none disabled:opacity-60"
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
      <CloseButton
        onClick={onCancelEditing}
        aria-label={copy.settings.cancel}
      />
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
