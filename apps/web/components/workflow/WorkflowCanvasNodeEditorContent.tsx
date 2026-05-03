'use client';

import { IfEditor, LoopEditor } from '@/components/workflow/WorkflowCanvasConditionalEditor';
import { WorkflowCanvasStartVariablesEditor } from '@/components/workflow/WorkflowCanvasStartVariablesEditor';
import { WorkflowCanvasToolNodeEditor } from '@/components/workflow/WorkflowCanvasToolNodeEditor';
import { useWebLocale } from '@/lib/i18n/provider';
import {
  type WorkflowCanvasDraft,
  type WorkflowCanvasNodeDraft,
  type WorkflowEditorKind,
  withAgentMessage,
  withLLMPrompt,
  withLLMSystemPrompt,
} from '@/lib/workflow-editor';
import { WorkflowVariableAutocompleteField } from './WorkflowVariableAutocompleteField';

interface WorkflowCanvasNodeEditorContentProps {
  editorKind: WorkflowEditorKind;
  draft: WorkflowCanvasDraft;
  selectedNode: WorkflowCanvasNodeDraft;
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
}

interface NodeEditorProps {
  editorKind: WorkflowEditorKind;
  draft: WorkflowCanvasDraft;
  selectedNode: WorkflowCanvasNodeDraft;
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
}

export function WorkflowCanvasNodeEditorContent(props: WorkflowCanvasNodeEditorContentProps) {
  const { editorKind, draft, selectedNode, onUpdateNode } = props;
  if (selectedNode.type === 'start') {
    return <WorkflowCanvasStartVariablesEditor selectedNode={selectedNode} onUpdateNode={onUpdateNode} />;
  }
  if (selectedNode.type === 'llm') {
    return <LLMEditor editorKind={editorKind} draft={draft} selectedNode={selectedNode} onUpdateNode={onUpdateNode} />;
  }
  if (selectedNode.type === 'if') {
    return <IfEditor editorKind={editorKind} draft={draft} selectedNode={selectedNode} onUpdateNode={onUpdateNode} />;
  }
  if (selectedNode.type === 'loop') {
    return <LoopEditor selectedNode={selectedNode} onUpdateNode={onUpdateNode} />;
  }
  if (selectedNode.type === 'tool') {
    return (
      <WorkflowCanvasToolNodeEditor
        editorKind={editorKind}
        draft={draft}
        selectedNode={selectedNode}
        onUpdateNode={onUpdateNode}
      />
    );
  }
  if (selectedNode.type === 'agent') {
    return <AgentEditor editorKind={editorKind} draft={draft} selectedNode={selectedNode} onUpdateNode={onUpdateNode} />;
  }
  return <EndEditor />;
}

function LLMEditor(props: NodeEditorProps) {
  const { copy } = useWebLocale();
  const { selectedNode, onUpdateNode } = props;

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
        editorKind={props.editorKind}
        draft={props.draft}
        selectedNode={selectedNode}
        mode="textarea"
        value={selectedNode.llm?.system_prompt ?? ''}
        rows={6}
        placeholder={copy.workflow.llmSystemPromptPlaceholder}
        onChange={(value) => onUpdateNode(withLLMSystemPrompt(selectedNode, value))}
      />
      <label className="workflow-arch-field-label">{copy.workflow.llmPrompt}</label>
      <TemplateEnabledField
        editorKind={props.editorKind}
        draft={props.draft}
        selectedNode={selectedNode}
        mode="textarea"
        value={selectedNode.llm?.prompt ?? ''}
        rows={8}
        placeholder={copy.workflow.llmPromptPlaceholder}
        onChange={(value) => onUpdateNode(withLLMPrompt(selectedNode, value))}
      />
      <p className="workflow-arch-field-note">{copy.workflow.runtimeVariableHint}</p>
    </div>
  );
}

function AgentEditor(props: NodeEditorProps) {
  const { copy } = useWebLocale();
  const { selectedNode, onUpdateNode } = props;
  return (
    <div className="workflow-arch-prop-group">
      <label className="workflow-arch-field-label">{copy.workflow.agentMessage}</label>
      <TemplateEnabledField
        editorKind={props.editorKind}
        draft={props.draft}
        selectedNode={selectedNode}
        mode="textarea"
        value={selectedNode.agent?.message ?? ''}
        rows={9}
        placeholder={copy.workflow.agentMessagePlaceholder}
        onChange={(value) => onUpdateNode(withAgentMessage(selectedNode, value))}
      />
      <p className="workflow-arch-field-note">{copy.workflow.runtimeVariableHint}</p>
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

interface TemplateEnabledFieldProps {
  editorKind: WorkflowEditorKind;
  draft: WorkflowCanvasDraft;
  selectedNode: WorkflowCanvasNodeDraft;
  mode: 'input' | 'textarea';
  value: string;
  rows?: number;
  placeholder?: string;
  onChange: (value: string) => void;
}

function TemplateEnabledField(props: TemplateEnabledFieldProps) {
  const { editorKind, draft, selectedNode, mode, value, rows, placeholder, onChange } = props;
  if (editorKind === 'workflow') {
    return (
      <WorkflowVariableAutocompleteField
        draft={draft}
        selectedNode={selectedNode}
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
