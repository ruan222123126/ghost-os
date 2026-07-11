'use client';

import type { ReactNode } from 'react';
import { IfEditor, LoopEditor } from '@/components/workflow/WorkflowCanvasConditionalEditor';
import { WorkflowCanvasAgentNodeEditor } from '@/components/workflow/WorkflowCanvasAgentNodeEditor';
import { WorkflowCanvasGroupNodeEditor } from '@/components/workflow/WorkflowCanvasGroupNodeEditor';
import type { WebLocale } from '@/lib/i18n/locale';
import { WorkflowCanvasToolNodeEditor } from '@/components/workflow/WorkflowCanvasToolNodeEditor';
import { WorkflowTemplateEnabledField } from '@/components/workflow/WorkflowTemplateEnabledField';
import { useWebLocale } from '@/lib/i18n/provider';
import type { PresetPayload } from '@/lib/types';
import {
  type WorkflowAgentRuntimeCatalog,
  type WorkflowEditorKind,
  type WorkflowCanvasDraft,
  type WorkflowCanvasNodeDraft,
  withLLMPrompt,
  withLLMSystemPrompt,
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

type NodeEditorRenderer = (props: WorkflowCanvasNodeEditorContentProps) => ReactNode;

const NODE_EDITOR_RENDERERS: Partial<Record<WorkflowCanvasNodeDraft['type'], NodeEditorRenderer>> = {
  agent: renderAgentEditor,
  group: renderGroupEditor,
  if: renderIfEditor,
  llm: renderLLMEditor,
  loop: renderLoopEditor,
  start: renderStartEditor,
  tool: renderToolEditor,
};

export function WorkflowCanvasNodeEditorContent(props: WorkflowCanvasNodeEditorContentProps) {
  const renderEditor = NODE_EDITOR_RENDERERS[props.selectedNode.type] ?? renderEndEditor;
  return renderEditor(props);
}

function renderStartEditor() {
  return <StartEditor />;
}

function renderLLMEditor(props: WorkflowCanvasNodeEditorContentProps) {
  return (
    <LLMEditor
      editorKind={props.editorKind}
      selectedNode={props.selectedNode}
      onUpdateNode={props.onUpdateNode}
    />
  );
}

function renderIfEditor(props: WorkflowCanvasNodeEditorContentProps) {
  return <IfEditor editorKind={props.editorKind} selectedNode={props.selectedNode} onUpdateNode={props.onUpdateNode} />;
}

function renderLoopEditor(props: WorkflowCanvasNodeEditorContentProps) {
  return <LoopEditor editorKind={props.editorKind} selectedNode={props.selectedNode} onUpdateNode={props.onUpdateNode} />;
}

function renderToolEditor(props: WorkflowCanvasNodeEditorContentProps) {
  return (
    <WorkflowCanvasToolNodeEditor
      editorKind={props.editorKind}
      selectedNode={props.selectedNode}
      onUpdateNode={props.onUpdateNode}
    />
  );
}

function renderAgentEditor(props: WorkflowCanvasNodeEditorContentProps) {
  const {
    agentRuntimeCatalog,
    agentRuntimeError,
    agentRuntimeLoading,
    editorKind,
    presetError,
    presetLoading,
    presets,
    selectedNode,
    onUpdateNode,
  } = props;

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

function renderGroupEditor(props: WorkflowCanvasNodeEditorContentProps) {
  return (
    <WorkflowCanvasGroupNodeEditor
      draft={props.draft}
      selectedNode={props.selectedNode}
      onUpdateNode={props.onUpdateNode}
    />
  );
}

function renderEndEditor() {
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
      <LLMModelField label={copy.workflow.llmModelConfiguration} />
      <LLMSystemPromptField
        editorKind={editorKind}
        selectedNode={selectedNode}
        label={copy.workflow.llmSystemDirectives}
        placeholder={copy.workflow.llmSystemPromptPlaceholder}
        onUpdateNode={onUpdateNode}
      />
      <LLMPromptField
        editorKind={editorKind}
        selectedNode={selectedNode}
        label={copy.workflow.llmPrompt}
        placeholder={copy.workflow.llmPromptPlaceholder}
        onUpdateNode={onUpdateNode}
      />
      <WorkflowRuntimeVariableHint editorKind={editorKind} text={copy.workflow.runtimeVariableHint} />
    </div>
  );
}

function LLMModelField(props: { label: string }) {
  return (
    <>
      <label className="workflow-arch-field-label workflow-arch-field-label--centered">{props.label}</label>
      <select>
        <option>Engine: GPT-4o-Mini</option>
        <option>Engine: Claude 3.5 Sonnet</option>
        <option>Engine: Llama 3 70B</option>
      </select>
    </>
  );
}

function LLMSystemPromptField(props: {
  editorKind: WorkflowEditorKind;
  selectedNode: WorkflowCanvasNodeDraft;
  label: string;
  placeholder: string;
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
}) {
  return (
    <>
      <label className="workflow-arch-field-label">{props.label}</label>
      <WorkflowTemplateEnabledField
        editorKind={props.editorKind}
        mode="textarea"
        rows={6}
        value={props.selectedNode.llm?.system_prompt ?? ''}
        placeholder={props.placeholder}
        onChange={(value) => props.onUpdateNode(withLLMSystemPrompt(props.selectedNode, value))}
      />
    </>
  );
}

function LLMPromptField(props: {
  editorKind: WorkflowEditorKind;
  selectedNode: WorkflowCanvasNodeDraft;
  label: string;
  placeholder: string;
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
}) {
  return (
    <>
      <label className="workflow-arch-field-label">{props.label}</label>
      <WorkflowTemplateEnabledField
        editorKind={props.editorKind}
        mode="textarea"
        rows={8}
        value={props.selectedNode.llm?.prompt ?? ''}
        placeholder={props.placeholder}
        onChange={(value) => props.onUpdateNode(withLLMPrompt(props.selectedNode, value))}
      />
    </>
  );
}

function WorkflowRuntimeVariableHint(props: { editorKind: WorkflowEditorKind; text: string }) {
  if (props.editorKind !== 'workflow') {
    return null;
  }
  return <p className="workflow-arch-field-note">{props.text}</p>;
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
