import type { ProviderConfig, TaskRuntimeOverrides, ToolPayload } from '@/lib/types';
import type { WorkflowCanvasDraft, WorkflowCanvasNodeDraft } from '@/lib/workflow-editor/types';

const TOOL_ALLOWLIST_ONLY = true;

export interface WorkflowAgentRuntimeCatalog {
  providers: ProviderConfig[];
  activeProvider: string;
  tools: ToolPayload[];
}

export function enabledWorkflowAgentToolNames(tools: ToolPayload[]): string[] {
  const names = tools
    .filter((tool) => tool.enabled)
    .map((tool) => tool.name.trim())
    .filter((name) => name.length > 0);
  return [...new Set(names)].sort((left, right) => left.localeCompare(right));
}

export function defaultWorkflowAgentRuntimeOverrides(enabledToolNames: string[]): TaskRuntimeOverrides {
  return {
    tool_allowlist_only: TOOL_ALLOWLIST_ONLY,
    tool_allowlist: [...enabledToolNames],
  };
}

export function normalizeWorkflowDraftAgentNodes(
  draft: WorkflowCanvasDraft,
  enabledToolNames: string[],
): WorkflowCanvasDraft {
  return {
    ...draft,
    nodes: draft.nodes.map((node) => normalizeWorkflowAgentNode(node, enabledToolNames)),
  };
}

export function normalizeWorkflowAgentNode(
  node: WorkflowCanvasNodeDraft,
  enabledToolNames: string[],
): WorkflowCanvasNodeDraft {
  if (node.type !== 'agent' || !node.agent || node.agent.runtime_overrides) {
    return node;
  }
  return {
    ...node,
    agent: {
      ...node.agent,
      runtime_overrides: defaultWorkflowAgentRuntimeOverrides(enabledToolNames),
    },
  };
}

export function cloneWorkflowTaskRuntimeOverrides(
  overrides?: TaskRuntimeOverrides,
): TaskRuntimeOverrides | undefined {
  if (!overrides) {
    return undefined;
  }
  return {
    provider_name: overrides.provider_name,
    model: overrides.model,
    system_prompt: overrides.system_prompt,
    tool_allowlist: overrides.tool_allowlist ? [...overrides.tool_allowlist] : undefined,
    tool_allowlist_only: overrides.tool_allowlist_only,
    max_turns: overrides.max_turns,
  };
}

export function validateWorkflowAgentRuntimeNodes(
  nodes: WorkflowCanvasNodeDraft[],
  catalog: WorkflowAgentRuntimeCatalog | undefined,
  errors: string[],
): void {
  if (!catalog) {
    return;
  }
  const enabledToolNames = new Set(enabledWorkflowAgentToolNames(catalog.tools));
  for (const node of nodes) {
    validateWorkflowAgentRuntimeNode(node, catalog.providers, enabledToolNames, errors);
  }
}

function validateWorkflowAgentRuntimeNode(
  node: WorkflowCanvasNodeDraft,
  providers: ProviderConfig[],
  enabledToolNames: Set<string>,
  errors: string[],
): void {
  if (node.type !== 'agent' || !node.agent?.runtime_overrides) {
    return;
  }
  const overrides = node.agent.runtime_overrides;
  const providerName = overrides.provider_name?.trim() ?? '';
  const model = overrides.model?.trim() ?? '';
  if ((providerName.length > 0 || model.length > 0) && (providerName.length === 0 || model.length === 0)) {
    errors.push(`workflow agent node "${node.id}" provider_name and model must be set together`);
  }
  if (providerName.length > 0) {
    validateWorkflowAgentProviderModel(node.id, providerName, model, providers, errors);
  }
  if (overrides.max_turns !== undefined && (!Number.isInteger(overrides.max_turns) || overrides.max_turns < 1)) {
    errors.push(`workflow agent node "${node.id}" max_turns must be > 0`);
  }
  validateWorkflowAgentTools(node.id, overrides, enabledToolNames, errors);
}

function validateWorkflowAgentProviderModel(
  nodeID: string,
  providerName: string,
  model: string,
  providers: ProviderConfig[],
  errors: string[],
): void {
  const provider = providers.find((item) => item.name.trim() === providerName);
  if (!provider) {
    errors.push(`workflow agent node "${nodeID}" provider_name "${providerName}" is not available`);
    return;
  }
  const models = new Set((provider.models ?? []).map((item) => item.trim()).filter((item) => item.length > 0));
  if (!models.has(model)) {
    errors.push(`workflow agent node "${nodeID}" model "${model}" is not available for provider "${providerName}"`);
  }
}

function validateWorkflowAgentTools(
  nodeID: string,
  overrides: TaskRuntimeOverrides,
  enabledToolNames: Set<string>,
  errors: string[],
): void {
  for (const toolName of overrides.tool_allowlist ?? []) {
    const normalized = toolName.trim();
    if (normalized.length === 0 || enabledToolNames.has(normalized)) {
      continue;
    }
    errors.push(`workflow agent node "${nodeID}" tool_allowlist contains disabled or unavailable tool "${normalized}"`);
  }
}
