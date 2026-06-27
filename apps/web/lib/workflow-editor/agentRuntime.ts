import type { ProviderConfig, TaskRuntimeOverrides, ToolPayload } from '@/lib/types';
import type { WorkflowCanvasDraft, WorkflowCanvasNodeDraft } from '@/lib/workflow-editor/types';

const TOOL_ALLOWLIST_ONLY = true;
const TASK_RUNTIME_OVERRIDE_FIELDS = [
  'provider_name',
  'model',
  'system_prompt',
  'preset_id',
  'tool_allowlist',
  'tool_allowlist_only',
  'max_turns',
] as const satisfies readonly (keyof TaskRuntimeOverrides)[];

export interface WorkflowAgentRuntimeCatalog {
  providers: ProviderConfig[];
  activeProvider: string;
  tools: ToolPayload[];
}

interface WorkflowAgentRuntimeValidationContext {
  providers: ProviderConfig[];
  enabledToolNames: Set<string>;
  errors: string[];
}

interface WorkflowAgentRuntimeNodeValidation {
  nodeID: string;
  overrides: TaskRuntimeOverrides;
  context: WorkflowAgentRuntimeValidationContext;
}

interface WorkflowAgentProviderModelValidation {
  nodeID: string;
  providerName: string;
  model: string;
  context: WorkflowAgentRuntimeValidationContext;
}

interface ProviderModelSelection {
  providerName: string;
  model: string;
}

export function enabledWorkflowAgentToolNames(tools: ToolPayload[]): string[] {
  return normalizeWorkflowAgentToolNames(tools.filter((tool) => tool.enabled).map((tool) => tool.name.trim()));
}

export function availableWorkflowAgentToolNames(tools: ToolPayload[]): string[] {
  return normalizeWorkflowAgentToolNames(tools.map((tool) => tool.name.trim()));
}

function normalizeWorkflowAgentToolNames(names: string[]): string[] {
  const filtered = names.filter((name) => name.length > 0);
  return [...new Set(filtered)].sort((left, right) => left.localeCompare(right));
}

export function defaultWorkflowAgentRuntimeOverrides(enabledToolNames: string[]): TaskRuntimeOverrides {
  return {
    tool_allowlist_only: TOOL_ALLOWLIST_ONLY,
    tool_allowlist: [...enabledToolNames],
  };
}

export function defaultOrchestrationAgentRuntimeOverrides(toolNames: string[]): TaskRuntimeOverrides {
  return defaultWorkflowAgentRuntimeOverrides(toolNames);
}

export function normalizeWorkflowDraftAgentNodes(
  draft: WorkflowCanvasDraft,
  enabledToolNames: string[],
): WorkflowCanvasDraft {
  let changed = false;
  const nodes = draft.nodes.map((node) => {
    const normalized = normalizeWorkflowAgentNode(node, enabledToolNames);
    if (normalized !== node) {
      changed = true;
    }
    return normalized;
  });
  if (!changed) {
    return draft;
  }
  return {
    ...draft,
    nodes,
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
  return cloneTaskRuntimeOverrides(overrides);
}

export function cloneOrchestrationTaskRuntimeOverrides(
  overrides?: TaskRuntimeOverrides,
): TaskRuntimeOverrides | undefined {
  const cloned = cloneTaskRuntimeOverrides(overrides);
  if (!cloned) {
    return undefined;
  }
  return normalizeEmptyTaskRuntimeOverrides({
    ...cloned,
    max_turns: undefined,
  });
}

function cloneTaskRuntimeOverrides(
  overrides?: TaskRuntimeOverrides,
): TaskRuntimeOverrides | undefined {
  if (!overrides) {
    return undefined;
  }
  return normalizeEmptyTaskRuntimeOverrides({
    provider_name: overrides.provider_name,
    model: overrides.model,
    system_prompt: overrides.system_prompt,
    preset_id: overrides.preset_id,
    tool_allowlist: overrides.tool_allowlist ? [...overrides.tool_allowlist] : undefined,
    tool_allowlist_only: overrides.tool_allowlist_only,
    max_turns: overrides.max_turns,
  });
}

function normalizeEmptyTaskRuntimeOverrides(
  overrides: TaskRuntimeOverrides,
): TaskRuntimeOverrides | undefined {
  if (TASK_RUNTIME_OVERRIDE_FIELDS.every((field) => overrides[field] === undefined)) {
    return undefined;
  }
  return overrides;
}

export function validateWorkflowAgentRuntimeNodes(
  nodes: WorkflowCanvasNodeDraft[],
  catalog: WorkflowAgentRuntimeCatalog | undefined,
  errors: string[],
): void {
  if (!catalog) {
    return;
  }
  const context = {
    providers: catalog.providers,
    enabledToolNames: new Set(enabledWorkflowAgentToolNames(catalog.tools)),
    errors,
  };
  for (const node of nodes) {
    validateWorkflowAgentRuntimeNode(node, context);
  }
}

function validateWorkflowAgentRuntimeNode(
  node: WorkflowCanvasNodeDraft,
  context: WorkflowAgentRuntimeValidationContext,
): void {
  if (node.type !== 'agent' || !node.agent?.runtime_overrides) {
    return;
  }
  const overrides = node.agent.runtime_overrides;
  validateWorkflowAgentRuntime({
    nodeID: node.id,
    overrides,
    context,
  });
}

function validateWorkflowAgentRuntime(input: WorkflowAgentRuntimeNodeValidation): void {
  validateWorkflowAgentProviderModelPair(input);
  validateWorkflowAgentMaxTurns(input);
  validateWorkflowAgentTools(input);
}

function validateWorkflowAgentProviderModelPair(input: WorkflowAgentRuntimeNodeValidation): void {
  const { context, nodeID, overrides } = input;
  const selection = resolveProviderModelSelection(overrides);
  if (hasPartialProviderModelSelection(selection)) {
    context.errors.push(`workflow agent node "${nodeID}" provider_name and model must be set together`);
  }
  if (hasCompleteProviderModelSelection(selection)) {
    validateWorkflowAgentProviderModel({
      nodeID,
      providerName: selection.providerName,
      model: selection.model,
      context,
    });
  }
}

function resolveProviderModelSelection(overrides: TaskRuntimeOverrides): ProviderModelSelection {
  return {
    providerName: normalizeProviderModelValue(overrides.provider_name),
    model: normalizeProviderModelValue(overrides.model),
  };
}

function normalizeProviderModelValue(value: string | undefined): string {
  return value?.trim() ?? '';
}

function hasPartialProviderModelSelection(selection: ProviderModelSelection): boolean {
  return hasProviderModelSelection(selection.providerName) !== hasProviderModelSelection(selection.model);
}

function hasCompleteProviderModelSelection(selection: ProviderModelSelection): boolean {
  return hasProviderModelSelection(selection.providerName) && hasProviderModelSelection(selection.model);
}

function hasProviderModelSelection(value: string): boolean {
  return value.length > 0;
}

function validateWorkflowAgentMaxTurns(input: WorkflowAgentRuntimeNodeValidation): void {
  const { context, nodeID, overrides } = input;
  if (!isValidWorkflowAgentMaxTurns(overrides.max_turns)) {
    context.errors.push(`workflow agent node "${nodeID}" max_turns must be > 0`);
  }
}

function isValidWorkflowAgentMaxTurns(maxTurns: number | undefined): boolean {
  return maxTurns === undefined || (Number.isInteger(maxTurns) && maxTurns >= 1);
}

function validateWorkflowAgentProviderModel(input: WorkflowAgentProviderModelValidation): void {
  const { context, model, nodeID, providerName } = input;
  const provider = findWorkflowAgentProvider(context.providers, providerName);
  if (!provider) {
    context.errors.push(`workflow agent node "${nodeID}" provider_name "${providerName}" is not available`);
    return;
  }
  if (!providerHasModel(provider, model)) {
    context.errors.push(`workflow agent node "${nodeID}" model "${model}" is not available for provider "${providerName}"`);
  }
}

function findWorkflowAgentProvider(
  providers: ProviderConfig[],
  providerName: string,
): ProviderConfig | undefined {
  return providers.find((item) => item.name.trim() === providerName);
}

function providerHasModel(provider: ProviderConfig, model: string): boolean {
  return providerModelNames(provider).has(model);
}

function providerModelNames(provider: ProviderConfig): Set<string> {
  return new Set((provider.models ?? []).map((item) => item.trim()).filter((item) => item.length > 0));
}

function validateWorkflowAgentTools(input: WorkflowAgentRuntimeNodeValidation): void {
  for (const toolName of input.overrides.tool_allowlist ?? []) {
    const normalized = toolName.trim();
    if (isAllowedWorkflowAgentTool(normalized, input.context.enabledToolNames)) {
      continue;
    }
    input.context.errors.push(`workflow agent node "${input.nodeID}" tool_allowlist contains disabled or unavailable tool "${normalized}"`);
  }
}

function isAllowedWorkflowAgentTool(
  toolName: string,
  enabledToolNames: ReadonlySet<string>,
): boolean {
  return toolName.length === 0 || enabledToolNames.has(toolName);
}
