'use client';

import { SoftDropdownSelect } from '@/components/SoftDropdownSelect';
import { TaskFormField } from '@/components/config/TaskFormField';
import type { TaskAgentMode, TaskEditorState, TaskRelayStopPolicy } from '@/lib/configTasks';
import { useWebLocale } from '@/lib/i18n/provider';

interface TaskRelaySectionProps {
  editor: TaskEditorState;
  controlsDisabled: boolean;
  onChangeEditor: (patch: Partial<TaskEditorState>) => void;
}

export function TaskRelaySection(props: TaskRelaySectionProps) {
  const { copy } = useWebLocale();
  const { editor, controlsDisabled, onChangeEditor } = props;
  const agentModeOptions: Array<{ value: TaskAgentMode; label: string }> = [
    { value: 'single', label: copy.settings.taskEditorAgentModeSingle },
    { value: 'relay', label: copy.settings.taskEditorAgentModeRelay },
  ];

  return (
    <details className="rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] p-4" open={editor.agentMode === 'relay'}>
      <summary className="cursor-pointer list-none text-[13px] font-semibold text-[#111111]">
        <span className="mr-1 inline-block text-[#737373]">{'>'}</span>
        {copy.settings.taskEditorRelaySection}
      </summary>
      <div className="mt-4 space-y-5">
        <TaskAgentModeField
          value={editor.agentMode}
          disabled={controlsDisabled}
          options={agentModeOptions}
          onChange={(agentMode) => onChangeEditor({ agentMode })}
        />
        {editor.agentMode === 'relay' ? (
          <TaskRelayControls editor={editor} controlsDisabled={controlsDisabled} onChangeEditor={onChangeEditor} />
        ) : null}
      </div>
    </details>
  );
}

function TaskAgentModeField(props: {
  value: TaskAgentMode;
  disabled: boolean;
  options: Array<{ value: TaskAgentMode; label: string }>;
  onChange: (mode: TaskAgentMode) => void;
}) {
  const { copy } = useWebLocale();
  const { value, disabled, options, onChange } = props;

  return (
    <TaskFormField label={copy.settings.taskEditorAgentModeLabel} description={copy.settings.taskEditorAgentModeDescription}>
      <SoftDropdownSelect
        value={value}
        disabled={disabled}
        options={options}
        onChange={(nextValue) => onChange(nextValue as TaskAgentMode)}
      />
    </TaskFormField>
  );
}

function TaskRelayControls(props: {
  editor: TaskEditorState;
  controlsDisabled: boolean;
  onChangeEditor: (patch: Partial<TaskEditorState>) => void;
}) {
  const { copy } = useWebLocale();
  const { editor, controlsDisabled, onChangeEditor } = props;
  const stopPolicyOptions: Array<{ value: TaskRelayStopPolicy; label: string }> = [
    { value: 'ai_decides', label: copy.settings.taskEditorRelayStopAIDecides },
    { value: 'max_rounds', label: copy.settings.taskEditorRelayStopMaxRounds },
  ];

  return (
    <>
      <TaskRelayStopPolicyField
        value={editor.relayStopPolicy}
        disabled={controlsDisabled}
        options={stopPolicyOptions}
        onChange={(relayStopPolicy) => onChangeEditor({ relayStopPolicy })}
      />
      <TaskRelayNumberField
        label={copy.settings.taskEditorRelayMaxRoundsLabel}
        description={copy.settings.taskEditorRelayMaxRoundsDescription}
        value={editor.relayMaxRounds}
        min={1}
        disabled={controlsDisabled}
        onChange={(relayMaxRounds) => onChangeEditor({ relayMaxRounds })}
      />
      <TaskRelayNumberField
        label={copy.settings.taskEditorRelayTimeoutLabel}
        description={copy.settings.taskEditorRelayTimeoutDescription}
        value={editor.relayExecutionTimeoutMS}
        min={0}
        disabled={controlsDisabled}
        onChange={(relayExecutionTimeoutMS) => onChangeEditor({ relayExecutionTimeoutMS })}
      />
    </>
  );
}

function TaskRelayStopPolicyField(props: {
  value: TaskRelayStopPolicy;
  disabled: boolean;
  options: Array<{ value: TaskRelayStopPolicy; label: string }>;
  onChange: (policy: TaskRelayStopPolicy) => void;
}) {
  const { copy } = useWebLocale();
  const { value, disabled, options, onChange } = props;

  return (
    <TaskFormField label={copy.settings.taskEditorRelayStopPolicyLabel} description={copy.settings.taskEditorRelayStopPolicyDescription}>
      <SoftDropdownSelect
        value={value}
        disabled={disabled}
        options={options}
        onChange={(nextValue) => onChange(nextValue as TaskRelayStopPolicy)}
      />
    </TaskFormField>
  );
}

function TaskRelayNumberField(props: {
  label: string;
  description: string;
  value: string;
  min: number;
  disabled: boolean;
  onChange: (value: string) => void;
}) {
  const { label, description, value, min, disabled, onChange } = props;

  return (
    <TaskFormField label={label} description={description}>
      <input
        value={value}
        disabled={disabled}
        type="number"
        min={min}
        onChange={(event) => onChange(event.target.value)}
        className="w-full rounded-[12px] border border-[#E5E5E5] bg-white px-4 py-2.5 font-mono text-[14px] text-[#111111] placeholder-[#A3A3A3] transition-colors focus:border-[#111111] focus:outline-none disabled:opacity-60"
      />
    </TaskFormField>
  );
}
