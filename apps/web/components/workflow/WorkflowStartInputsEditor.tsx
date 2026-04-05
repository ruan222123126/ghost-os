'use client';

import type { WorkflowCanvasNodeDraft } from '@/lib/workflow-editor/types';

const INPUT_TYPES = ['string', 'number', 'boolean', 'object', 'array'] as const;

interface NodeEditorProps {
  node: WorkflowCanvasNodeDraft;
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
}

type StartInputDraft = NonNullable<NonNullable<WorkflowCanvasNodeDraft['start']>['inputs']>[number];

export function WorkflowStartInputsEditor(props: NodeEditorProps) {
  const { node, onUpdateNode } = props;
  const inputs = node.start?.inputs ?? [];

  return (
    <div className="workflow-fieldset">
      <h3>start.inputs</h3>
      {inputs.map((input, index) => (
        <div className="workflow-input-item" key={`${input.name}-${index}`}>
          <label>
            <span>Name</span>
            <input value={input.name} onChange={(event) => updateInput(node, index, { name: event.target.value }, onUpdateNode)} />
          </label>
          <label>
            <span>Type</span>
            <select value={input.type} onChange={(event) => updateInput(node, index, { type: event.target.value }, onUpdateNode)}>
              {INPUT_TYPES.map((type) => (
                <option key={type} value={type}>
                  {type}
                </option>
              ))}
            </select>
          </label>
          <label>
            <span>Description</span>
            <input value={input.description ?? ''} onChange={(event) => updateInput(node, index, { description: event.target.value }, onUpdateNode)} />
          </label>
          <label className="workflow-check">
            <input type="checkbox" checked={input.required ?? false} onChange={(event) => updateInput(node, index, { required: event.target.checked }, onUpdateNode)} />
            required
          </label>
          <label className="workflow-check">
            <input type="checkbox" checked={Object.prototype.hasOwnProperty.call(input, 'default')} onChange={() => toggleDefault(node, index, onUpdateNode)} />
            has default
          </label>
          {renderDefaultField(node, input, index, onUpdateNode)}
          <button type="button" onClick={() => removeInput(node, index, onUpdateNode)}>Remove Input</button>
        </div>
      ))}
      <button type="button" onClick={() => addInput(node, onUpdateNode)}>Add Input</button>
    </div>
  );
}

export function WorkflowLLMNodeEditor(props: NodeEditorProps) {
  const { node, onUpdateNode } = props;
  return (
    <>
      <label>
        <span>Prompt</span>
        <textarea rows={4} value={node.llm?.prompt ?? ''} onChange={(event) => updateLLM(node, { prompt: event.target.value }, onUpdateNode)} />
      </label>
      <label>
        <span>System Prompt</span>
        <textarea rows={3} value={node.llm?.system_prompt ?? ''} onChange={(event) => updateLLM(node, { system_prompt: event.target.value }, onUpdateNode)} />
      </label>
    </>
  );
}

export function WorkflowAgentNodeEditor(props: NodeEditorProps) {
  const { node, onUpdateNode } = props;
  return (
    <label>
      <span>Message</span>
      <textarea rows={4} value={node.agent?.message ?? ''} onChange={(event) => onUpdateNode({ ...node, agent: { message: event.target.value } })} />
    </label>
  );
}

function renderDefaultField(
  node: WorkflowCanvasNodeDraft,
  input: StartInputDraft,
  index: number,
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void,
) {
  if (!Object.prototype.hasOwnProperty.call(input, 'default')) {
    return null;
  }
  if (input.type === 'boolean') {
    return (
      <label>
        <span>Default</span>
        <select value={String(Boolean(input.default))} onChange={(event) => updateInput(node, index, { default: event.target.value === 'true' }, onUpdateNode)}>
          <option value="true">true</option>
          <option value="false">false</option>
        </select>
      </label>
    );
  }

  return (
    <label>
      <span>Default</span>
      <textarea rows={input.type === 'object' || input.type === 'array' ? 4 : 1} value={defaultValueText(input.default)} onChange={(event) => updateDefault(node, index, event.target.value, onUpdateNode)} />
    </label>
  );
}

function updateInput(
  node: WorkflowCanvasNodeDraft,
  index: number,
  patch: Record<string, unknown>,
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void,
) {
  const inputs = [...(node.start?.inputs ?? [])];
  inputs[index] = { ...inputs[index], ...patch };
  onUpdateNode({ ...node, start: { inputs } });
}

function toggleDefault(node: WorkflowCanvasNodeDraft, index: number, onUpdateNode: (node: WorkflowCanvasNodeDraft) => void) {
  const inputs = [...(node.start?.inputs ?? [])];
  const current = inputs[index];

  if (Object.prototype.hasOwnProperty.call(current, 'default')) {
    const { default: _default, ...rest } = current;
    inputs[index] = rest;
  } else {
    inputs[index] = { ...current, default: defaultByType(current.type) };
  }
  onUpdateNode({ ...node, start: { inputs } });
}

function updateDefault(
  node: WorkflowCanvasNodeDraft,
  index: number,
  rawValue: string,
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void,
) {
  const inputs = [...(node.start?.inputs ?? [])];
  const input = inputs[index];
  inputs[index] = { ...input, default: parseDefault(rawValue, input.type) };
  onUpdateNode({ ...node, start: { inputs } });
}

function parseDefault(rawValue: string, inputType: string): unknown {
  if (inputType === 'string') {
    return rawValue;
  }
  if (inputType === 'number') {
    const parsed = Number(rawValue);
    return Number.isFinite(parsed) ? parsed : rawValue;
  }
  if (inputType === 'boolean') {
    return rawValue.trim().toLowerCase() === 'true';
  }

  try {
    return JSON.parse(rawValue);
  } catch {
    return rawValue;
  }
}

function defaultByType(inputType: string): unknown {
  if (inputType === 'string') {
    return '';
  }
  if (inputType === 'number') {
    return 0;
  }
  if (inputType === 'boolean') {
    return false;
  }
  return inputType === 'array' ? [] : {};
}

function defaultValueText(value: unknown): string {
  if (value === undefined) {
    return '';
  }
  if (typeof value === 'string') {
    return value;
  }

  return JSON.stringify(value, null, 2);
}

function removeInput(node: WorkflowCanvasNodeDraft, index: number, onUpdateNode: (node: WorkflowCanvasNodeDraft) => void) {
  const inputs = (node.start?.inputs ?? []).filter((_, inputIndex) => inputIndex !== index);
  onUpdateNode({ ...node, start: { inputs } });
}

function addInput(node: WorkflowCanvasNodeDraft, onUpdateNode: (node: WorkflowCanvasNodeDraft) => void) {
  const inputs = [...(node.start?.inputs ?? []), { name: '', type: 'string' as const, required: false }];
  onUpdateNode({ ...node, start: { inputs } });
}

function updateLLM(
  node: WorkflowCanvasNodeDraft,
  patch: Partial<NonNullable<WorkflowCanvasNodeDraft['llm']>>,
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void,
) {
  onUpdateNode({
    ...node,
    llm: {
      prompt: node.llm?.prompt ?? '',
      system_prompt: node.llm?.system_prompt ?? '',
      ...patch,
    },
  });
}
