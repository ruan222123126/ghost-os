import type { SettingsCopy } from '@/lib/i18n/messages/settings';
import type {
  PresetPayload,
  PresetPromptRefs,
  PromptLibraryItem,
  ToolPayload,
} from '@/lib/types';

export type PresetPromptRefSlot = 'rule' | 'core_job';

export interface PresetEditorDraft {
  name: string;
  tool_allowlist: string[];
  prompt_refs: PresetPromptRefs;
}

export interface PresetEditorState {
  mode: 'create' | 'edit';
  presetID?: string;
  original: PresetEditorDraft;
  draft: PresetEditorDraft;
}

export const PRESET_PROMPT_REF_FIELDS = [
  { slot: 'rule', insertPoint: 'rule' },
  { slot: 'core_job', insertPoint: 'core_job' },
] as const;

export function newPresetDraft(): PresetEditorDraft {
  return {
    name: '',
    tool_allowlist: [],
    prompt_refs: {},
  };
}

export function presetToDraft(preset: PresetPayload): PresetEditorDraft {
  return {
    name: preset.name,
    tool_allowlist: [...preset.tool_allowlist],
    prompt_refs: clonePromptRefs(preset.prompt_refs),
  };
}

export function normalizePresetDraft(draft: PresetEditorDraft): PresetEditorDraft {
  return {
    name: draft.name.trim(),
    tool_allowlist: normalizePresetToolAllowlist(draft.tool_allowlist),
    prompt_refs: normalizePresetPromptRefs(draft.prompt_refs),
  };
}

export function presetDraftEqual(left: PresetEditorDraft, right: PresetEditorDraft): boolean {
  const normalizedLeft = normalizePresetDraft(left);
  const normalizedRight = normalizePresetDraft(right);

  return normalizedLeft.name === normalizedRight.name
    && listEqual(normalizedLeft.tool_allowlist, normalizedRight.tool_allowlist)
    && normalizedLeft.prompt_refs.rule === normalizedRight.prompt_refs.rule
    && normalizedLeft.prompt_refs.core_job === normalizedRight.prompt_refs.core_job
    && normalizedLeft.prompt_refs.memory === normalizedRight.prompt_refs.memory
    && listEqual(normalizedLeft.prompt_refs.context ?? [], normalizedRight.prompt_refs.context ?? []);
}

export function togglePresetToolSelection(
  toolAllowlist: string[],
  toolName: string,
  selected: boolean,
): string[] {
  const names = selected
    ? [...toolAllowlist, toolName]
    : toolAllowlist.filter((item) => item !== toolName);

  return normalizePresetToolAllowlist(names);
}

export function presetPromptSlotLabel(settings: SettingsCopy, slot: PresetPromptRefSlot): string {
  if (slot === 'rule') {
    return settings.promptsLibraryInsertPointRule;
  }
  return settings.promptsLibraryInsertPointCoreJob;
}

export function confirmPresetDeletion(message: string): boolean {
  if (typeof window === 'undefined') {
    return true;
  }
  return window.confirm(message);
}

export function togglePresetContextSelection(
  currentRefs: string[] | undefined,
  promptID: string,
  selected: boolean,
): string[] | undefined {
  const refs = currentRefs ?? [];
  const nextRefs = selected
    ? [...refs, promptID]
    : refs.filter((item) => item !== promptID);
  const normalized = normalizePresetContextRefs(nextRefs);
  return normalized.length === 0 ? undefined : normalized;
}

export function findActivePresetID(
  presets: PresetPayload[],
  tools: ToolPayload[],
  promptLibrary: PromptLibraryItem[],
  modelSelectionEnabled: boolean | null,
): string | null {
  if (modelSelectionEnabled !== false) {
    return null;
  }

  const currentToolAllowlist = normalizePresetToolAllowlist(
    tools.filter((tool) => tool.enabled).map((tool) => tool.name),
  );
  const currentPromptRefs = activePresetPromptRefs(promptLibrary);

  for (const preset of presets) {
    if (!presetMatchesCurrentState(preset, currentToolAllowlist, currentPromptRefs)) {
      continue;
    }
    return preset.id;
  }
  return null;
}

function normalizePresetToolAllowlist(toolAllowlist: string[]): string[] {
  const seen = new Set<string>();
  const normalized = toolAllowlist
    .map((item) => item.trim())
    .filter((item) => item !== '')
    .filter((item) => {
      if (seen.has(item)) {
        return false;
      }
      seen.add(item);
      return true;
    });

  return normalized.sort((left, right) => left.localeCompare(right));
}

function normalizePresetPromptRefs(promptRefs: PresetPromptRefs): PresetPromptRefs {
  const normalized: PresetPromptRefs = {};
  const rule = promptRefs.rule?.trim();
  const coreJob = promptRefs.core_job?.trim();
  const memory = promptRefs.memory?.trim();
  const context = normalizePresetContextRefs(promptRefs.context ?? []);

  if (rule) {
    normalized.rule = rule;
  }
  if (coreJob) {
    normalized.core_job = coreJob;
  }
  if (memory) {
    normalized.memory = memory;
  }
  if (context.length > 0) {
    normalized.context = context;
  }
  return normalized;
}

function listEqual(left: string[], right: string[]): boolean {
  if (left.length !== right.length) {
    return false;
  }
  return left.every((item, index) => item === right[index]);
}

function presetMatchesCurrentState(
  preset: PresetPayload,
  toolAllowlist: string[],
  promptRefs: PresetPromptRefs,
): boolean {
  const normalizedPreset = normalizePresetDraft(presetToDraft(preset));

  return listEqual(normalizedPreset.tool_allowlist, toolAllowlist)
    && normalizedPreset.prompt_refs.rule === promptRefs.rule
    && normalizedPreset.prompt_refs.core_job === promptRefs.core_job
    && normalizedPreset.prompt_refs.memory === promptRefs.memory
    && listEqual(normalizedPreset.prompt_refs.context ?? [], promptRefs.context ?? []);
}

function activePresetPromptRefs(promptLibrary: PromptLibraryItem[]): PresetPromptRefs {
  const refs: PresetPromptRefs = {};
  const contextRefs: string[] = [];

  for (const item of promptLibrary) {
    if (!item.active) {
      continue;
    }
    if (item.insert_point === 'rule') {
      refs.rule = item.id;
      continue;
    }
    if (item.insert_point === 'core_job') {
      refs.core_job = item.id;
      continue;
    }
    if (item.insert_point === 'memory') {
      refs.memory = item.id;
      continue;
    }
    contextRefs.push(item.id);
  }

  if (contextRefs.length > 0) {
    refs.context = contextRefs;
  }
  return refs;
}

function clonePromptRefs(promptRefs: PresetPromptRefs): PresetPromptRefs {
  if (!promptRefs.context) {
    return { ...promptRefs };
  }
  return {
    ...promptRefs,
    context: [...promptRefs.context],
  };
}

function normalizePresetContextRefs(promptRefs: string[]): string[] {
  const seen = new Set<string>();
  return promptRefs
    .map((item) => item.trim())
    .filter((item) => item !== '')
    .filter((item) => {
      if (seen.has(item)) {
        return false;
      }
      seen.add(item);
      return true;
    });
}
