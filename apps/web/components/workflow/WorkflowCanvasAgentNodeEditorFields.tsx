'use client';

import { WorkflowTemplateEnabledField } from '@/components/workflow/WorkflowTemplateEnabledField';
import {
  AgentEditorNotes,
  AgentMaxTurnsField,
  AgentToolAllowlistField,
} from '@/components/workflow/WorkflowCanvasAgentNodeEditorToolFields';
import {
  applyPresetOverride,
  buildModelOptions,
  buildPresetOptions,
  buildProviderOptions,
  nextProviderOverrides,
  nextRuntimeOverrides,
  type SelectOption,
} from '@/components/workflow/workflowAgentNodeEditorState';
import type { WorkflowCopy } from '@/lib/i18n/messages/workflow';
import type { PresetPayload, TaskRuntimeOverrides } from '@/lib/types';
import {
  type WorkflowAgentRuntimeCatalog,
  type WorkflowCanvasNodeDraft,
  type WorkflowEditorKind,
  withAgentMessage,
  withAgentRuntimeOverrides,
  withAgentTitle,
} from '@/lib/workflow-editor';

export interface AgentEditorContext {
  agentRuntimeCatalog?: WorkflowAgentRuntimeCatalog;
  agentRuntimeError: string;
  agentRuntimeLoading: boolean;
  copy: WorkflowCopy;
  editorKind: WorkflowEditorKind;
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
  overrides: TaskRuntimeOverrides;
  presetError: string;
  presetLoading: boolean;
  presets: PresetPayload[];
  selectedNode: WorkflowCanvasNodeDraft;
  selectedToolNames: string[];
  toolOptions: SelectOption[];
  toolOrder: string[];
}

interface NextPresetRuntimeOverridesOptions {
  current: TaskRuntimeOverrides;
  orderedToolNames: string[];
  presetID: string;
  presets: PresetPayload[];
}

export function AgentEditorForm(props: { context: AgentEditorContext }) {
  const { context } = props;

  return (
    <div className="workflow-arch-prop-group">
      <AgentTitleField context={context} />
      <AgentMessageField context={context} />
      <AgentProviderField context={context} />
      <AgentModelField context={context} />
      <AgentPresetField context={context} />
      <AgentSystemPromptField context={context} />
      <AgentToolAllowlistField context={context} />
      <AgentMaxTurnsField context={context} />
      <AgentEditorNotes context={context} />
    </div>
  );
}

function AgentTitleField(props: { context: AgentEditorContext }) {
  const { context } = props;

  if (context.editorKind !== 'orchestration') {
    return null;
  }

  return (
    <>
      <label className="workflow-arch-field-label">{context.copy.agentTitle}</label>
      <input
        type="text"
        value={context.selectedNode.agent?.title ?? ''}
        placeholder={context.copy.agentTitlePlaceholder}
        onChange={(event) => context.onUpdateNode(withAgentTitle(context.selectedNode, event.target.value))}
      />
    </>
  );
}

function AgentMessageField(props: { context: AgentEditorContext }) {
  const { context } = props;

  return (
    <>
      <label className="workflow-arch-field-label">{context.copy.agentMessage}</label>
      <WorkflowTemplateEnabledField
        editorKind={context.editorKind}
        mode="textarea"
        rows={7}
        value={context.selectedNode.agent?.message ?? ''}
        placeholder={context.copy.agentMessagePlaceholder}
        onChange={(value) => context.onUpdateNode(withAgentMessage(context.selectedNode, value))}
      />
    </>
  );
}

function AgentProviderField(props: { context: AgentEditorContext }) {
  const { context } = props;
  const providerOptions = buildProviderOptions(
    context.agentRuntimeCatalog,
    runtimeText(context.overrides.provider_name),
    context.copy,
  );

  return (
    <>
      <label className="workflow-arch-field-label">{context.copy.agentProvider}</label>
      <select
        value={runtimeText(context.overrides.provider_name)}
        onChange={(event) => updateAgentProvider(context, event.target.value)}
      >
        <option value="">{providerPlaceholder(context)}</option>
        {providerOptions.map((option) => (
          <option key={option.value} value={option.value} disabled={option.disabled}>{option.label}</option>
        ))}
      </select>
    </>
  );
}

function AgentModelField(props: { context: AgentEditorContext }) {
  const { context } = props;
  const modelOptions = buildModelOptions({
    catalog: context.agentRuntimeCatalog,
    providerName: runtimeText(context.overrides.provider_name),
    currentValue: runtimeText(context.overrides.model),
    copy: context.copy,
  });

  return (
    <>
      <label className="workflow-arch-field-label">{context.copy.agentModel}</label>
      <select
        value={runtimeText(context.overrides.model)}
        onChange={(event) => updateAgentRuntimeOverrides(context, { model: optionalInputValue(event.target.value) })}
      >
        <option value="">{context.copy.agentModelPlaceholder}</option>
        {modelOptions.map((option) => (
          <option key={option.value} value={option.value} disabled={option.disabled}>{option.label}</option>
        ))}
      </select>
    </>
  );
}

function AgentPresetField(props: { context: AgentEditorContext }) {
  const { context } = props;

  if (context.editorKind !== 'orchestration') {
    return null;
  }

  const presetOptions = buildPresetOptions(context.presets, runtimeText(context.overrides.preset_id), context.copy);

  return (
    <>
      <label className="workflow-arch-field-label">{context.copy.agentPreset}</label>
      <select
        data-testid="orchestration-agent-preset-select"
        value={runtimeText(context.overrides.preset_id)}
        onChange={(event) => updateAgentRuntimeOverrides(context, nextPresetRuntimeOverrides({
          current: context.overrides,
          orderedToolNames: context.toolOrder,
          presetID: event.target.value,
          presets: context.presets,
        }))}
      >
        <option value="">{presetPlaceholder(context)}</option>
        {presetOptions.map((option) => (
          <option key={option.value} value={option.value} disabled={option.disabled}>{option.label}</option>
        ))}
      </select>
    </>
  );
}

function AgentSystemPromptField(props: { context: AgentEditorContext }) {
  const { context } = props;

  return (
    <>
      <label className="workflow-arch-field-label">{context.copy.agentSystemPrompt}</label>
      <WorkflowTemplateEnabledField
        editorKind={context.editorKind}
        mode="textarea"
        rows={5}
        value={runtimeText(context.overrides.system_prompt)}
        disabled={isSystemPromptLocked(context)}
        placeholder={context.copy.agentSystemPromptPlaceholder}
        onChange={(value) => updateAgentRuntimeOverrides(context, { system_prompt: optionalSystemPrompt(value) })}
      />
    </>
  );
}

function updateAgentProvider(context: AgentEditorContext, providerName: string): void {
  context.onUpdateNode(withAgentRuntimeOverrides(
    context.selectedNode,
    nextProviderOverrides(context.overrides, providerName, context.agentRuntimeCatalog),
  ));
}

function updateAgentRuntimeOverrides(
  context: AgentEditorContext,
  overrides: TaskRuntimeOverrides | undefined,
): void {
  context.onUpdateNode(withAgentRuntimeOverrides(context.selectedNode, overrides));
}

function nextPresetRuntimeOverrides(
  options: NextPresetRuntimeOverridesOptions,
): TaskRuntimeOverrides | undefined {
  const { current, orderedToolNames, presetID, presets } = options;

  if (presetID.trim().length === 0) {
    return nextRuntimeOverrides(current, { preset_id: undefined });
  }

  const preset = presets.find((item) => item.id === presetID);
  if (!preset) {
    return nextRuntimeOverrides(current, { preset_id: presetID });
  }

  return applyPresetOverride(current, preset, orderedToolNames);
}

function runtimeText(value: string | undefined): string {
  if (value === undefined) {
    return '';
  }
  return value;
}

function optionalInputValue(value: string): string | undefined {
  if (value.length === 0) {
    return undefined;
  }
  return value;
}

function optionalSystemPrompt(value: string): string | undefined {
  if (value.trim().length === 0) {
    return undefined;
  }
  return value;
}

function providerPlaceholder(context: AgentEditorContext): string {
  if (context.agentRuntimeLoading) {
    return context.copy.agentProviderLoading;
  }
  return context.copy.agentProviderPlaceholder;
}

function presetPlaceholder(context: AgentEditorContext): string {
  if (context.presetLoading) {
    return context.copy.agentPresetLoading;
  }
  return context.copy.agentPresetPlaceholder;
}

function isSystemPromptLocked(context: AgentEditorContext): boolean {
  if (context.editorKind !== 'orchestration') {
    return false;
  }
  return runtimeText(context.overrides.preset_id).trim().length > 0;
}
