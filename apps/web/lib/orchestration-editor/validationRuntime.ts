import type {
  PresetPayload,
  ProviderConfig,
  TaskRuntimeOverrides,
} from '@/lib/types';
import { availableWorkflowAgentToolNames, type WorkflowAgentRuntimeCatalog } from '@/lib/workflow-editor/agentRuntime';
import type {
  WorkflowCanvasDraft,
  WorkflowCanvasNodeDraft,
} from '@/lib/workflow-editor/types';

export interface OrchestrationRuntimeValidationOptions {
  agentRuntimeCatalog?: WorkflowAgentRuntimeCatalog;
  presets?: PresetPayload[];
}

interface RuntimeValidationContext {
  providers: ProviderConfig[];
  presetIDs: Set<string>;
  tools: Set<string>;
}

interface ProviderModelValidationOptions {
  nodeID: string;
  providerName: string;
  model: string;
  providers: ProviderConfig[];
  errors: string[];
}

interface RuntimeOverrideValidationOptions {
  nodeID: string;
  overrides: TaskRuntimeOverrides;
  context: RuntimeValidationContext;
  errors: string[];
}

interface PresetIDValidationOptions {
  nodeID: string;
  rawPresetID?: string;
  presetIDs: Set<string>;
  errors: string[];
}

interface ToolAllowlistValidationOptions {
  nodeID: string;
  toolAllowlist: string[];
  tools: Set<string>;
  errors: string[];
}

export function validateOrchestrationAgentRuntime(
  draft: WorkflowCanvasDraft,
  options?: OrchestrationRuntimeValidationOptions,
): string[] {
  if (!options?.agentRuntimeCatalog) {
    return [];
  }
  const errors: string[] = [];
  const context = buildRuntimeValidationContext(options.agentRuntimeCatalog, options.presets ?? []);
  for (const node of draft.nodes) {
    validateAgentRuntimeNode(node, context, errors);
  }
  return errors;
}

function buildRuntimeValidationContext(
  catalog: WorkflowAgentRuntimeCatalog,
  presets: PresetPayload[],
): RuntimeValidationContext {
  return {
    providers: catalog.providers,
    presetIDs: buildPresetIDs(presets),
    tools: new Set(availableWorkflowAgentToolNames(catalog.tools)),
  };
}

function buildPresetIDs(presets: PresetPayload[]): Set<string> {
  return new Set(presets.map((preset) => preset.id.trim()).filter((presetID) => presetID.length > 0));
}

function validateAgentRuntimeNode(
  node: WorkflowCanvasNodeDraft,
  context: RuntimeValidationContext,
  errors: string[],
): void {
  if (node.type !== 'agent' || !node.agent?.runtime_overrides) {
    return;
  }
  validateRuntimeOverrides({
    nodeID: node.id,
    overrides: node.agent.runtime_overrides,
    context,
    errors,
  });
}

function validateRuntimeOverrides(options: RuntimeOverrideValidationOptions): void {
  const { nodeID, overrides, context, errors } = options;
  validateProviderAndModel({
    nodeID,
    providerName: overrides.provider_name ?? '',
    model: overrides.model ?? '',
    providers: context.providers,
    errors,
  });
  validatePresetID({
    nodeID,
    rawPresetID: overrides.preset_id,
    presetIDs: context.presetIDs,
    errors,
  });
  validateToolAllowlist({
    nodeID,
    toolAllowlist: overrides.tool_allowlist ?? [],
    tools: context.tools,
    errors,
  });
}

function validatePresetID(options: PresetIDValidationOptions): void {
  const { nodeID, rawPresetID, presetIDs, errors } = options;
  const presetID = rawPresetID?.trim() ?? '';
  if (presetID.length > 0 && !presetIDs.has(presetID)) {
    errors.push(`orchestration agent node "${nodeID}" preset_id "${presetID}" is not available`);
  }
}

function validateToolAllowlist(options: ToolAllowlistValidationOptions): void {
  const { nodeID, toolAllowlist, tools, errors } = options;
  for (const toolName of toolAllowlist) {
    const normalizedToolName = toolName.trim();
    if (!tools.has(normalizedToolName)) {
      errors.push(`orchestration agent node "${nodeID}" tool_allowlist contains unavailable tool "${normalizedToolName}"`);
    }
  }
}

function validateProviderAndModel(options: ProviderModelValidationOptions): void {
  const { providerName, model } = options;
  if (!hasProviderModelInput(providerName, model)) {
    return;
  }
  if (!hasCompleteProviderModel(providerName, model)) {
    options.errors.push(`orchestration agent node "${options.nodeID}" provider_name and model must be set together`);
    return;
  }
  validateProviderModelAvailability(options);
}

function hasProviderModelInput(providerName: string, model: string): boolean {
  return Boolean(providerName || model);
}

function hasCompleteProviderModel(providerName: string, model: string): boolean {
  return Boolean(providerName && model);
}

function validateProviderModelAvailability(options: ProviderModelValidationOptions): void {
  const provider = findProvider(options.providers, options.providerName);
  if (!provider) {
    options.errors.push(`orchestration agent node "${options.nodeID}" provider_name "${options.providerName}" is not available`);
    return;
  }
  if (!providerHasModel(provider, options.model)) {
    options.errors.push(`orchestration agent node "${options.nodeID}" model "${options.model}" is not available for provider "${options.providerName}"`);
  }
}

function findProvider(providers: ProviderConfig[], providerName: string): ProviderConfig | undefined {
  return providers.find((item) => item.name.trim() === providerName.trim());
}

function providerHasModel(provider: ProviderConfig, model: string): boolean {
  return (provider.models ?? []).includes(model);
}
