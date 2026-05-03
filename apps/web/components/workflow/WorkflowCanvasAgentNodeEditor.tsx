'use client';

import { WorkflowVariableAutocompleteField } from '@/components/workflow/WorkflowVariableAutocompleteField';
import { useWebLocale } from '@/lib/i18n/provider';
import type { TaskRuntimeOverrides, ToolPayload } from '@/lib/types';
import {
  cloneWorkflowTaskRuntimeOverrides,
  enabledWorkflowAgentToolNames,
  type WorkflowAgentRuntimeCatalog,
  type WorkflowCanvasNodeDraft,
  withAgentMessage,
  withAgentRuntimeOverrides,
} from '@/lib/workflow-editor';

interface WorkflowCanvasAgentNodeEditorProps {
  editorKind: 'workflow' | 'orchestration';
  selectedNode: WorkflowCanvasNodeDraft;
  agentRuntimeCatalog?: WorkflowAgentRuntimeCatalog;
  agentRuntimeLoading: boolean;
  agentRuntimeError: string;
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
}

interface SelectOption {
  disabled: boolean;
  label: string;
  value: string;
}

export function WorkflowCanvasAgentNodeEditor(props: WorkflowCanvasAgentNodeEditorProps) {
  const { copy } = useWebLocale();
  const {
    agentRuntimeCatalog,
    agentRuntimeError,
    agentRuntimeLoading,
    editorKind,
    selectedNode,
    onUpdateNode,
  } = props;
  const overrides = cloneWorkflowTaskRuntimeOverrides(selectedNode.agent?.runtime_overrides) ?? {};
  const providerOptions = buildProviderOptions(agentRuntimeCatalog, overrides.provider_name ?? '', copy.workflow);
  const modelOptions = buildModelOptions(agentRuntimeCatalog, overrides.provider_name ?? '', overrides.model ?? '', copy.workflow);
  const toolOptions = buildToolOptions(agentRuntimeCatalog?.tools ?? [], overrides.tool_allowlist ?? [], copy.workflow);
  const enabledToolNames = enabledWorkflowAgentToolNames(agentRuntimeCatalog?.tools ?? []);

  return (
    <div className="workflow-arch-prop-group">
      <label className="workflow-arch-field-label">{copy.workflow.agentMessage}</label>
      <TemplateEnabledField
        editorKind={editorKind}
        mode="textarea"
        rows={7}
        value={selectedNode.agent?.message ?? ''}
        placeholder={copy.workflow.agentMessagePlaceholder}
        onChange={(value) => onUpdateNode(withAgentMessage(selectedNode, value))}
      />
      <label className="workflow-arch-field-label">{copy.workflow.agentProvider}</label>
      <select
        value={overrides.provider_name ?? ''}
        onChange={(event) => onUpdateNode(withAgentRuntimeOverrides(selectedNode, nextProviderOverrides(
          overrides,
          event.target.value,
          agentRuntimeCatalog,
        )))}
      >
        <option value="">{agentRuntimeLoading ? copy.workflow.agentProviderLoading : copy.workflow.agentProviderPlaceholder}</option>
        {providerOptions.map((option) => (
          <option key={option.value} value={option.value} disabled={option.disabled}>{option.label}</option>
        ))}
      </select>
      <label className="workflow-arch-field-label">{copy.workflow.agentModel}</label>
      <select
        value={overrides.model ?? ''}
        onChange={(event) => onUpdateNode(withAgentRuntimeOverrides(selectedNode, nextRuntimeOverrides(overrides, {
          model: event.target.value || undefined,
        })))}
      >
        <option value="">{copy.workflow.agentModelPlaceholder}</option>
        {modelOptions.map((option) => (
          <option key={option.value} value={option.value} disabled={option.disabled}>{option.label}</option>
        ))}
      </select>
      <label className="workflow-arch-field-label">{copy.workflow.agentSystemPrompt}</label>
      <TemplateEnabledField
        editorKind={editorKind}
        mode="textarea"
        rows={5}
        value={overrides.system_prompt ?? ''}
        placeholder={copy.workflow.agentSystemPromptPlaceholder}
        onChange={(value) => onUpdateNode(withAgentRuntimeOverrides(selectedNode, nextRuntimeOverrides(overrides, {
          system_prompt: value.trim().length > 0 ? value : undefined,
        })))}
      />
      <label className="workflow-arch-field-label">{copy.workflow.agentToolAllowlist}</label>
      <div className="workflow-arch-field-note">
        {agentRuntimeLoading ? copy.workflow.agentToolsLoading : copy.workflow.agentNoTools}
      </div>
      <div className="max-h-48 overflow-y-auto rounded-[18px] border border-[#E5E5E5] bg-white px-3 py-2">
        {toolOptions.map((option) => (
          <label key={option.value} className={`flex items-center gap-2 py-1 text-[13px] ${option.disabled ? 'text-[#A3A3A3]' : 'text-[#171717]'}`}>
            <input
              type="checkbox"
              checked={(overrides.tool_allowlist ?? []).includes(option.value)}
              disabled={option.disabled}
              onChange={() => onUpdateNode(withAgentRuntimeOverrides(selectedNode, toggleToolOverride(
                overrides,
                option.value,
                enabledToolNames,
              )))}
            />
            <span>{option.label}</span>
          </label>
        ))}
      </div>
      <label className="workflow-arch-field-label">{copy.workflow.agentMaxTurns}</label>
      <input
        type="number"
        min={1}
        step={1}
        value={overrides.max_turns === undefined ? '' : String(overrides.max_turns)}
        onChange={(event) => onUpdateNode(withAgentRuntimeOverrides(selectedNode, nextRuntimeOverrides(overrides, {
          max_turns: parseOptionalPositiveInteger(event.target.value),
        })))}
      />
      {agentRuntimeError.trim().length > 0 ? <p className="workflow-arch-field-note">{agentRuntimeError}</p> : null}
      {editorKind === 'workflow' ? <p className="workflow-arch-field-note">{copy.workflow.runtimeVariableHint}</p> : null}
    </div>
  );
}

function buildProviderOptions(
  catalog: WorkflowAgentRuntimeCatalog | undefined,
  currentValue: string,
  copy: ReturnType<typeof useWebLocale>['copy']['workflow'],
): SelectOption[] {
  const mapped = (catalog?.providers ?? [])
    .slice()
    .sort((left, right) => left.name.localeCompare(right.name))
    .map((provider) => ({
      value: provider.name,
      label: `${provider.name} (${provider.type})`,
      disabled: false,
    }));
  return withCurrentDisabledOption(mapped, currentValue, copy.agentProviderDisabled, copy.agentProviderUnavailable);
}

function buildModelOptions(
  catalog: WorkflowAgentRuntimeCatalog | undefined,
  providerName: string,
  currentValue: string,
  copy: ReturnType<typeof useWebLocale>['copy']['workflow'],
): SelectOption[] {
  const provider = (catalog?.providers ?? []).find((item) => item.name === providerName);
  const mapped = (provider?.models ?? []).map((model) => ({
    value: model,
    label: model,
    disabled: false,
  }));
  return withCurrentDisabledOption(mapped, currentValue, copy.agentModelDisabled, copy.agentModelUnavailable);
}

function buildToolOptions(
  tools: ToolPayload[],
  selectedNames: string[],
  copy: ReturnType<typeof useWebLocale>['copy']['workflow'],
): SelectOption[] {
  const mapped = tools
    .filter((tool) => tool.enabled)
    .slice()
    .sort((left, right) => left.name.localeCompare(right.name))
    .map((tool) => ({ value: tool.name, label: tool.name, disabled: false }));
  return selectedNames.reduce((options, name) => {
    return withCurrentDisabledOption(options, name, copy.agentToolDisabled, copy.agentToolUnavailable);
  }, mapped);
}

function withCurrentDisabledOption(
  options: SelectOption[],
  currentValue: string,
  disabledLabel: (name: string) => string,
  unavailableLabel: (name: string) => string,
): SelectOption[] {
  const normalized = currentValue.trim();
  if (normalized.length === 0 || options.some((option) => option.value === normalized)) {
    return options;
  }
  return [{
    value: normalized,
    label: unavailableLabel(normalized),
    disabled: true,
  }, ...options];
}

function nextProviderOverrides(
  current: TaskRuntimeOverrides,
  providerName: string,
  catalog: WorkflowAgentRuntimeCatalog | undefined,
): TaskRuntimeOverrides | undefined {
  const provider = catalog?.providers.find((item) => item.name === providerName);
  const nextModel = (provider?.models ?? []).includes(current.model ?? '') ? current.model : undefined;
  return nextRuntimeOverrides(current, {
    provider_name: providerName || undefined,
    model: nextModel,
  });
}

function toggleToolOverride(
  current: TaskRuntimeOverrides,
  toolName: string,
  enabledToolNames: string[],
): TaskRuntimeOverrides | undefined {
  const next = new Set(current.tool_allowlist ?? []);
  if (next.has(toolName)) {
    next.delete(toolName);
  } else {
    next.add(toolName);
  }
  const ordered = enabledToolNames.filter((name) => next.has(name));
  return nextRuntimeOverrides(current, {
    tool_allowlist_only: true,
    tool_allowlist: ordered,
  });
}

function nextRuntimeOverrides(
  current: TaskRuntimeOverrides,
  patch: Partial<TaskRuntimeOverrides>,
): TaskRuntimeOverrides | undefined {
  const next = {
    ...current,
    ...patch,
  };
  if (next.provider_name === undefined) {
    next.model = undefined;
  }
  if (next.tool_allowlist_only === true && next.tool_allowlist === undefined) {
    next.tool_allowlist = [];
  }
  if (!hasMeaningfulRuntimeOverrides(next)) {
    return undefined;
  }
  return next;
}

function hasMeaningfulRuntimeOverrides(overrides: TaskRuntimeOverrides): boolean {
  return Boolean(
    overrides.provider_name
    || overrides.model
    || overrides.system_prompt
    || overrides.tool_allowlist_only
    || overrides.tool_allowlist?.length
    || overrides.max_turns,
  );
}

function parseOptionalPositiveInteger(raw: string): number | undefined {
  const value = raw.trim();
  if (value.length === 0) {
    return undefined;
  }
  const parsed = Number.parseInt(value, 10);
  return Number.isFinite(parsed) ? parsed : undefined;
}

function TemplateEnabledField(props: {
  editorKind: 'workflow' | 'orchestration';
  mode: 'input' | 'textarea';
  value: string;
  rows?: number;
  placeholder?: string;
  onChange: (value: string) => void;
}) {
  const { editorKind, mode, value, rows, placeholder, onChange } = props;
  if (editorKind === 'workflow') {
    return (
      <WorkflowVariableAutocompleteField
        mode={mode}
        value={value}
        rows={rows}
        placeholder={placeholder}
        onChange={onChange}
      />
    );
  }
  return mode === 'textarea'
    ? <textarea rows={rows} value={value} placeholder={placeholder} onChange={(event) => onChange(event.target.value)} />
    : <input type="text" value={value} placeholder={placeholder} onChange={(event) => onChange(event.target.value)} />;
}
