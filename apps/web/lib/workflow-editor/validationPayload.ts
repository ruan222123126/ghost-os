import {
  type WorkflowAgentRuntimeCatalog,
  validateWorkflowAgentRuntimeNodes,
} from '@/lib/workflow-editor/agentRuntime';
import {
  INPUT_NAME_PATTERN,
  INPUT_NAME_MAX_LENGTH,
  LOOP_ROLE_END,
  LOOP_ROLE_START,
  WORKFLOW_IF_OPERATORS,
} from '@/lib/workflow-editor/constants';
import type { WorkflowCanvasNodeDraft } from '@/lib/workflow-editor/types';

const SUPPORTED_INPUT_TYPES = ['string', 'number', 'boolean', 'object', 'array'] as const;

export function validateNodePayloads(
  nodes: WorkflowCanvasNodeDraft[],
  errors: string[],
  agentRuntimeCatalog?: WorkflowAgentRuntimeCatalog,
): void {
  for (const node of nodes) {
    switch (node.type) {
      case 'start':
        validateStartNodePayload(node, errors);
        break;
      case 'tool':
        validateToolNodePayload(node, errors);
        break;
      case 'llm':
        validateLLMNodePayload(node, errors);
        break;
      case 'agent':
        validateAgentNodePayload(node, errors);
        break;
      case 'if':
        validateIfNodePayload(node, errors);
        break;
      case 'loop':
        validateLoopNodePayload(node, errors);
        break;
      case 'end':
        if (node.start || node.tool || node.llm || node.agent || node.if || node.loop) {
          errors.push(`workflow node "${node.id}" payload does not match type "end"`);
        }
        break;
      default:
        errors.push(`unsupported workflow node type "${String(node.type)}"`);
    }
  }
  validateWorkflowAgentRuntimeNodes(nodes, agentRuntimeCatalog, errors);
}

function validateStartNodePayload(node: WorkflowCanvasNodeDraft, errors: string[]): void {
  if (node.tool || node.llm || node.agent || node.if || node.loop) {
    errors.push(`workflow node "${node.id}" payload does not match type "start"`);
  }
  validateStartInputs(node, errors);
}

function validateToolNodePayload(node: WorkflowCanvasNodeDraft, errors: string[]): void {
  if (!node.tool || node.start || node.llm || node.agent || node.if || node.loop) {
    errors.push(`workflow node "${node.id}" payload does not match type "tool"`);
    return;
  }
  if (node.tool.tool_name.trim().length === 0) {
    errors.push(`workflow tool node "${node.id}" requires tool_name`);
  }
}

function validateLLMNodePayload(node: WorkflowCanvasNodeDraft, errors: string[]): void {
  if (!node.llm || node.start || node.tool || node.agent || node.if || node.loop) {
    errors.push(`workflow node "${node.id}" payload does not match type "llm"`);
    return;
  }
  if (node.llm.prompt.trim().length === 0) {
    errors.push(`workflow llm node "${node.id}" requires prompt`);
  }
}

function validateAgentNodePayload(node: WorkflowCanvasNodeDraft, errors: string[]): void {
  if (!node.agent || node.start || node.tool || node.llm || node.if || node.loop) {
    errors.push(`workflow node "${node.id}" payload does not match type "agent"`);
    return;
  }
  if (node.agent.message.trim().length === 0) {
    errors.push(`workflow agent node "${node.id}" requires message`);
  }
}

function validateIfNodePayload(node: WorkflowCanvasNodeDraft, errors: string[]): void {
  if (!node.if || node.start || node.tool || node.llm || node.agent || node.loop) {
    errors.push(`workflow node "${node.id}" payload does not match type "if"`);
    return;
  }
  if (!WORKFLOW_IF_OPERATORS.includes(node.if.operator)) {
    errors.push(`workflow if node "${node.id}" uses unsupported operator "${node.if.operator}"`);
  }
  if (!node.if.true_node_id?.trim() || !node.if.false_node_id?.trim()) {
    errors.push(`workflow if node "${node.id}" requires true_node_id and false_node_id`);
  }
  if (node.if.true_node_id?.trim() === node.if.false_node_id?.trim()) {
    errors.push(`workflow if node "${node.id}" true_node_id and false_node_id must differ`);
  }
  if (node.if.operator !== 'is_empty' && node.if.operator !== 'not_empty' && !node.if.value?.trim()) {
    errors.push(`workflow if node "${node.id}" requires value for operator "${node.if.operator}"`);
  }
}

function validateLoopNodePayload(node: WorkflowCanvasNodeDraft, errors: string[]): void {
  if (!node.loop || node.start || node.tool || node.llm || node.agent || node.if) {
    errors.push(`workflow node "${node.id}" payload does not match type "loop"`);
    return;
  }
  const loopID = node.loop.loop_id?.trim() ?? '';
  if (loopID.length === 0) {
    errors.push(`workflow loop node "${node.id}" requires loop_id`);
  }
  if (node.loop.role !== LOOP_ROLE_START && node.loop.role !== LOOP_ROLE_END) {
    errors.push(`workflow loop node "${node.id}" requires role=start|end`);
  }
  if (node.loop.role === LOOP_ROLE_START && (!node.loop.max_iterations || node.loop.max_iterations < 1)) {
    errors.push(`workflow loop node "${node.id}" start role requires max_iterations > 0`);
  }
}

function validateStartInputs(node: WorkflowCanvasNodeDraft, errors: string[]): void {
  const inputs = node.start?.inputs ?? [];
  const names = new Set<string>();
  for (let index = 0; index < inputs.length; index += 1) {
    const input = inputs[index];
    const context = `workflow start node "${node.id}" input[${index}]`;
    const name = input.name.trim();
    if (name.length === 0) {
      errors.push(`${context} name must be a non-empty string`);
      continue;
    }
    if (!INPUT_NAME_PATTERN.test(name)) {
      errors.push(`${context} name must match ^[A-Za-z_][A-Za-z0-9_]*$`);
    }
    if (name.length > INPUT_NAME_MAX_LENGTH) {
      errors.push(`${context} name length must be <= ${INPUT_NAME_MAX_LENGTH}`);
    }
    if (names.has(name)) {
      errors.push(`${context} duplicate name "${name}"`);
    }
    names.add(name);
    if (!SUPPORTED_INPUT_TYPES.includes(input.type as (typeof SUPPORTED_INPUT_TYPES)[number])) {
      errors.push(`${context} uses unsupported type "${input.type}"`);
      continue;
    }
    if (Object.prototype.hasOwnProperty.call(input, 'default') && !inputDefaultMatchesType(input.type, input.default)) {
      errors.push(`${context} value must match declared type "${input.type}"`);
    }
  }
}

function inputDefaultMatchesType(inputType: string, value: unknown): boolean {
  switch (inputType) {
    case 'string':
      return typeof value === 'string';
    case 'number':
      return typeof value === 'number' && Number.isFinite(value);
    case 'boolean':
      return typeof value === 'boolean';
    case 'object':
      return typeof value === 'object' && value !== null && !Array.isArray(value);
    case 'array':
      return Array.isArray(value);
    default:
      return false;
  }
}
