'use client';

import { TaskFormField } from '@/components/config/TaskFormField';
import type { TaskEditorState } from '@/lib/configTasks';
import { useWebLocale } from '@/lib/i18n/provider';

interface TaskRuntimeSectionProps {
  editor: TaskEditorState;
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
        <TaskRuntimeModelField editor={editor} disabled={disabled} onChangeEditor={onChangeEditor} />
        <TaskRuntimeToolsField editor={editor} disabled={disabled} onChangeEditor={onChangeEditor} />
      </div>
    </details>
  );
}

function TaskRuntimeToggle(props: TaskRuntimeSectionProps) {
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
