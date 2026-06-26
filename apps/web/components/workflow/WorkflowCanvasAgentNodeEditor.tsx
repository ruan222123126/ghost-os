'use client';

import { WorkflowVariableAutocompleteField } from '@/components/workflow/WorkflowVariableAutocompleteField';
import {
  applyPresetOverride,
  buildModelOptions,
  buildPresetOptions,
  buildProviderOptions,
  buildToolOptions,
  nextProviderOverrides,
  nextRuntimeOverrides,
  parseOptionalPositiveInteger,
  toggleToolOverride,
  toolOrderForEditor,
} from '@/components/workflow/workflowAgentNodeEditorState';
import { useWebLocale } from '@/lib/i18n/provider';
import type { PresetPayload, TaskRuntimeOverrides } from '@/lib/types';
import {
  cloneOrchestrationTaskRuntimeOverrides,
  cloneWorkflowTaskRuntimeOverrides,
  type WorkflowAgentRuntimeCatalog,
  type WorkflowCanvasNodeDraft,
  withAgentMessage,
  withAgentTitle,
  withAgentRuntimeOverrides,
} from '@/lib/workflow-editor';

interface WorkflowCanvasAgentNodeEditorProps {
  editorKind: 'workflow' | 'orchestration';
  selectedNode: WorkflowCanvasNodeDraft;
  agentRuntimeCatalog?: WorkflowAgentRuntimeCatalog;
  agentRuntimeLoading: boolean;
  agentRuntimeError: string;
  presets?: PresetPayload[];
  presetLoading?: boolean;
  presetError?: string;
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
}

export function WorkflowCanvasAgentNodeEditor(props: WorkflowCanvasAgentNodeEditorProps) {
  const { copy } = useWebLocale();
  const {
    agentRuntimeCatalog,
    agentRuntimeError,
    agentRuntimeLoading,
    editorKind,
    presetError = '',
    presetLoading = false,
    presets = [],
    selectedNode,
    onUpdateNode,
  } = props;
  const overrides = editorKind === 'orchestration'
    ? cloneOrchestrationTaskRuntimeOverrides(selectedNode.agent?.runtime_overrides) ?? {}
    : cloneWorkflowTaskRuntimeOverrides(selectedNode.agent?.runtime_overrides) ?? {};
  const toolOptions = buildToolOptions({
    tools: agentRuntimeCatalog?.tools ?? [],
    selectedNames: overrides.tool_allowlist ?? [],
    copy: copy.workflow,
    showAllTools: editorKind === 'orchestration',
  });
  const toolOrder = toolOrderForEditor(
    agentRuntimeCatalog?.tools ?? [],
    editorKind === 'orchestration',
  );
  const providerOptions = buildProviderOptions(agentRuntimeCatalog, overrides.provider_name ?? '', copy.workflow);
  const modelOptions = buildModelOptions({
    catalog: agentRuntimeCatalog,
    providerName: overrides.provider_name ?? '',
    currentValue: overrides.model ?? '',
    copy: copy.workflow,
  });
  const presetOptions = buildPresetOptions(presets, overrides.preset_id ?? '', copy.workflow);
  const systemPromptLocked = editorKind === 'orchestration' && (overrides.preset_id?.trim().length ?? 0) > 0;
  const toolStatusText = agentRuntimeLoading
    ? copy.workflow.agentToolsLoading
    : (toolOptions.length === 0 ? copy.workflow.agentNoTools : '');

  return (
    <div className="workflow-arch-prop-group">
      {editorKind === 'orchestration' ? (
        <>
          <label className="workflow-arch-field-label">{copy.workflow.agentTitle}</label>
          <input
            type="text"
            value={selectedNode.agent?.title ?? ''}
            placeholder={copy.workflow.agentTitlePlaceholder}
            onChange={(event) => onUpdateNode(withAgentTitle(selectedNode, event.target.value))}
          />
        </>
      ) : null}
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
      {editorKind === 'orchestration' ? (
        <>
          <label className="workflow-arch-field-label">{copy.workflow.agentPreset}</label>
          <select
            data-testid="orchestration-agent-preset-select"
            value={overrides.preset_id ?? ''}
            onChange={(event) => onUpdateNode(withAgentRuntimeOverrides(
              selectedNode,
              nextPresetRuntimeOverrides(overrides, event.target.value, presets, toolOrder),
            ))}
          >
            <option value="">{presetLoading ? copy.workflow.agentPresetLoading : copy.workflow.agentPresetPlaceholder}</option>
            {presetOptions.map((option) => (
              <option key={option.value} value={option.value} disabled={option.disabled}>{option.label}</option>
            ))}
          </select>
        </>
      ) : null}
      <label className="workflow-arch-field-label">{copy.workflow.agentSystemPrompt}</label>
      <TemplateEnabledField
        editorKind={editorKind}
        mode="textarea"
        rows={5}
        value={overrides.system_prompt ?? ''}
        disabled={systemPromptLocked}
        placeholder={copy.workflow.agentSystemPromptPlaceholder}
        onChange={(value) => onUpdateNode(withAgentRuntimeOverrides(selectedNode, nextRuntimeOverrides(overrides, {
          system_prompt: value.trim().length > 0 ? value : undefined,
        })))}
      />
      <label className="workflow-arch-field-label">{copy.workflow.agentToolAllowlist}</label>
      {toolStatusText ? <div className="workflow-arch-field-note">{toolStatusText}</div> : null}
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
                toolOrder,
              )))}
            />
            <span>{option.label}</span>
          </label>
        ))}
      </div>
      {editorKind === 'workflow' ? (
        <>
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
        </>
      ) : null}
      {agentRuntimeError.trim().length > 0 ? <p className="workflow-arch-field-note">{agentRuntimeError}</p> : null}
      {presetError.trim().length > 0 ? <p className="workflow-arch-field-note">{presetError}</p> : null}
      {editorKind === 'workflow' ? <p className="workflow-arch-field-note">{copy.workflow.runtimeVariableHint}</p> : null}
    </div>
  );
}

function nextPresetRuntimeOverrides(
  current: TaskRuntimeOverrides,
  presetID: string,
  presets: PresetPayload[],
  orderedToolNames: string[],
) {
  if (presetID.trim().length === 0) {
    return nextRuntimeOverrides(current, { preset_id: undefined });
  }
  const preset = presets.find((item) => item.id === presetID);
  if (!preset) {
    return nextRuntimeOverrides(current, { preset_id: presetID });
  }
  return applyPresetOverride(current, preset, orderedToolNames);
}

function TemplateEnabledField(props: {
  editorKind: 'workflow' | 'orchestration';
  mode: 'input' | 'textarea';
  value: string;
  rows?: number;
  disabled?: boolean;
  placeholder?: string;
  onChange: (value: string) => void;
}) {
  const { editorKind, mode, value, rows, disabled = false, placeholder, onChange } = props;
  if (editorKind === 'workflow') {
    return (
      <WorkflowVariableAutocompleteField
        mode={mode}
        value={value}
        rows={rows}
        disabled={disabled}
        placeholder={placeholder}
        onChange={onChange}
      />
    );
  }
  return mode === 'textarea'
    ? <textarea rows={rows} disabled={disabled} value={value} placeholder={placeholder} onChange={(event) => onChange(event.target.value)} />
    : <input type="text" disabled={disabled} value={value} placeholder={placeholder} onChange={(event) => onChange(event.target.value)} />;
}
