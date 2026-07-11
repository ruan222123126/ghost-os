'use client';

import {
  parseOptionalPositiveInteger,
  type SelectOption,
  toggleToolOverride,
} from '@/components/workflow/workflowAgentNodeEditorState';
import type { TaskRuntimeOverrides } from '@/lib/types';
import { withAgentRuntimeOverrides } from '@/lib/workflow-editor';
import type { AgentEditorContext } from '@/components/workflow/WorkflowCanvasAgentNodeEditorFields';

export function AgentToolAllowlistField(props: { context: AgentEditorContext }) {
  const { context } = props;
  const statusText = toolStatusText(context);

  return (
    <>
      <label className="workflow-arch-field-label">{context.copy.agentToolAllowlist}</label>
      {statusText ? <div className="workflow-arch-field-note">{statusText}</div> : null}
      <div className="max-h-48 overflow-y-auto rounded-[18px] border border-[#E5E5E5] bg-white px-3 py-2">
        {context.toolOptions.map((option) => (
          <AgentToolOptionRow key={option.value} context={context} option={option} />
        ))}
      </div>
    </>
  );
}

export function AgentMaxTurnsField(props: { context: AgentEditorContext }) {
  const { context } = props;

  if (context.editorKind !== 'workflow') {
    return null;
  }

  return (
    <>
      <label className="workflow-arch-field-label">{context.copy.agentMaxTurns}</label>
      <input
        type="number"
        min={1}
        step={1}
        value={maxTurnsInputValue(context.overrides)}
        onChange={(event) => updateAgentRuntimeOverrides(context, {
          max_turns: parseOptionalPositiveInteger(event.target.value),
        })}
      />
    </>
  );
}

export function AgentEditorNotes(props: { context: AgentEditorContext }) {
  const { context } = props;

  return (
    <>
      <TrimmedFieldNote text={context.agentRuntimeError} />
      <TrimmedFieldNote text={context.presetError} />
      {context.editorKind === 'workflow' ? <p className="workflow-arch-field-note">{context.copy.runtimeVariableHint}</p> : null}
    </>
  );
}

function AgentToolOptionRow(props: { context: AgentEditorContext; option: SelectOption }) {
  const { context, option } = props;

  return (
    <label className={`flex items-center gap-2 py-1 text-[13px] ${toolOptionTextClass(option)}`}>
      <input
        type="checkbox"
        checked={context.selectedToolNames.includes(option.value)}
        disabled={option.disabled}
        onChange={() => updateAgentRuntimeOverrides(
          context,
          toggleToolOverride(context.overrides, option.value, context.toolOrder),
        )}
      />
      <span>{option.label}</span>
    </label>
  );
}

function TrimmedFieldNote(props: { text: string }) {
  const { text } = props;

  if (text.trim().length === 0) {
    return null;
  }

  return <p className="workflow-arch-field-note">{text}</p>;
}

function updateAgentRuntimeOverrides(
  context: AgentEditorContext,
  overrides: TaskRuntimeOverrides | undefined,
): void {
  context.onUpdateNode(withAgentRuntimeOverrides(context.selectedNode, overrides));
}

function toolStatusText(context: AgentEditorContext): string {
  if (context.agentRuntimeLoading) {
    return context.copy.agentToolsLoading;
  }
  if (context.toolOptions.length === 0) {
    return context.copy.agentNoTools;
  }
  return '';
}

function toolOptionTextClass(option: SelectOption): string {
  if (option.disabled) {
    return 'text-[#A3A3A3]';
  }
  return 'text-[#171717]';
}

function maxTurnsInputValue(overrides: TaskRuntimeOverrides): string {
  if (overrides.max_turns === undefined) {
    return '';
  }
  return String(overrides.max_turns);
}
