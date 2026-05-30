import type { PresetPayload, ProviderConfig } from '@/lib/types';
import { availableWorkflowAgentToolNames, type WorkflowAgentRuntimeCatalog } from '@/lib/workflow-editor/agentRuntime';
import type { WorkflowCanvasDraft, WorkflowValidationResult } from '@/lib/workflow-editor/types';

const SPEAKING_MODES = ['sequential', 'parallel', 'owner'] as const;
const GROUP_REQUIRED_ERROR = 'orchestration requires at least 1 group node';
const LEGACY_BOUNDARY_ERROR = 'orchestration node type "%s" is removed; run `bin/ghost-bridge migrate orchestrations`';

export function validateOrchestrationDraft(
  draft: WorkflowCanvasDraft,
  options?: { agentRuntimeCatalog?: WorkflowAgentRuntimeCatalog; presets?: PresetPayload[] },
): WorkflowValidationResult {
  const errors: string[] = [];
  const nodeMap = new Map<string, WorkflowCanvasDraft['nodes'][number]>();
  const controlIn = new Map<string, number>();
  const controlOut = new Map<string, number>();
  const controlNext = new Map<string, string>();
  const groupMembers = new Map<string, string[]>();
  const rawControlIn = new Map<string, number>();
  const rawControlOut = new Map<string, number>();
  for (const node of draft.nodes) {
    if (!node.id.trim()) {
      errors.push('orchestration node id is required');
    } else if (nodeMap.has(node.id)) {
      errors.push(`duplicate orchestration node id "${node.id}"`);
    }
    nodeMap.set(node.id, node);
    validateOrchestrationNode(node, errors);
  }
  for (const edge of draft.edges) {
    validateOrchestrationEdge(edge, nodeMap, rawControlIn, rawControlOut, controlIn, controlOut, controlNext, groupMembers, errors);
  }
  validateOrchestrationDegrees(draft, rawControlIn, rawControlOut, controlIn, controlOut, errors);
  validateOrchestrationConnectivity(draft, controlIn, controlNext, errors);
  validateOrchestrationMembers(draft, groupMembers, errors);
  validateOrchestrationAgentRuntime(draft, options?.agentRuntimeCatalog, options?.presets ?? [], errors);
  return { valid: errors.length === 0, errors };
}

function validateOrchestrationNode(node: WorkflowCanvasDraft['nodes'][number], errors: string[]) {
  if (node.type === 'group') {
    if (!node.group?.title?.trim()) {
      errors.push(`orchestration group node "${node.id}" requires title`);
    }
    if (!SPEAKING_MODES.includes((node.group?.speaking_mode ?? '') as (typeof SPEAKING_MODES)[number])) {
      errors.push(`orchestration group node "${node.id}" requires speaking_mode sequential|parallel|owner`);
    }
    if (!Number.isInteger(node.group?.max_rounds) || (node.group?.max_rounds ?? 0) < 1) {
      errors.push(`orchestration group node "${node.id}" requires max_rounds > 0`);
    }
    if (node.group?.speaking_mode === 'owner' && !node.group?.owner_agent_id?.trim()) {
      errors.push(`orchestration group node "${node.id}" requires owner_agent_id in owner mode`);
    }
  }
  if (node.type === 'agent') {
    if (!node.agent?.title?.trim() || !node.agent.message.trim()) {
      errors.push(`orchestration agent node "${node.id}" requires title and message`);
    }
  }
  if (node.type === 'start' || node.type === 'end') {
    errors.push(LEGACY_BOUNDARY_ERROR.replace('%s', node.type));
  }
  if (node.type !== 'start' && node.type !== 'end' && node.type !== 'group' && node.type !== 'agent') {
    errors.push(`unsupported orchestration node type "${String(node.type)}"`);
  }
}

function validateOrchestrationEdge(
  edge: WorkflowCanvasDraft['edges'][number],
  nodeMap: Map<string, WorkflowCanvasDraft['nodes'][number]>,
  rawControlIn: Map<string, number>,
  rawControlOut: Map<string, number>,
  controlIn: Map<string, number>,
  controlOut: Map<string, number>,
  controlNext: Map<string, string>,
  groupMembers: Map<string, string[]>,
  errors: string[],
) {
  const source = nodeMap.get(edge.from_node_id);
  const target = nodeMap.get(edge.to_node_id);
  if (!source || !target) {
    errors.push('orchestration edge references unknown node id');
    return;
  }
  if (edge.kind === 'member') {
    if (source.type !== 'agent' || target.type !== 'group') {
      errors.push('orchestration member edge must be agent -> group');
      return;
    }
    groupMembers.set(target.id, [...(groupMembers.get(target.id) ?? []), source.id]);
    return;
  }
  if (edge.kind !== 'control') {
    errors.push(`unsupported orchestration edge kind "${String(edge.kind)}"`);
    return;
  }
  if (!(source.type === 'group' && target.type === 'group')) {
    errors.push(`orchestration control edge "${edge.from_node_id}" -> "${edge.to_node_id}" is invalid`);
    return;
  }
  rawControlOut.set(source.id, (rawControlOut.get(source.id) ?? 0) + 1);
  rawControlIn.set(target.id, (rawControlIn.get(target.id) ?? 0) + 1);
  if (source.type === 'group' && target.type === 'group') {
    controlNext.set(source.id, target.id);
    controlOut.set(source.id, (controlOut.get(source.id) ?? 0) + 1);
    controlIn.set(target.id, (controlIn.get(target.id) ?? 0) + 1);
  }
}

function validateOrchestrationDegrees(
  draft: WorkflowCanvasDraft,
  rawControlIn: Map<string, number>,
  rawControlOut: Map<string, number>,
  controlIn: Map<string, number>,
  controlOut: Map<string, number>,
  errors: string[],
) {
  const groups = draft.nodes.filter((node) => node.type === 'group');
  const hasVisibleConfig = groups.length > 0 || draft.nodes.some((node) => node.type === 'agent') || draft.edges.length > 0;
  if (groups.length === 0) {
    if (hasVisibleConfig) {
      errors.push(GROUP_REQUIRED_ERROR);
    }
    return;
  }

  let entryCount = 0;
  let exitCount = 0;
  for (const node of draft.nodes) {
    const inDegree = controlIn.get(node.id) ?? 0;
    const outDegree = controlOut.get(node.id) ?? 0;
    const rawInDegree = rawControlIn.get(node.id) ?? 0;
    const rawOutDegree = rawControlOut.get(node.id) ?? 0;
    if (node.type === 'group' && (inDegree > 1 || outDegree > 1)) {
      errors.push(`orchestration group node "${node.id}" must have in<=1 and out<=1`);
    }
    if (node.type === 'group' && inDegree === 0) {
      entryCount += 1;
    }
    if (node.type === 'group' && outDegree === 0) {
      exitCount += 1;
    }
    if (node.type === 'agent' && (rawInDegree !== 0 || rawOutDegree !== 0)) {
      errors.push(`orchestration agent node "${node.id}" cannot participate in control flow`);
    }
  }
  if (entryCount !== 1) {
    errors.push('orchestration requires exactly 1 entry group');
  }
  if (exitCount !== 1) {
    errors.push('orchestration requires exactly 1 exit group');
  }
}

function validateOrchestrationConnectivity(
  draft: WorkflowCanvasDraft,
  controlIn: Map<string, number>,
  controlNext: Map<string, string>,
  errors: string[],
) {
  const groups = draft.nodes.filter((node) => node.type === 'group');
  if (groups.length === 0) {
    return;
  }
  const startNode = groups.find((node) => (controlIn.get(node.id) ?? 0) === 0);
  if (!startNode) {
    return;
  }
  const visited = new Set<string>();
  let currentID = startNode?.id ?? '';
  let hasCycle = false;
  while (currentID) {
    if (visited.has(currentID)) {
      hasCycle = true;
      break;
    }
    visited.add(currentID);
    currentID = controlNext.get(currentID) ?? '';
  }
  if (hasCycle) {
    errors.push('orchestration control flow contains a cycle');
  }
  for (const node of groups) {
    if (!visited.has(node.id)) {
      errors.push(`orchestration node "${node.id}" is disconnected from control flow`);
    }
  }
}

function validateOrchestrationMembers(
  draft: WorkflowCanvasDraft,
  groupMembers: Map<string, string[]>,
  errors: string[],
) {
  for (const node of draft.nodes) {
    if (node.type === 'group' && (groupMembers.get(node.id)?.length ?? 0) === 0) {
      errors.push(`orchestration group node "${node.id}" requires at least one member`);
      continue;
    }
    if (node.type !== 'group' || node.group?.speaking_mode !== 'owner') {
      continue;
    }
    const ownerID = node.group.owner_agent_id?.trim() ?? '';
    if (!ownerID) {
      continue;
    }
    const members = groupMembers.get(node.id) ?? [];
    if (!members.includes(ownerID)) {
      errors.push(`orchestration group node "${node.id}" owner_agent_id "${ownerID}" must be an existing member`);
    }
  }
}

function validateOrchestrationAgentRuntime(
  draft: WorkflowCanvasDraft,
  catalog: WorkflowAgentRuntimeCatalog | undefined,
  presets: PresetPayload[] | undefined,
  errors: string[],
) {
  if (!catalog) {
    return;
  }
  const tools = new Set(availableWorkflowAgentToolNames(catalog.tools));
  const presetIDs = presets
    ? new Set(presets.map((preset) => preset.id.trim()).filter((presetID) => presetID.length > 0))
    : undefined;
  for (const node of draft.nodes) {
    if (node.type !== 'agent' || !node.agent?.runtime_overrides) {
      continue;
    }
    validateProviderAndModel(node.id, node.agent.runtime_overrides.provider_name ?? '', node.agent.runtime_overrides.model ?? '', catalog.providers, errors);
    const presetID = node.agent.runtime_overrides.preset_id?.trim() ?? '';
    if (presetID.length > 0 && presetIDs && !presetIDs.has(presetID)) {
      errors.push(`orchestration agent node "${node.id}" preset_id "${presetID}" is not available`);
    }
    for (const toolName of node.agent.runtime_overrides.tool_allowlist ?? []) {
      if (!tools.has(toolName.trim())) {
        errors.push(`orchestration agent node "${node.id}" tool_allowlist contains unavailable tool "${toolName.trim()}"`);
      }
    }
  }
}

function validateProviderAndModel(nodeID: string, providerName: string, model: string, providers: ProviderConfig[], errors: string[]) {
  if ((providerName || model) && (!providerName || !model)) {
    errors.push(`orchestration agent node "${nodeID}" provider_name and model must be set together`);
    return;
  }
  if (!providerName) {
    return;
  }
  const provider = providers.find((item) => item.name.trim() === providerName.trim());
  if (!provider) {
    errors.push(`orchestration agent node "${nodeID}" provider_name "${providerName}" is not available`);
    return;
  }
  if (!(provider.models ?? []).includes(model)) {
    errors.push(`orchestration agent node "${nodeID}" model "${model}" is not available for provider "${providerName}"`);
  }
}
