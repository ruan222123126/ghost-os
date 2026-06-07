import {
  expectNumber,
  expectRecord,
  expectString,
  expectStringEnum,
  pickKnownKeys,
  parseOptionalRecord,
  parseOptionalString,
  parseOptionalStringArray,
  parseOptionalBoolean,
  parseOptionalNumber,
} from '@/lib/api/shared';
import type {
  OrchestrationAgentNode,
  OrchestrationDefinition,
  OrchestrationEdge,
  OrchestrationGroupNode,
  OrchestrationNode,
  TaskRuntimeOverrides,
} from '@/lib/types';

const NODE_KEYS = ['id', 'type', 'group', 'agent'] as const;
const GROUP_KEYS = ['title', 'shared_context', 'speaking_mode', 'owner_agent_id', 'max_rounds'] as const;
const AGENT_KEYS = ['title', 'message', 'runtime_overrides'] as const;
const EDGE_KEYS = ['from_node_id', 'to_node_id', 'kind'] as const;
const DEFINITION_KEYS = ['nodes', 'edges'] as const;
const NODE_TYPES = ['group', 'agent'] as const;
const EDGE_KINDS = ['control', 'member'] as const;
const SPEAKING_MODES = ['sequential', 'parallel', 'owner'] as const;
const TASK_RUNTIME_OVERRIDE_KEYS = [
  'provider_name',
  'model',
  'system_prompt',
  'preset_id',
  'tool_allowlist',
  'tool_allowlist_only',
  'max_turns',
] as const;

export function parseOrchestrationDefinition(value: unknown, label: string): OrchestrationDefinition {
  const record = pickKnownKeys(expectRecord(value, label), DEFINITION_KEYS);
  if (!Array.isArray(record.nodes) || !Array.isArray(record.edges)) {
    throw new Error(`Invalid ${label}: expected nodes and edges arrays`);
  }
  return {
    nodes: record.nodes.map((node, index) => parseOrchestrationNode(node, `${label}.nodes[${index}]`)),
    edges: record.edges.map((edge, index) => parseOrchestrationEdge(edge, `${label}.edges[${index}]`)),
  };
}

function parseOrchestrationNode(value: unknown, label: string): OrchestrationNode {
  const record = pickKnownKeys(expectRecord(value, label), NODE_KEYS);
  const type = expectStringEnum(record.type, NODE_TYPES, `${label}.type`);
  const group = parseOptionalRecord(record.group, `${label}.group`);
  const agent = parseOptionalRecord(record.agent, `${label}.agent`);
  return {
    id: expectString(record.id, `${label}.id`),
    type,
    group: group ? parseGroupNode(group, `${label}.group`) : undefined,
    agent: agent ? parseAgentNode(agent, `${label}.agent`) : undefined,
  };
}

function parseGroupNode(value: Record<string, unknown>, label: string): OrchestrationGroupNode {
  const picked = pickKnownKeys(value, GROUP_KEYS);
  return {
    title: expectString(picked.title, `${label}.title`),
    shared_context: parseOptionalString(picked.shared_context, `${label}.shared_context`) ?? '',
    speaking_mode: expectStringEnum(picked.speaking_mode, SPEAKING_MODES, `${label}.speaking_mode`),
    owner_agent_id: parseOptionalString(picked.owner_agent_id, `${label}.owner_agent_id`) ?? undefined,
    max_rounds: expectNumber(picked.max_rounds, `${label}.max_rounds`),
  };
}

function parseAgentNode(value: Record<string, unknown>, label: string): OrchestrationAgentNode {
  const picked = pickKnownKeys(value, AGENT_KEYS);
  return {
    title: expectString(picked.title, `${label}.title`),
    message: expectString(picked.message, `${label}.message`),
    runtime_overrides: parseTaskRuntimeOverrides(picked.runtime_overrides, `${label}.runtime_overrides`),
  };
}

function parseOrchestrationEdge(value: unknown, label: string): OrchestrationEdge {
  const record = pickKnownKeys(expectRecord(value, label), EDGE_KEYS);
  return {
    from_node_id: expectString(record.from_node_id, `${label}.from_node_id`),
    to_node_id: expectString(record.to_node_id, `${label}.to_node_id`),
    kind: expectStringEnum(record.kind, EDGE_KINDS, `${label}.kind`),
  };
}

function parseTaskRuntimeOverrides(value: unknown, label: string): TaskRuntimeOverrides | undefined {
  const record = parseOptionalRecord(value, label);
  if (record === undefined) {
    return undefined;
  }
  const picked = pickKnownKeys(record, TASK_RUNTIME_OVERRIDE_KEYS);
  const parsed = {
    provider_name: parseOptionalString(picked.provider_name, `${label}.provider_name`),
    model: parseOptionalString(picked.model, `${label}.model`),
    system_prompt: parseOptionalString(picked.system_prompt, `${label}.system_prompt`),
    preset_id: parseOptionalString(picked.preset_id, `${label}.preset_id`),
    tool_allowlist: parseOptionalStringArray(picked.tool_allowlist, `${label}.tool_allowlist`),
    tool_allowlist_only: parseOptionalBoolean(picked.tool_allowlist_only, `${label}.tool_allowlist_only`),
    max_turns: parseOptionalNumber(picked.max_turns, `${label}.max_turns`),
  };
  if (
    !parsed.provider_name
    && !parsed.model
    && !parsed.system_prompt
    && !parsed.preset_id
    && !parsed.tool_allowlist?.length
    && parsed.tool_allowlist_only === undefined
    && parsed.max_turns === undefined
  ) {
    return undefined;
  }
  return parsed;
}
