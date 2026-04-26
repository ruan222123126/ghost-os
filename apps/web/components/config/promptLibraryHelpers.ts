import type { SettingsCopy } from '@/lib/i18n/messages/settings';
import type { PromptLibraryItem } from '@/lib/types';

export interface PromptLibraryEditorState {
  mode: 'create' | 'edit';
  original: PromptLibraryItem;
  draft: PromptLibraryItem;
}

export const PROMPT_INSERT_POINT_OPTIONS = [
  { value: 'core_job', labelKey: 'promptsLibraryInsertPointCoreJob' as const },
  { value: 'memory', labelKey: 'promptsLibraryInsertPointMemory' as const },
] as const;

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

export function enforceExclusiveActivation(promptLibrary: PromptLibraryItem[], cardID: string): PromptLibraryItem[] {
  const target = promptLibrary.find((item) => item.id === cardID);
  if (!target || !target.active) {
    return promptLibrary;
  }
  return promptLibrary.map((item) => {
    if (item.insert_point === target.insert_point) {
      return { ...item, active: item.id === target.id };
    }
    return item;
  });
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
  if (insertPoint === 'core_job') {
    return settings.promptsLibraryInsertPointCoreJob;
  }
  if (insertPoint === 'memory') {
    return settings.promptsLibraryInsertPointMemory;
  }
  return insertPoint;
}

export function confirmPromptDeletion(message: string): boolean {
  if (typeof window === 'undefined') {
    return true;
  }
  return window.confirm(message);
}
