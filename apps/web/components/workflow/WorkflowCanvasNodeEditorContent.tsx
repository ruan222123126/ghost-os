'use client';

import { IfEditor, LoopEditor } from '@/components/workflow/WorkflowCanvasConditionalEditor';
import { WorkflowCanvasAgentNodeEditor } from '@/components/workflow/WorkflowCanvasAgentNodeEditor';
import type { WebLocale } from '@/lib/i18n/locale';
import { WorkflowCanvasToolNodeEditor } from '@/components/workflow/WorkflowCanvasToolNodeEditor';
import { WorkflowVariableAutocompleteField } from '@/components/workflow/WorkflowVariableAutocompleteField';
import { useWebLocale } from '@/lib/i18n/provider';
import type { PresetPayload } from '@/lib/types';
import { defaultOrchestrationGroupSharedContext } from '@/lib/orchestration-editor/groupDefaults';
import {
  type WorkflowAgentRuntimeCatalog,
  type WorkflowEditorKind,
  type WorkflowCanvasDraft,
  type WorkflowCanvasNodeDraft,
  withLLMPrompt,
  withLLMSystemPrompt,
  withGroupNode,
} from '@/lib/workflow-editor';

interface WorkflowCanvasNodeEditorContentProps {
  editorKind: WorkflowEditorKind;
  selectedNode: WorkflowCanvasNodeDraft;
  draft?: WorkflowCanvasDraft;
  agentRuntimeCatalog?: WorkflowAgentRuntimeCatalog;
  agentRuntimeLoading: boolean;
  agentRuntimeError: string;
  presets?: PresetPayload[];
  presetLoading: boolean;
  presetError: string;
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
}

export function WorkflowCanvasNodeEditorContent(props: WorkflowCanvasNodeEditorContentProps) {
  const {
    agentRuntimeCatalog,
    agentRuntimeError,
    agentRuntimeLoading,
    editorKind,
    presetError,
    presetLoading,
    presets,
    draft,
    selectedNode,
    onUpdateNode,
  } = props;
  if (selectedNode.type === 'start') {
    return <StartEditor />;
  }
  if (selectedNode.type === 'llm') {
    return <LLMEditor editorKind={editorKind} selectedNode={selectedNode} onUpdateNode={onUpdateNode} />;
  }
  if (selectedNode.type === 'if') {
    return <IfEditor editorKind={editorKind} selectedNode={selectedNode} onUpdateNode={onUpdateNode} />;
  }
  if (selectedNode.type === 'loop') {
    return <LoopEditor editorKind={editorKind} selectedNode={selectedNode} onUpdateNode={onUpdateNode} />;
  }
  if (selectedNode.type === 'tool') {
    return <WorkflowCanvasToolNodeEditor editorKind={editorKind} selectedNode={selectedNode} onUpdateNode={onUpdateNode} />;
  }
  if (selectedNode.type === 'agent') {
    return (
      <WorkflowCanvasAgentNodeEditor
        editorKind={editorKind}
        selectedNode={selectedNode}
        agentRuntimeCatalog={agentRuntimeCatalog}
        agentRuntimeLoading={agentRuntimeLoading}
        agentRuntimeError={agentRuntimeError}
        presets={presets}
        presetLoading={presetLoading}
        presetError={presetError}
        onUpdateNode={onUpdateNode}
      />
    );
  }
  if (selectedNode.type === 'group') {
    return <GroupEditor draft={draft} selectedNode={selectedNode} onUpdateNode={onUpdateNode} />;
  }
  return <EndEditor />;
}

function StartEditor() {
  const { locale } = useWebLocale();
  return (
    <div className="workflow-arch-prop-group">
      <p className="workflow-arch-field-note">{variablesDisabledText(locale)}</p>
    </div>
  );
}

function LLMEditor(props: {
  editorKind: WorkflowEditorKind;
  selectedNode: WorkflowCanvasNodeDraft;
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
}) {
  const { editorKind, selectedNode, onUpdateNode } = props;
  const { copy } = useWebLocale();

  return (
    <div className="workflow-arch-prop-group">
      <label className="workflow-arch-field-label workflow-arch-field-label--centered">{copy.workflow.llmModelConfiguration}</label>
      <select>
        <option>Engine: GPT-4o-Mini</option>
        <option>Engine: Claude 3.5 Sonnet</option>
        <option>Engine: Llama 3 70B</option>
      </select>
      <label className="workflow-arch-field-label">{copy.workflow.llmSystemDirectives}</label>
      <TemplateEnabledField
        editorKind={editorKind}
        mode="textarea"
        rows={6}
        value={selectedNode.llm?.system_prompt ?? ''}
        placeholder={copy.workflow.llmSystemPromptPlaceholder}
        onChange={(value) => onUpdateNode(withLLMSystemPrompt(selectedNode, value))}
      />
      <label className="workflow-arch-field-label">{copy.workflow.llmPrompt}</label>
      <TemplateEnabledField
        editorKind={editorKind}
        mode="textarea"
        rows={8}
        value={selectedNode.llm?.prompt ?? ''}
        placeholder={copy.workflow.llmPromptPlaceholder}
        onChange={(value) => onUpdateNode(withLLMPrompt(selectedNode, value))}
      />
      {editorKind === 'workflow' ? <p className="workflow-arch-field-note">{copy.workflow.runtimeVariableHint}</p> : null}
    </div>
  );
}

function EndEditor() {
  const { copy } = useWebLocale();
  return (
    <div className="workflow-arch-end-tip">
      <p>{copy.workflow.terminus}</p>
    </div>
  );
}

function GroupEditor(props: {
  draft?: WorkflowCanvasDraft;
  selectedNode: WorkflowCanvasNodeDraft;
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
}) {
  const { copy, locale } = useWebLocale();
  const { draft, selectedNode, onUpdateNode } = props;
  const memberOptions = resolveGroupMemberOptions(draft, selectedNode.id);
  const speakingMode = selectedNode.group?.speaking_mode ?? 'sequential';
  return (
    <div className="workflow-arch-prop-group">
      <label className="workflow-arch-field-label">{copy.workflow.groupTitle}</label>
      <input
        type="text"
        value={selectedNode.group?.title ?? ''}
        placeholder={copy.workflow.groupTitlePlaceholder}
        onChange={(event) => onUpdateNode(withGroupNode(selectedNode, { title: event.target.value }))}
      />
      <label className="workflow-arch-field-label">{groupSharedContextLabel(locale)}</label>
      <textarea
        rows={7}
        value={selectedNode.group?.shared_context ?? ''}
        placeholder={defaultOrchestrationGroupSharedContext(locale)}
        onChange={(event) => onUpdateNode(withGroupNode(selectedNode, { shared_context: event.target.value }))}
      />
      <p className="workflow-arch-field-note">{groupSharedContextNote(locale)}</p>
      <label className="workflow-arch-field-label">{copy.workflow.groupSpeakingMode}</label>
      <select
        value={speakingMode}
        onChange={(event) => onUpdateNode(withGroupNode(selectedNode, {
          speaking_mode: event.target.value as 'sequential' | 'parallel' | 'owner',
          owner_agent_id: event.target.value === 'owner' ? (selectedNode.group?.owner_agent_id ?? '') : '',
        }))}
      >
        <option value="sequential">{copy.workflow.groupSpeakingModeSequential}</option>
        <option value="parallel">{copy.workflow.groupSpeakingModeParallel}</option>
        <option value="owner">{copy.workflow.groupSpeakingModeOwner}</option>
      </select>
      {speakingMode === 'owner' ? (
        <>
          <label className="workflow-arch-field-label">{copy.workflow.groupOwnerAgent}</label>
          <select
            value={selectedNode.group?.owner_agent_id ?? ''}
            onChange={(event) => onUpdateNode(withGroupNode(selectedNode, { owner_agent_id: event.target.value }))}
          >
            <option value="">{copy.workflow.groupOwnerAgentPlaceholder}</option>
            {memberOptions.map((item) => (
              <option key={item.id} value={item.id}>{item.label}</option>
            ))}
          </select>
          <p className="workflow-arch-field-note">{copy.workflow.groupOwnerDispatchNote}</p>
        </>
      ) : null}
      <label className="workflow-arch-field-label">{copy.workflow.groupMaxRounds}</label>
      <input
        type="number"
        min={1}
        step={1}
        value={selectedNode.group?.max_rounds ?? 1}
        onChange={(event) => onUpdateNode(withGroupNode(selectedNode, { max_rounds: Number.parseInt(event.target.value, 10) || 0 }))}
      />
      {speakingMode === 'owner' ? (
        <p className="workflow-arch-field-note">{copy.workflow.groupOwnerMaxRoundsHint}</p>
      ) : null}
    </div>
  );
}

function resolveGroupMemberOptions(draft: WorkflowCanvasDraft | undefined, groupNodeID: string): Array<{ id: string; label: string }> {
  if (!draft) {
    return [];
  }
  const nodeByID = new Map(draft.nodes.map((node) => [node.id, node] as const));
  return draft.edges
    .filter((edge) => edge.kind === 'member' && edge.to_node_id === groupNodeID)
    .map((edge) => nodeByID.get(edge.from_node_id))
    .filter((node): node is WorkflowCanvasNodeDraft => node?.type === 'agent')
    .map((node) => ({
      id: node.id,
      label: `${node.agent?.title?.trim() || node.id} (${node.id})`,
    }));
}

function variablesDisabledText(locale: WebLocale): string {
  if (locale === 'zh-CN') {
    return '通用工作流变量已暂时关闭；当前仅保留 screen_control 编排内 find_icon 返回坐标的专用引用。';
  }
  return 'General workflow variables are temporarily disabled. Only the screen_control find_icon coordinate reference remains.';
}

function groupSharedContextLabel(locale: WebLocale): string {
  if (locale === 'zh-CN') {
    return '群主初始提示词';
  }
  return 'Group Owner Initial Prompt';
}

function groupSharedContextNote(locale: WebLocale): string {
  if (locale === 'zh-CN') {
    return '这段内容会作为群主开场提示和群共享上下文注入给所有成员。';
  }
  return 'This text is injected to every member as the group owner opening prompt and shared context.';
}

function TemplateEnabledField(props: {
  editorKind: WorkflowEditorKind;
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
  if (mode === 'textarea') {
    return (
      <textarea
        rows={rows}
        value={value}
        placeholder={placeholder}
        onChange={(event) => onChange(event.target.value)}
      />
    );
  }
  return (
    <input
      type="text"
      value={value}
      placeholder={placeholder}
      onChange={(event) => onChange(event.target.value)}
    />
  );
}
