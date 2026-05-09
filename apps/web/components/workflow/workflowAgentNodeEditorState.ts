import type { WorkflowCopy } from '@/lib/i18n/messages/workflow';
import type { PresetPayload, TaskRuntimeOverrides, ToolPayload } from '@/lib/types';
import {
  availableWorkflowAgentToolNames,
  enabledWorkflowAgentToolNames,
  type WorkflowAgentRuntimeCatalog,
} from '@/lib/workflow-editor';

export interface SelectOption {
  disabled: boolean;
  label: string;
  value: string;
}

export function buildProviderOptions(
  catalog: WorkflowAgentRuntimeCatalog | undefined,
  currentValue: string,
  copy: WorkflowCopy,
): SelectOption[] {
  const mapped = (catalog?.providers ?? [])
    .slice()
    .sort((left, right) => left.name.localeCompare(right.name))
    .map((provider) => ({
      value: provider.name,
      label: `${provider.name} (${provider.type})`,
      disabled: false,
    }));
  return withCurrentDisabledOption(mapped, currentValue, copy.agentProviderUnavailable);
}

export function buildModelOptions(
  catalog: WorkflowAgentRuntimeCatalog | undefined,
  providerName: string,
  currentValue: string,
  copy: WorkflowCopy,
): SelectOption[] {
  const provider = (catalog?.providers ?? []).find((item) => item.name === providerName);
  const mapped = (provider?.models ?? []).map((model) => ({
    value: model,
    label: model,
    disabled: false,
  }));
  return withCurrentDisabledOption(mapped, currentValue, copy.agentModelUnavailable);
}

export function buildPresetOptions(
  presets: PresetPayload[],
  currentValue: string,
  copy: WorkflowCopy,
): SelectOption[] {
  const mapped = presets
    .slice()
    .sort((left, right) => left.name.localeCompare(right.name))
    .map((preset) => ({
      value: preset.id,
      label: preset.name,
      disabled: false,
    }));
  return withCurrentDisabledOption(mapped, currentValue, copy.agentPresetUnavailable);
}

export function buildToolOptions(
  tools: ToolPayload[],
  selectedNames: string[],
  copy: WorkflowCopy,
  showAllTools: boolean,
): SelectOption[] {
  const visible = showAllTools ? tools : tools.filter((tool) => tool.enabled);
  const mapped = visible
    .slice()
    .sort((left, right) => left.name.localeCompare(right.name))
    .map((tool) => ({ value: tool.name, label: tool.name, disabled: false }));
  return selectedNames.reduce((options, name) => {
    return withCurrentDisabledOption(options, name, copy.agentToolUnavailable);
  }, mapped);
}

export function toolOrderForEditor(
  tools: ToolPayload[],
  showAllTools: boolean,
): string[] {
  return showAllTools
    ? availableWorkflowAgentToolNames(tools)
    : enabledWorkflowAgentToolNames(tools);
}

export function nextProviderOverrides(
  current: TaskRuntimeOverrides,
  providerName: string,
  catalog: WorkflowAgentRuntimeCatalog | undefined,
): TaskRuntimeOverrides | undefined {
  const provider = catalog?.providers.find((item) => item.name === providerName);
  const nextModel = (provider?.models ?? []).includes(current.model ?? '') ? current.model : undefined;
  return nextRuntimeOverrides(current, {
    provider_name: providerName || undefined,
    model: nextModel,
  });
}

export function toggleToolOverride(
  current: TaskRuntimeOverrides,
  toolName: string,
  orderedToolNames: string[],
): TaskRuntimeOverrides | undefined {
  const selected = new Set(current.tool_allowlist ?? []);
  if (selected.has(toolName)) {
    selected.delete(toolName);
  } else {
    selected.add(toolName);
  }
  return nextRuntimeOverrides(current, {
    tool_allowlist_only: true,
    tool_allowlist: orderSelectedToolNames([...selected], orderedToolNames),
  });
}

export function applyPresetOverride(
  current: TaskRuntimeOverrides,
  preset: PresetPayload,
  orderedToolNames: string[],
): TaskRuntimeOverrides | undefined {
  return nextRuntimeOverrides(current, {
    preset_id: preset.id,
    system_prompt: undefined,
    tool_allowlist_only: true,
    tool_allowlist: orderSelectedToolNames(preset.tool_allowlist, orderedToolNames),
  });
}

export function nextRuntimeOverrides(
  current: TaskRuntimeOverrides,
  patch: Partial<TaskRuntimeOverrides>,
): TaskRuntimeOverrides | undefined {
  const next = { ...current, ...patch };
  if (next.provider_name === undefined) {
    next.model = undefined;
  }
  if (next.tool_allowlist_only === true && next.tool_allowlist === undefined) {
    next.tool_allowlist = [];
  }
  if (!hasMeaningfulRuntimeOverrides(next)) {
    return undefined;
  }
  return next;
}

export function parseOptionalPositiveInteger(raw: string): number | undefined {
  const value = raw.trim();
  if (value.length === 0) {
    return undefined;
  }
  const parsed = Number.parseInt(value, 10);
  return Number.isFinite(parsed) ? parsed : undefined;
}

function orderSelectedToolNames(selectedNames: string[], orderedToolNames: string[]): string[] {
  const orderedSet = new Set(orderedToolNames);
  const normalized = selectedNames.map((name) => name.trim()).filter((name) => name.length > 0);
  const selectedSet = new Set(normalized);
  const ordered = orderedToolNames.filter((name) => selectedSet.has(name));
  const missing = normalized.filter((name) => !orderedSet.has(name));
  return [...ordered, ...missing];
}

function withCurrentDisabledOption(
  options: SelectOption[],
  currentValue: string,
  unavailableLabel: (name: string) => string,
): SelectOption[] {
  const normalized = currentValue.trim();
  if (normalized.length === 0 || options.some((option) => option.value === normalized)) {
    return options;
  }
  return [{
    value: normalized,
    label: unavailableLabel(normalized),
    disabled: true,
  }, ...options];
}

function hasMeaningfulRuntimeOverrides(overrides: TaskRuntimeOverrides): boolean {
  return Boolean(
    overrides.provider_name
    || overrides.model
    || overrides.system_prompt
    || overrides.preset_id
    || overrides.tool_allowlist_only
    || overrides.tool_allowlist?.length
    || overrides.max_turns,
  );
}
