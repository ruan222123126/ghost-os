'use client';

import { SoftDropdownSelect, type DropdownSelectOption } from '@/components/SoftDropdownSelect';
import { TaskFormField } from '@/components/config/TaskFormField';
import type { TaskEditorState } from '@/lib/configTasks';
import { useWebLocale } from '@/lib/i18n/provider';
import type { PresetPayload } from '@/lib/types';

interface TaskRuntimeSectionProps {
  editor: TaskEditorState;
  presets: PresetPayload[];
  controlsDisabled: boolean;
  onChangeEditor: (patch: Partial<TaskEditorState>) => void;
}

export function TaskRuntimeSection(props: TaskRuntimeSectionProps) {
  const { copy } = useWebLocale();
  const { editor, controlsDisabled, onChangeEditor } = props;
  const disabled = controlsDisabled || !editor.runtimeOverridesEnabled;

  return (
    <details className="rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] p-4">
      <summary className="cursor-pointer list-none text-[13px] font-semibold text-[#111111]">
        <span className="mr-1 inline-block text-[#737373]">{'>'}</span>
        {copy.settings.taskEditorRuntimeSection}
      </summary>
      <div className="mt-4 space-y-5">
        <TaskRuntimeToggle editor={editor} controlsDisabled={controlsDisabled} onChangeEditor={onChangeEditor} />
        <TaskRuntimePresetField
          editor={editor}
          presets={props.presets}
          disabled={disabled}
          onChangeEditor={onChangeEditor}
        />
        <TaskRuntimeModelField editor={editor} disabled={disabled} onChangeEditor={onChangeEditor} />
        <TaskRuntimeToolsField editor={editor} disabled={disabled} onChangeEditor={onChangeEditor} />
      </div>
    </details>
  );
}

function TaskRuntimeToggle(props: Omit<TaskRuntimeSectionProps, 'presets'>) {
  const { copy } = useWebLocale();
  const { editor, controlsDisabled, onChangeEditor } = props;

  return (
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
  );
}

function TaskRuntimePresetField(props: {
  editor: TaskEditorState;
  presets: PresetPayload[];
  disabled: boolean;
  onChangeEditor: (patch: Partial<TaskEditorState>) => void;
}) {
  const { copy } = useWebLocale();
  const { editor, presets, disabled, onChangeEditor } = props;
  const options = buildPresetOptions(editor.runtimePresetId, presets, copy);

  return (
    <TaskFormField
      label={copy.settings.taskEditorRuntimePresetLabel}
      description={copy.settings.taskEditorRuntimePresetDescription}
    >
      <SoftDropdownSelect
        value={editor.runtimePresetId}
        disabled={disabled}
        options={options}
        testId="task-runtime-preset-select"
        onChange={(presetId) => onChangeEditor(taskRuntimePresetPatch(presetId, presets))}
      />
    </TaskFormField>
  );
}

function TaskRuntimeModelField(props: {
  editor: TaskEditorState;
  disabled: boolean;
  onChangeEditor: (patch: Partial<TaskEditorState>) => void;
}) {
  const { copy } = useWebLocale();
  const { editor, disabled, onChangeEditor } = props;

  return (
    <TaskFormField label={copy.settings.taskEditorRuntimeModelLabel} description={copy.settings.taskEditorRuntimeModelDescription}>
      <input
        value={editor.runtimeModel}
        disabled={disabled}
        onChange={(event) => onChangeEditor({ runtimeModel: event.target.value })}
        placeholder={copy.settings.taskEditorRuntimeModelPlaceholder}
        className="w-full rounded-[12px] border border-[#E5E5E5] bg-white px-4 py-2.5 font-mono text-[14px] text-[#111111] placeholder-[#A3A3A3] transition-colors focus:border-[#111111] focus:outline-none disabled:opacity-60"
      />
    </TaskFormField>
  );
}

function buildPresetOptions(
  currentValue: string,
  presets: PresetPayload[],
  copy: ReturnType<typeof useWebLocale>['copy'],
): DropdownSelectOption[] {
  const options = [
    { value: '', label: copy.settings.taskEditorRuntimePresetNone },
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

function taskRuntimePresetPatch(
  presetId: string,
  presets: PresetPayload[],
): Partial<TaskEditorState> {
  const id = presetId.trim();
  if (id === '') {
    return { runtimePresetId: '' };
  }
  const preset = presets.find((item) => item.id === id);
  return {
    runtimePresetId: id,
    runtimeToolAllowlist: preset ? preset.tool_allowlist.join(', ') : '',
  };
}

function TaskRuntimeToolsField(props: {
  editor: TaskEditorState;
  disabled: boolean;
  onChangeEditor: (patch: Partial<TaskEditorState>) => void;
}) {
  const { copy } = useWebLocale();
  const { editor, disabled, onChangeEditor } = props;

  return (
    <TaskFormField label={copy.settings.taskEditorRuntimeToolsLabel} description={copy.settings.taskEditorRuntimeToolsDescription}>
      <textarea
        value={editor.runtimeToolAllowlist}
        disabled={disabled}
        rows={3}
        onChange={(event) => onChangeEditor({ runtimeToolAllowlist: event.target.value })}
        placeholder={copy.settings.taskEditorRuntimeToolsPlaceholder}
        className="w-full rounded-[12px] border border-[#E5E5E5] bg-white px-4 py-2.5 font-mono text-[13px] text-[#111111] placeholder-[#A3A3A3] transition-colors focus:border-[#111111] focus:outline-none disabled:opacity-60"
      />
    </TaskFormField>
  );
}
