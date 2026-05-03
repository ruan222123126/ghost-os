'use client';

import { IfEditor, LoopEditor } from '@/components/workflow/WorkflowCanvasConditionalEditor';
import { WorkflowCanvasAgentNodeEditor } from '@/components/workflow/WorkflowCanvasAgentNodeEditor';
import type { WebLocale } from '@/lib/i18n/locale';
import { WorkflowCanvasToolNodeEditor } from '@/components/workflow/WorkflowCanvasToolNodeEditor';
import { WorkflowVariableAutocompleteField } from '@/components/workflow/WorkflowVariableAutocompleteField';
import { useWebLocale } from '@/lib/i18n/provider';
import {
  type WorkflowAgentRuntimeCatalog,
  type WorkflowEditorKind,
  type WorkflowCanvasNodeDraft,
  withLLMPrompt,
  withLLMSystemPrompt,
} from '@/lib/workflow-editor';

interface WorkflowCanvasNodeEditorContentProps {
  editorKind: WorkflowEditorKind;
  selectedNode: WorkflowCanvasNodeDraft;
  agentRuntimeCatalog?: WorkflowAgentRuntimeCatalog;
  agentRuntimeLoading: boolean;
  agentRuntimeError: string;
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
}

export function WorkflowCanvasNodeEditorContent(props: WorkflowCanvasNodeEditorContentProps) {
  const {
    agentRuntimeCatalog,
    agentRuntimeError,
    agentRuntimeLoading,
    editorKind,
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
        onUpdateNode={onUpdateNode}
      />
    );
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

function variablesDisabledText(locale: WebLocale): string {
  if (locale === 'zh-CN') {
    return '通用工作流变量已暂时关闭；当前仅保留 screen_control 编排内 find_icon 返回坐标的专用引用。';
  }
  return 'General workflow variables are temporarily disabled. Only the screen_control find_icon coordinate reference remains.';
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
