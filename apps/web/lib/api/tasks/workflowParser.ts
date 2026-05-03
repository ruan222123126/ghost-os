import {
  expectBoolean,
  expectNumber,
  expectRecord,
  expectString,
  expectStringEnum,
  pickKnownKeys,
  parseOptionalBoolean,
  parseOptionalNumber,
  parseOptionalRecord,
  parseOptionalString,
  parseOptionalStringArray,
} from '@/lib/api/shared';

const WORKFLOW_KEYS = ['nodes', 'edges'] as const;
const WORKFLOW_NODE_KEYS = ['id', 'type', 'start', 'tool', 'llm', 'agent', 'if', 'loop'] as const;
const WORKFLOW_START_KEYS = ['inputs'] as const;
const WORKFLOW_INPUT_VARIABLE_KEYS = ['name', 'type', 'required', 'default', 'description'] as const;
const WORKFLOW_INPUT_VARIABLE_TYPES = ['string', 'number', 'boolean', 'object', 'array'] as const;
const WORKFLOW_TOOL_KEYS = ['tool_name', 'arguments'] as const;
const WORKFLOW_LLM_KEYS = ['prompt', 'system_prompt'] as const;
const WORKFLOW_AGENT_KEYS = ['message', 'runtime_overrides'] as const;
const TASK_RUNTIME_OVERRIDE_KEYS = [
  'provider_name',
  'model',
  'system_prompt',
  'tool_allowlist',
  'tool_allowlist_only',
  'max_turns',
] as const;
const WORKFLOW_IF_KEYS = ['source_node_id', 'operator', 'value', 'true_node_id', 'false_node_id'] as const;
const WORKFLOW_LOOP_KEYS = ['max_iterations', 'body_node_id', 'exit_node_id'] as const;
const WORKFLOW_IF_OPERATORS = ['equals', 'not_equals', 'contains', 'not_contains', 'is_empty', 'not_empty'] as const;
const WORKFLOW_EDGE_KEYS = ['from_node_id', 'to_node_id'] as const;
const WORKFLOW_NODE_TYPES = ['start', 'tool', 'llm', 'agent', 'if', 'loop', 'end'] as const;

interface ParsedWorkflowInputVariable {
  name: string;
  type: 'string' | 'number' | 'boolean' | 'object' | 'array';
  required?: boolean;
  default?: unknown;
  description?: string;
}

interface ParsedWorkflowStartNode {
  inputs?: ParsedWorkflowInputVariable[];
}

interface ParsedTaskRuntimeOverrides {
  provider_name?: string;
  model?: string;
  system_prompt?: string;
  tool_allowlist?: string[];
  tool_allowlist_only?: boolean;
  max_turns?: number;
}

interface ParsedWorkflowNode {
  id: string;
  type: 'start' | 'tool' | 'llm' | 'agent' | 'if' | 'loop' | 'end';
  start?: ParsedWorkflowStartNode;
  tool?: { tool_name: string; arguments?: Record<string, unknown> };
  llm?: { prompt: string; system_prompt?: string };
  agent?: {
    message: string;
    runtime_overrides?: ParsedTaskRuntimeOverrides;
  };
  if?: {
    source_node_id?: string;
    operator: (typeof WORKFLOW_IF_OPERATORS)[number];
    value?: string;
    true_node_id: string;
    false_node_id: string;
  };
  loop?: { max_iterations: number; body_node_id: string; exit_node_id: string };
}

interface ParsedWorkflowDefinition {
  nodes: ParsedWorkflowNode[];
  edges: Array<{ from_node_id: string; to_node_id: string }>;
}

export function parseWorkflowDefinition(value: unknown, label: string): ParsedWorkflowDefinition {
  const record = pickKnownKeys(expectRecord(value, label), WORKFLOW_KEYS);
  if (!Array.isArray(record.nodes)) {
    throw new Error(`Invalid ${label}.nodes: expected array`);
  }
  if (!Array.isArray(record.edges)) {
    throw new Error(`Invalid ${label}.edges: expected array`);
  }
  return {
    nodes: record.nodes.map((node, index) => parseWorkflowNode(node, `${label}.nodes[${index}]`)),
    edges: record.edges.map((edge, index) => parseWorkflowEdge(edge, `${label}.edges[${index}]`)),
  };
}

function parseWorkflowNode(value: unknown, label: string): ParsedWorkflowNode {
  const record = pickKnownKeys(expectRecord(value, label), WORKFLOW_NODE_KEYS);
  const start = parseOptionalRecord(record.start, `${label}.start`);
  const tool = parseOptionalRecord(record.tool, `${label}.tool`);
  const llm = parseOptionalRecord(record.llm, `${label}.llm`);
  const agent = parseOptionalRecord(record.agent, `${label}.agent`);
  const ifNode = parseOptionalRecord(record.if, `${label}.if`);
  const loop = parseOptionalRecord(record.loop, `${label}.loop`);
  return {
    id: expectString(record.id, `${label}.id`),
    type: expectStringEnum(record.type, WORKFLOW_NODE_TYPES, `${label}.type`),
    start: start ? parseWorkflowStartNode(start, `${label}.start`) : undefined,
    tool: tool ? parseWorkflowToolNode(tool, `${label}.tool`) : undefined,
    llm: llm ? parseWorkflowLLMNode(llm, `${label}.llm`) : undefined,
    agent: agent ? parseWorkflowAgentNode(agent, `${label}.agent`) : undefined,
    if: ifNode ? parseWorkflowIfNode(ifNode, `${label}.if`) : undefined,
    loop: loop ? parseWorkflowLoopNode(loop, `${label}.loop`) : undefined,
  };
}

function parseWorkflowStartNode(value: unknown, label: string): ParsedWorkflowStartNode {
  const record = pickKnownKeys(expectRecord(value, label), WORKFLOW_START_KEYS);
  if (record.inputs === undefined) {
    return {};
  }
  if (!Array.isArray(record.inputs)) {
    throw new Error(`Invalid ${label}.inputs: expected array`);
  }
  return {
    inputs: record.inputs.map((input, index) => parseWorkflowInputVariable(input, `${label}.inputs[${index}]`)),
  };
}

function parseWorkflowInputVariable(value: unknown, label: string): ParsedWorkflowInputVariable {
  const record = pickKnownKeys(expectRecord(value, label), WORKFLOW_INPUT_VARIABLE_KEYS);
  const inputType = expectStringEnum(record.type, WORKFLOW_INPUT_VARIABLE_TYPES, `${label}.type`);
  const parsed: ParsedWorkflowInputVariable = {
    name: expectString(record.name, `${label}.name`),
    type: inputType,
    required: parseOptionalBoolean(record.required, `${label}.required`),
    description: parseOptionalString(record.description, `${label}.description`),
  };
  if (Object.prototype.hasOwnProperty.call(record, 'default')) {
    parsed.default = parseWorkflowInputDefault(record.default, inputType, `${label}.default`);
  }
  return parsed;
}

function parseWorkflowInputDefault(value: unknown, inputType: ParsedWorkflowInputVariable['type'], label: string): unknown {
  switch (inputType) {
    case 'string':
      return expectString(value, label);
    case 'number':
      return expectNumber(value, label);
    case 'boolean':
      return expectBoolean(value, label);
    case 'object':
      return expectRecord(value, label);
    case 'array':
      if (!Array.isArray(value)) {
        throw new Error(`Invalid ${label}: expected array`);
      }
      return value;
    default:
      throw new Error(`Invalid ${label}: unsupported input type`);
  }
}

function parseWorkflowToolNode(value: Record<string, unknown>, label: string): ParsedWorkflowNode['tool'] {
  const picked = pickKnownKeys(value, WORKFLOW_TOOL_KEYS);
  return {
    tool_name: expectString(picked.tool_name, `${label}.tool_name`),
    arguments: parseOptionalRecord(picked.arguments, `${label}.arguments`),
  };
}

function parseWorkflowLLMNode(value: Record<string, unknown>, label: string): ParsedWorkflowNode['llm'] {
  const picked = pickKnownKeys(value, WORKFLOW_LLM_KEYS);
  return {
    prompt: expectString(picked.prompt, `${label}.prompt`),
    system_prompt: parseOptionalString(picked.system_prompt, `${label}.system_prompt`),
  };
}

function parseWorkflowAgentNode(value: Record<string, unknown>, label: string): ParsedWorkflowNode['agent'] {
  const picked = pickKnownKeys(value, WORKFLOW_AGENT_KEYS);
  return {
    message: expectString(picked.message, `${label}.message`),
    runtime_overrides: parseTaskRuntimeOverrides(picked.runtime_overrides, `${label}.runtime_overrides`),
  };
}

function parseTaskRuntimeOverrides(
  value: unknown,
  label: string,
): ParsedTaskRuntimeOverrides | undefined {
  const record = parseOptionalRecord(value, label);
  if (record === undefined) {
    return undefined;
  }
  const picked = pickKnownKeys(record, TASK_RUNTIME_OVERRIDE_KEYS);
  const parsed = {
    provider_name: parseOptionalString(picked.provider_name, `${label}.provider_name`),
    model: parseOptionalString(picked.model, `${label}.model`),
    system_prompt: parseOptionalString(picked.system_prompt, `${label}.system_prompt`),
    tool_allowlist: parseOptionalStringArray(picked.tool_allowlist, `${label}.tool_allowlist`),
    tool_allowlist_only: parseOptionalBoolean(picked.tool_allowlist_only, `${label}.tool_allowlist_only`),
    max_turns: parseOptionalNumber(picked.max_turns, `${label}.max_turns`),
  };
  if (
    !parsed.provider_name
    && !parsed.model
    && !parsed.system_prompt
    && !parsed.tool_allowlist?.length
    && parsed.tool_allowlist_only === undefined
    && parsed.max_turns === undefined
  ) {
    return undefined;
  }
  return parsed;
}

function parseWorkflowIfNode(value: Record<string, unknown>, label: string): ParsedWorkflowNode['if'] {
  const picked = pickKnownKeys(value, WORKFLOW_IF_KEYS);
  return {
    source_node_id: parseOptionalString(picked.source_node_id, `${label}.source_node_id`),
    operator: expectStringEnum(picked.operator, WORKFLOW_IF_OPERATORS, `${label}.operator`),
    value: parseOptionalString(picked.value, `${label}.value`),
    true_node_id: expectString(picked.true_node_id, `${label}.true_node_id`),
    false_node_id: expectString(picked.false_node_id, `${label}.false_node_id`),
  };
}

function parseWorkflowLoopNode(value: Record<string, unknown>, label: string): ParsedWorkflowNode['loop'] {
  const picked = pickKnownKeys(value, WORKFLOW_LOOP_KEYS);
  return {
    max_iterations: expectNumber(picked.max_iterations, `${label}.max_iterations`),
    body_node_id: expectString(picked.body_node_id, `${label}.body_node_id`),
    exit_node_id: expectString(picked.exit_node_id, `${label}.exit_node_id`),
  };
}

function parseWorkflowEdge(value: unknown, label: string): { from_node_id: string; to_node_id: string } {
  const record = pickKnownKeys(expectRecord(value, label), WORKFLOW_EDGE_KEYS);
  return {
    from_node_id: expectString(record.from_node_id, `${label}.from_node_id`),
    to_node_id: expectString(record.to_node_id, `${label}.to_node_id`),
  };
}
