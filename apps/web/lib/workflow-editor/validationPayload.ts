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
const NODE_PAYLOAD_FIELDS = ['start', 'tool', 'llm', 'agent', 'if', 'loop'] as const;

type WorkflowPayloadField = (typeof NODE_PAYLOAD_FIELDS)[number];
type SupportedInputType = (typeof SUPPORTED_INPUT_TYPES)[number];
type WorkflowNodePayloadValidator = (node: WorkflowCanvasNodeDraft, errors: string[]) => void;
type WorkflowStartInput = NonNullable<NonNullable<WorkflowCanvasNodeDraft['start']>['inputs']>[number];
type WorkflowIfOperator = NonNullable<WorkflowCanvasNodeDraft['if']>['operator'];
type WorkflowNodeWithPayload<K extends WorkflowPayloadField> = WorkflowCanvasNodeDraft & {
  [P in K]-?: NonNullable<WorkflowCanvasNodeDraft[P]>;
};
interface StartInputValidationContext {
  label: string;
  names: Set<string>;
  errors: string[];
}

const WORKFLOW_NODE_PAYLOAD_VALIDATORS: Partial<
  Record<WorkflowCanvasNodeDraft['type'], WorkflowNodePayloadValidator>
> = {
  start: validateStartNodePayload,
  tool: validateToolNodePayload,
  llm: validateLLMNodePayload,
  agent: validateAgentNodePayload,
  if: validateIfNodePayload,
  loop: validateLoopNodePayload,
  end: validateEndNodePayload,
};

const INPUT_DEFAULT_MATCHERS: Record<SupportedInputType, (value: unknown) => boolean> = {
  string: (value) => typeof value === 'string',
  number: (value) => typeof value === 'number' && Number.isFinite(value),
  boolean: (value) => typeof value === 'boolean',
  object: (value) => typeof value === 'object' && value !== null && !Array.isArray(value),
  array: Array.isArray,
};

export function validateNodePayloads(
  nodes: WorkflowCanvasNodeDraft[],
  errors: string[],
  agentRuntimeCatalog?: WorkflowAgentRuntimeCatalog,
): void {
  for (const node of nodes) {
    const validator = WORKFLOW_NODE_PAYLOAD_VALIDATORS[node.type];
    if (!validator) {
      errors.push(`unsupported workflow node type "${String(node.type)}"`);
      continue;
    }
    validator(node, errors);
  }
  validateWorkflowAgentRuntimeNodes(nodes, agentRuntimeCatalog, errors);
}

function validateStartNodePayload(node: WorkflowCanvasNodeDraft, errors: string[]): void {
  if (hasUnexpectedPayload(node, 'start')) {
    errors.push(`workflow node "${node.id}" payload does not match type "start"`);
  }
  validateStartInputs(node, errors);
}

function validateToolNodePayload(node: WorkflowCanvasNodeDraft, errors: string[]): void {
  if (!hasExpectedPayload(node, 'tool')) {
    errors.push(`workflow node "${node.id}" payload does not match type "tool"`);
    return;
  }
  if (node.tool.tool_name.trim().length === 0) {
    errors.push(`workflow tool node "${node.id}" requires tool_name`);
  }
}

function validateLLMNodePayload(node: WorkflowCanvasNodeDraft, errors: string[]): void {
  if (!hasExpectedPayload(node, 'llm')) {
    errors.push(`workflow node "${node.id}" payload does not match type "llm"`);
    return;
  }
  if (node.llm.prompt.trim().length === 0) {
    errors.push(`workflow llm node "${node.id}" requires prompt`);
  }
}

function validateAgentNodePayload(node: WorkflowCanvasNodeDraft, errors: string[]): void {
  if (!hasExpectedPayload(node, 'agent')) {
    errors.push(`workflow node "${node.id}" payload does not match type "agent"`);
    return;
  }
  if (node.agent.message.trim().length === 0) {
    errors.push(`workflow agent node "${node.id}" requires message`);
  }
}

function validateIfNodePayload(node: WorkflowCanvasNodeDraft, errors: string[]): void {
  if (!hasExpectedPayload(node, 'if')) {
    errors.push(`workflow node "${node.id}" payload does not match type "if"`);
    return;
  }
  validateIfOperator(node, errors);
  validateIfTargets(node, errors);
  validateIfValue(node, errors);
}

function validateLoopNodePayload(node: WorkflowCanvasNodeDraft, errors: string[]): void {
  if (!hasExpectedPayload(node, 'loop')) {
    errors.push(`workflow node "${node.id}" payload does not match type "loop"`);
    return;
  }
  validateLoopID(node, errors);
  validateLoopRole(node, errors);
  validateLoopIterations(node, errors);
}

function validateEndNodePayload(node: WorkflowCanvasNodeDraft, errors: string[]): void {
  if (hasUnexpectedPayload(node)) {
    errors.push(`workflow node "${node.id}" payload does not match type "end"`);
  }
}

function hasExpectedPayload<K extends WorkflowPayloadField>(
  node: WorkflowCanvasNodeDraft,
  expected: K,
): node is WorkflowNodeWithPayload<K> {
  return Boolean(node[expected]) && !hasUnexpectedPayload(node, expected);
}

function hasUnexpectedPayload(
  node: WorkflowCanvasNodeDraft,
  expected?: WorkflowPayloadField,
): boolean {
  return NODE_PAYLOAD_FIELDS.some((field) => field !== expected && Boolean(node[field]));
}

function validateIfOperator(node: WorkflowCanvasNodeDraft, errors: string[]): void {
  const ifConfig = node.if;
  if (ifConfig && !WORKFLOW_IF_OPERATORS.includes(ifConfig.operator)) {
    errors.push(`workflow if node "${node.id}" uses unsupported operator "${ifConfig.operator}"`);
  }
}

function validateIfTargets(node: WorkflowCanvasNodeDraft, errors: string[]): void {
  const trueNodeID = optionalTrimmedText(node.if?.true_node_id);
  const falseNodeID = optionalTrimmedText(node.if?.false_node_id);
  if (!trueNodeID || !falseNodeID) {
    errors.push(`workflow if node "${node.id}" requires true_node_id and false_node_id`);
  }
  if (trueNodeID === falseNodeID) {
    errors.push(`workflow if node "${node.id}" true_node_id and false_node_id must differ`);
  }
}

function optionalTrimmedText(value: string | undefined): string {
  return value?.trim() ?? '';
}

function validateIfValue(node: WorkflowCanvasNodeDraft, errors: string[]): void {
  const ifConfig = node.if;
  if (!ifConfig || ifOperatorAllowsEmptyValue(ifConfig.operator)) {
    return;
  }
  if (!ifConfig.value?.trim()) {
    errors.push(`workflow if node "${node.id}" requires value for operator "${ifConfig.operator}"`);
  }
}

function ifOperatorAllowsEmptyValue(operator: WorkflowIfOperator): boolean {
  return operator === 'is_empty' || operator === 'not_empty';
}

function validateLoopID(node: WorkflowCanvasNodeDraft, errors: string[]): void {
  if (loopID(node).length === 0) {
    errors.push(`workflow loop node "${node.id}" requires loop_id`);
  }
}

function validateLoopRole(node: WorkflowCanvasNodeDraft, errors: string[]): void {
  if (node.loop?.role !== LOOP_ROLE_START && node.loop?.role !== LOOP_ROLE_END) {
    errors.push(`workflow loop node "${node.id}" requires role=start|end`);
  }
}

function validateLoopIterations(node: WorkflowCanvasNodeDraft, errors: string[]): void {
  if (node.loop?.role === LOOP_ROLE_START && !hasPositiveMaxIterations(node)) {
    errors.push(`workflow loop node "${node.id}" start role requires max_iterations > 0`);
  }
}

function loopID(node: WorkflowCanvasNodeDraft): string {
  return node.loop?.loop_id?.trim() ?? '';
}

function hasPositiveMaxIterations(node: WorkflowCanvasNodeDraft): boolean {
  return Boolean(node.loop?.max_iterations && node.loop.max_iterations >= 1);
}

function validateStartInputs(node: WorkflowCanvasNodeDraft, errors: string[]): void {
  const inputs = node.start?.inputs ?? [];
  const names = new Set<string>();
  for (let index = 0; index < inputs.length; index += 1) {
    const input = inputs[index];
    validateStartInput(input, {
      label: `workflow start node "${node.id}" input[${index}]`,
      names,
      errors,
    });
  }
}

function validateStartInput(
  input: WorkflowStartInput,
  context: StartInputValidationContext,
): void {
  const name = input.name.trim();
  if (!validateStartInputName(name, context)) {
    return;
  }
  validateStartInputType(input, context);
}

function validateStartInputName(
  name: string,
  context: StartInputValidationContext,
): boolean {
  if (name.length === 0) {
    context.errors.push(`${context.label} name must be a non-empty string`);
    return false;
  }
  if (!INPUT_NAME_PATTERN.test(name)) {
    context.errors.push(`${context.label} name must match ^[A-Za-z_][A-Za-z0-9_]*$`);
  }
  if (name.length > INPUT_NAME_MAX_LENGTH) {
    context.errors.push(`${context.label} name length must be <= ${INPUT_NAME_MAX_LENGTH}`);
  }
  if (context.names.has(name)) {
    context.errors.push(`${context.label} duplicate name "${name}"`);
  }
  context.names.add(name);
  return true;
}

function validateStartInputType(
  input: WorkflowStartInput,
  context: StartInputValidationContext,
): void {
  if (!isSupportedInputType(input.type)) {
    context.errors.push(`${context.label} uses unsupported type "${input.type}"`);
    return;
  }
  if (hasDefaultValue(input) && !inputDefaultMatchesType(input.type, input.default)) {
    context.errors.push(`${context.label} value must match declared type "${input.type}"`);
  }
}

function isSupportedInputType(
  inputType: string,
): inputType is SupportedInputType {
  return SUPPORTED_INPUT_TYPES.includes(inputType as SupportedInputType);
}

function hasDefaultValue(
  input: WorkflowStartInput,
): boolean {
  return Object.prototype.hasOwnProperty.call(input, 'default');
}

function inputDefaultMatchesType(inputType: string, value: unknown): boolean {
  return isSupportedInputType(inputType) && INPUT_DEFAULT_MATCHERS[inputType](value);
}
