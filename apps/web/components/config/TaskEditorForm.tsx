'use client';

import type { FormEvent, ReactNode } from 'react';
import { ignorePromise } from '@/lib/errors';
import type {
  TaskEditorKind,
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

const SCHEDULE_MODE_OPTIONS: Array<{ value: TaskScheduleMode; label: string }> = [
  { value: 'interval', label: 'Interval' },
  { value: 'cron', label: 'Cron' },
];
const TASK_KIND_OPTIONS: Array<{ value: TaskEditorKind; label: string }> = [
  { value: 'workflow', label: 'Workflow' },
  { value: 'agent_message', label: 'Text Task' },
];

export function TaskEditorForm(props: TaskEditorFormProps) {
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
  const title = resolveEditorTitle(editorMode, editor.taskKind);
  const submitLabel = resolveSubmitLabel(editorMode, editor.taskKind, saving);

  return (
    <form className="space-y-8" onSubmit={(event) => handleTaskSubmit(event, onSubmit)}>
      <div>
        <p className="mb-2 text-[11px] font-bold uppercase tracking-[0.15em] text-[#737373]">
          {editorMode === 'edit' ? 'Editing Task' : 'New Task'}
        </p>
        <h3 className="text-[20px] font-medium text-[#111111]">{title}</h3>
      </div>

      <div className="rounded-[16px] border border-[#E5E5E5] bg-white p-6">
        <div className="space-y-5">
          {editorMode === 'create' ? (
            <FormField label="Task Kind" description="Choose a text task or a workflow task.">
              <select
                value={editor.taskKind}
                disabled={controlsDisabled}
                onChange={(event) => {
                  const taskKind = event.target.value as TaskEditorKind;
                  onChangeEditor({
                    taskKind,
                    runtimeOverridesEnabled: taskKind === 'agent_message' ? editor.runtimeOverridesEnabled : false,
                  });
                }}
                className="w-full rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-2.5 text-[14px] text-[#111111] transition-colors focus:border-[#111111] focus:outline-none"
              >
                {TASK_KIND_OPTIONS.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
            </FormField>
          ) : null}

          {editor.taskKind === 'agent_message' ? (
            <FormField label="Message" description="Prompt sent by scheduler when task runs.">
              <textarea
                value={editor.message}
                disabled={controlsDisabled}
                rows={4}
                onChange={(event) => onChangeEditor({ message: event.target.value })}
                className="w-full rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-2.5 text-[14px] text-[#111111] placeholder-[#A3A3A3] transition-colors focus:border-[#111111] focus:outline-none"
              />
            </FormField>
          ) : (
            <FormField
              label="Session ID (Optional)"
              description="Workflow will be generated from frontend text tasks in this session. Leave empty to generate a new session ID on each run."
            >
              <input
                list="task-session-options"
                value={editor.workflowSessionId}
                disabled={controlsDisabled}
                onChange={(event) => onChangeEditor({ workflowSessionId: event.target.value })}
                className="w-full rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-2.5 font-mono text-[14px] text-[#111111] placeholder-[#A3A3A3] transition-colors focus:border-[#111111] focus:outline-none"
              />
            </FormField>
          )}

          <FormField label="Schedule Mode" description="Pick one mode: interval or cron.">
            <select
              value={editor.scheduleMode}
              disabled={controlsDisabled}
              onChange={(event) => onChangeEditor({ scheduleMode: event.target.value as TaskScheduleMode })}
              className="w-full rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-2.5 text-[14px] text-[#111111] transition-colors focus:border-[#111111] focus:outline-none"
            >
              {SCHEDULE_MODE_OPTIONS.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          </FormField>

          {editor.scheduleMode === 'interval' ? (
            <FormField
              label="Interval Seconds"
              description="Positive integer. Example: 300 means every 5 minutes."
            >
              <input
                value={editor.intervalSeconds}
                disabled={controlsDisabled}
                onChange={(event) => onChangeEditor({ intervalSeconds: event.target.value })}
                className="w-full rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-2.5 font-mono text-[14px] text-[#111111] placeholder-[#A3A3A3] transition-colors focus:border-[#111111] focus:outline-none"
              />
            </FormField>
          ) : (
            <FormField label="Cron Expression" description='Example: "0 9 * * 1" (every Monday at 09:00).'>
              <input
                value={editor.cronExpr}
                disabled={controlsDisabled}
                onChange={(event) => onChangeEditor({ cronExpr: event.target.value })}
                className="w-full rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-2.5 font-mono text-[14px] text-[#111111] placeholder-[#A3A3A3] transition-colors focus:border-[#111111] focus:outline-none"
              />
            </FormField>
          )}

          {editor.taskKind === 'agent_message' ? (
            <>
              <FormField
                label="Session ID (Optional)"
                description="Leave empty to start a new session each run."
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
                  Custom Runtime Config (Optional)
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
                    Enable task-specific runtime config
                  </label>

                  <FormField
                    label="Model (Optional)"
                    description="Leave empty to use the global default model."
                  >
                    <input
                      value={editor.runtimeModel}
                      disabled={controlsDisabled || !editor.runtimeOverridesEnabled}
                      onChange={(event) => onChangeEditor({ runtimeModel: event.target.value })}
                      placeholder="e.g. gpt-5.4"
                      className="w-full rounded-[12px] border border-[#E5E5E5] bg-white px-4 py-2.5 font-mono text-[14px] text-[#111111] placeholder-[#A3A3A3] transition-colors focus:border-[#111111] focus:outline-none disabled:opacity-60"
                    />
                  </FormField>

                  <FormField
                    label="Allowed Tools (Optional)"
                    description="Comma or newline separated tool names. Leave empty to use global tool config."
                  >
                    <textarea
                      value={editor.runtimeToolAllowlist}
                      disabled={controlsDisabled || !editor.runtimeOverridesEnabled}
                      rows={3}
                      onChange={(event) => onChangeEditor({ runtimeToolAllowlist: event.target.value })}
                      placeholder="script_exec, web_search"
                      className="w-full rounded-[12px] border border-[#E5E5E5] bg-white px-4 py-2.5 font-mono text-[13px] text-[#111111] placeholder-[#A3A3A3] transition-colors focus:border-[#111111] focus:outline-none disabled:opacity-60"
                    />
                  </FormField>
                </div>
              </details>
            </>
          ) : null}

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
          Cancel
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

function resolveEditorTitle(mode: TaskEditorMode, kind: TaskEditorKind): string {
  if (mode === 'edit') {
    return 'Edit Task';
  }

  return kind === 'workflow' ? 'New Workflow' : 'New Task';
}

function resolveSubmitLabel(mode: TaskEditorMode, kind: TaskEditorKind, saving: boolean): string {
  if (saving) {
    return 'Saving...';
  }
  if (mode === 'edit') {
    return 'Save';
  }

  return kind === 'workflow' ? 'Create Workflow' : 'Create';
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
