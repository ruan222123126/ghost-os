import type { SettingsCopy } from '@/lib/i18n/messages/settings';
import type { PromptLibraryItem } from '@/lib/types';

export interface PromptLibraryEditorState {
  mode: 'create' | 'edit';
  original: PromptLibraryItem;
  draft: PromptLibraryItem;
}

const BASE_PROMPT_INSERT_POINT_OPTIONS = [
  { value: 'rule', labelKey: 'promptsLibraryInsertPointRule' as const },
  { value: 'core_job', labelKey: 'promptsLibraryInsertPointCoreJob' as const },
  { value: 'context', labelKey: 'promptsLibraryInsertPointContext' as const },
] as const;

const LEGACY_MEMORY_PROMPT_INSERT_POINT_OPTION = {
  value: 'memory',
  labelKey: 'promptsLibraryInsertPointMemory' as const,
};

export function promptInsertPointOptionsForEditor(
  insertPoint: PromptLibraryItem['insert_point'],
) {
  if (insertPoint !== 'memory') {
    return BASE_PROMPT_INSERT_POINT_OPTIONS;
  }
  return [...BASE_PROMPT_INSERT_POINT_OPTIONS, LEGACY_MEMORY_PROMPT_INSERT_POINT_OPTION];
}

export function normalizePromptLibraryCard(item: PromptLibraryItem): PromptLibraryItem {
  return {
    ...item,
    id: item.id.trim(),
    name: item.name.trim(),
    content: item.content.trim(),
  };
}

export function promptLibraryCardEqual(left: PromptLibraryItem, right: PromptLibraryItem): boolean {
  return left.id === right.id
    && left.name === right.name
    && left.insert_point === right.insert_point
    && left.content === right.content
    && left.active === right.active;
}

export function applyPromptLibraryActivationRules(
  previousPromptLibrary: PromptLibraryItem[],
  nextPromptLibrary: PromptLibraryItem[],
  cardID: string,
): PromptLibraryItem[] {
  const target = findPromptLibraryCard(nextPromptLibrary, cardID);
  if (!target || !target.active) {
    return nextPromptLibrary;
  }
  if (target.insert_point !== 'context') {
    return activateExclusiveInsertPoint(nextPromptLibrary, target.insert_point, cardID);
  }
  if (isActiveContextCard(findPromptLibraryCard(previousPromptLibrary, cardID))) {
    return nextPromptLibrary;
  }
  return moveContextCardAfterActiveSection(nextPromptLibrary, cardID);
}

export function newPromptLibraryCard(promptLibrary: PromptLibraryItem[]): PromptLibraryItem {
  const nextIndex = promptLibrary.length + 1;
  return {
    id: `prompt-card-${nextIndex}-${Date.now()}`,
    name: '',
    insert_point: 'core_job',
    content: '',
    active: false,
  };
}

export function labelForInsertPoint(insertPoint: PromptLibraryItem['insert_point'], settings: SettingsCopy): string {
  if (insertPoint === 'rule') {
    return settings.promptsLibraryInsertPointRule;
  }
  if (insertPoint === 'core_job') {
    return settings.promptsLibraryInsertPointCoreJob;
  }
  if (insertPoint === 'memory') {
    return settings.promptsLibraryInsertPointMemory;
  }
  if (insertPoint === 'context') {
    return settings.promptsLibraryInsertPointContext;
  }
  return insertPoint;
}

export function confirmPromptDeletion(message: string): boolean {
  if (typeof window === 'undefined') {
    return true;
  }
  return window.confirm(message);
}

function activateExclusiveInsertPoint(
  promptLibrary: PromptLibraryItem[],
  insertPoint: PromptLibraryItem['insert_point'],
  cardID: string,
): PromptLibraryItem[] {
  return promptLibrary.map((item) => {
    if (item.insert_point !== insertPoint) {
      return item;
    }
    return { ...item, active: item.id === cardID };
  });
}

function findPromptLibraryCard(
  promptLibrary: PromptLibraryItem[],
  cardID: string,
): PromptLibraryItem | undefined {
  return promptLibrary.find((item) => item.id === cardID);
}

function isActiveContextCard(item?: PromptLibraryItem): boolean {
  return item?.insert_point === 'context' && item.active;
}

function moveContextCardAfterActiveSection(
  promptLibrary: PromptLibraryItem[],
  cardID: string,
): PromptLibraryItem[] {
  const targetIndex = promptLibrary.findIndex((item) => item.id === cardID);
  if (targetIndex < 0) {
    return promptLibrary;
  }

  let destinationIndex = -1;
  for (const [index, item] of promptLibrary.entries()) {
    if (item.id === cardID) {
      continue;
    }
    if (isActiveContextCard(item)) {
      destinationIndex = index;
    }
  }
  if (destinationIndex < 0) {
    return promptLibrary;
  }

  const next = [...promptLibrary];
  const [target] = next.splice(targetIndex, 1);
  let insertIndex = destinationIndex + 1;
  if (targetIndex < insertIndex) {
    insertIndex -= 1;
  }
  next.splice(insertIndex, 0, target);
  return next;
}
