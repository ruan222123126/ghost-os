'use client';

import type { FormEvent } from 'react';
import { CloseButton } from '@/components/CloseButton';
import { SoftDropdownSelect } from '@/components/SoftDropdownSelect';
import { TaskFormField } from '@/components/config/TaskFormField';
import { ignorePromise } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { LoopEditorMode, LoopEditorState, LoopScheduleMode, LoopStopPolicy } from '@/lib/configLoops';
import type { PresetPayload } from '@/lib/types';

interface LoopEditorFormProps {
  mode: LoopEditorMode;
  editor: LoopEditorState;
  presets: PresetPayload[];
  controlsDisabled: boolean;
  saving: boolean;
  onChangeEditor: (patch: Partial<LoopEditorState>) => void;
  onSubmit: () => Promise<boolean>;
  onCancel: () => void;
}

export function LoopEditorForm(props: LoopEditorFormProps) {
  const { copy } = useWebLocale();
  const { mode, editor, presets, controlsDisabled, saving, onChangeEditor, onSubmit, onCancel } = props;

  return (
    <form className="space-y-8" onSubmit={(event) => handleSubmit(event, onSubmit)}>
      <div>
        <p className="mb-2 text-[11px] font-bold uppercase tracking-[0.15em] text-[#737373]">
          {mode === 'edit' ? copy.settings.loopEditorEditing : copy.settings.loopEditorNew}
        </p>
        <h3 className="text-[20px] font-medium text-[#111111]">
          {mode === 'edit' ? copy.settings.loopEditorEditTitle : copy.settings.loopEditorCreateTitle}
        </h3>
      </div>

      <div className="rounded-[16px] border border-[#E5E5E5] bg-white p-6">
        <div className="space-y-5">
          <LoopMessageField editor={editor} controlsDisabled={controlsDisabled} onChangeEditor={onChangeEditor} />
          <LoopScheduleFields editor={editor} controlsDisabled={controlsDisabled} onChangeEditor={onChangeEditor} />
          <LoopStopPolicyField editor={editor} controlsDisabled={controlsDisabled} onChangeEditor={onChangeEditor} />
          <LoopMaxRoundsField editor={editor} controlsDisabled={controlsDisabled} onChangeEditor={onChangeEditor} />
          <LoopPresetField editor={editor} presets={presets} controlsDisabled={controlsDisabled} onChangeEditor={onChangeEditor} />
          <LoopTimeoutField editor={editor} controlsDisabled={controlsDisabled} onChangeEditor={onChangeEditor} />
        </div>
      </div>

      <LoopEditorActions
        controlsDisabled={controlsDisabled}
        submitLabel={saving ? copy.settings.saving : resolveSubmitLabel(mode, copy)}
        onCancel={onCancel}
      />
    </form>
  );
}

function LoopMessageField(props: LoopFieldProps) {
  const { copy } = useWebLocale();
  const { editor, controlsDisabled, onChangeEditor } = props;

  return (
    <TaskFormField label={copy.settings.loopEditorMessageLabel} description={copy.settings.loopEditorMessageDescription}>
      <textarea
        value={editor.message}
        disabled={controlsDisabled}
        rows={5}
        onChange={(event) => onChangeEditor({ message: event.target.value })}
        className="w-full rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-2.5 text-[14px] text-[#111111] placeholder-[#A3A3A3] transition-colors focus:border-[#111111] focus:outline-none"
      />
    </TaskFormField>
  );
}

function LoopScheduleFields(props: LoopFieldProps) {
  const { copy } = useWebLocale();
  const { editor, controlsDisabled, onChangeEditor } = props;
  const cronMode = editor.scheduleMode === 'cron';
  const options: Array<{ value: LoopScheduleMode; label: string }> = [
    { value: 'interval', label: copy.settings.taskEditorModeInterval },
    { value: 'cron', label: copy.settings.taskEditorModeCron },
  ];

  return (
    <>
      <TaskFormField label={copy.settings.loopEditorScheduleModeLabel} description={copy.settings.loopEditorScheduleModeDescription}>
        <SoftDropdownSelect
          value={editor.scheduleMode}
          disabled={controlsDisabled}
          options={options}
          onChange={(value) => onChangeEditor({ scheduleMode: value as LoopScheduleMode })}
        />
      </TaskFormField>
      <TaskFormField
        label={cronMode ? copy.settings.taskEditorCronLabel : copy.settings.taskEditorIntervalLabel}
        description={cronMode ? copy.settings.taskEditorCronDescription : copy.settings.taskEditorIntervalDescription}
      >
        <input
          value={cronMode ? editor.cronExpr : editor.intervalSeconds}
          disabled={controlsDisabled}
          onChange={(event) => onChangeEditor(cronMode ? { cronExpr: event.target.value } : { intervalSeconds: event.target.value })}
          className="w-full rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-2.5 font-mono text-[14px] text-[#111111] transition-colors focus:border-[#111111] focus:outline-none"
        />
      </TaskFormField>
    </>
  );
}

function LoopStopPolicyField(props: LoopFieldProps) {
  const { copy } = useWebLocale();
  const { editor, controlsDisabled, onChangeEditor } = props;
  const options: Array<{ value: LoopStopPolicy; label: string }> = [
    { value: 'ai_decides', label: copy.settings.loopStopAIDecides },
    { value: 'max_rounds', label: copy.settings.loopStopMaxRounds },
  ];

  return (
    <TaskFormField label={copy.settings.loopEditorStopPolicyLabel} description={copy.settings.loopEditorStopPolicyDescription}>
      <SoftDropdownSelect
        value={editor.stopPolicy}
        disabled={controlsDisabled}
        options={options}
        onChange={(value) => onChangeEditor({ stopPolicy: value as LoopStopPolicy })}
      />
    </TaskFormField>
  );
}

function LoopMaxRoundsField(props: LoopFieldProps) {
  const { copy } = useWebLocale();
  const { editor, controlsDisabled, onChangeEditor } = props;

  return (
    <LoopNumberField
      label={copy.settings.loopEditorMaxRoundsLabel}
      description={copy.settings.loopEditorMaxRoundsDescription}
      value={editor.maxRounds}
      min={1}
      disabled={controlsDisabled}
      onChange={(maxRounds) => onChangeEditor({ maxRounds })}
    />
  );
}

function LoopPresetField(props: LoopFieldProps & { presets: PresetPayload[] }) {
  const { copy } = useWebLocale();
  const { editor, presets, controlsDisabled, onChangeEditor } = props;
  const options = [
    { value: '', label: copy.settings.loopEditorPresetNone },
    ...presets.map((preset) => ({ value: preset.id, label: preset.name })),
  ];

  return (
    <TaskFormField label={copy.settings.loopEditorPresetLabel} description={copy.settings.loopEditorPresetDescription}>
      <SoftDropdownSelect
        value={editor.presetId}
        disabled={controlsDisabled}
        options={options}
        onChange={(presetId) => onChangeEditor({ presetId })}
      />
    </TaskFormField>
  );
}

function LoopTimeoutField(props: LoopFieldProps) {
  const { copy } = useWebLocale();
  const { editor, controlsDisabled, onChangeEditor } = props;

  return (
    <LoopNumberField
      label={copy.settings.loopEditorTimeoutLabel}
      description={copy.settings.loopEditorTimeoutDescription}
      value={editor.executionTimeoutMS}
      min={0}
      disabled={controlsDisabled}
      onChange={(executionTimeoutMS) => onChangeEditor({ executionTimeoutMS })}
    />
  );
}

interface LoopFieldProps {
  editor: LoopEditorState;
  controlsDisabled: boolean;
  onChangeEditor: (patch: Partial<LoopEditorState>) => void;
}

function LoopNumberField(props: {
  label: string;
  description: string;
  value: string;
  min: number;
  disabled: boolean;
  onChange: (value: string) => void;
}) {
  return (
    <TaskFormField label={props.label} description={props.description}>
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

function LoopEditorActions(props: {
  controlsDisabled: boolean;
  submitLabel: string;
  onCancel: () => void;
}) {
  const { copy } = useWebLocale();

  return (
    <div className="flex items-center justify-end gap-3">
      <CloseButton onClick={props.onCancel} aria-label={copy.settings.cancel} />
      <button type="submit" disabled={props.controlsDisabled} className="rounded-full bg-[#111111] px-8 py-2.5 text-[13px] font-semibold text-white transition-colors hover:bg-[#333333] disabled:cursor-not-allowed disabled:opacity-50">
        {props.submitLabel}
      </button>
    </div>
  );
}

function resolveSubmitLabel(
  mode: LoopEditorMode,
  copy: ReturnType<typeof useWebLocale>['copy'],
): string {
  return mode === 'edit' ? copy.settings.taskEditorSubmitSave : copy.settings.taskEditorSubmitCreate;
}

function handleSubmit(event: FormEvent<HTMLFormElement>, onSubmit: () => Promise<boolean>) {
  event.preventDefault();
  ignorePromise(onSubmit());
}
