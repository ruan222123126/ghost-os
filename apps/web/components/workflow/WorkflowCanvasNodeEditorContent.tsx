'use client';

import { IfEditor, LoopEditor } from '@/components/workflow/WorkflowCanvasConditionalEditor';
import { WorkflowCanvasStartVariablesEditor } from '@/components/workflow/WorkflowCanvasStartVariablesEditor';
import { WorkflowCanvasToolNodeEditor } from '@/components/workflow/WorkflowCanvasToolNodeEditor';
import { useWebLocale } from '@/lib/i18n/provider';
import {
  type WorkflowCanvasNodeDraft,
  withAgentMessage,
  withLLMPrompt,
  withLLMSystemPrompt,
} from '@/lib/workflow-editor';

interface WorkflowCanvasNodeEditorContentProps {
  selectedNode: WorkflowCanvasNodeDraft;
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
}

interface NodeEditorProps {
  selectedNode: WorkflowCanvasNodeDraft;
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
}

export function WorkflowCanvasNodeEditorContent(props: WorkflowCanvasNodeEditorContentProps) {
  const { selectedNode, onUpdateNode } = props;
  if (selectedNode.type === 'start') {
    return <WorkflowCanvasStartVariablesEditor selectedNode={selectedNode} onUpdateNode={onUpdateNode} />;
  }
  if (selectedNode.type === 'llm') {
    return <LLMEditor selectedNode={selectedNode} onUpdateNode={onUpdateNode} />;
  }
  if (selectedNode.type === 'if') {
    return <IfEditor selectedNode={selectedNode} onUpdateNode={onUpdateNode} />;
  }
  if (selectedNode.type === 'loop') {
    return <LoopEditor selectedNode={selectedNode} onUpdateNode={onUpdateNode} />;
  }
  if (selectedNode.type === 'tool') {
    return <WorkflowCanvasToolNodeEditor selectedNode={selectedNode} onUpdateNode={onUpdateNode} />;
  }
  if (selectedNode.type === 'agent') {
    return <AgentEditor selectedNode={selectedNode} onUpdateNode={onUpdateNode} />;
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
      <textarea
        rows={6}
        value={selectedNode.llm?.system_prompt ?? ''}
        placeholder={copy.workflow.llmSystemPromptPlaceholder}
        onChange={(event) => onUpdateNode(withLLMSystemPrompt(selectedNode, event.target.value))}
      />
      <label className="workflow-arch-field-label">{copy.workflow.llmPrompt}</label>
      <textarea
        rows={8}
        value={selectedNode.llm?.prompt ?? ''}
        placeholder={copy.workflow.llmPromptPlaceholder}
        onChange={(event) => onUpdateNode(withLLMPrompt(selectedNode, event.target.value))}
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
      <textarea
        rows={9}
        value={selectedNode.agent?.message ?? ''}
        placeholder={copy.workflow.agentMessagePlaceholder}
        onChange={(event) => onUpdateNode(withAgentMessage(selectedNode, event.target.value))}
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
