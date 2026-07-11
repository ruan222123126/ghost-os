import type { ToolChatMessage } from '@/lib/types';
import {
  type ActionItem,
  normalizeToolName,
  parseRecord,
  readString,
  truncateSummary,
  type RecordValue,
} from './common';
import { resolveToolCallArgs } from './toolCalls';

const TOOL_SEARCH_NAME = 'sfind';
const TOOL_SEARCH_ACTION_SEARCH = 'search';
const TOOL_SEARCH_ACTION_LOAD = 'load';
const TOOL_SEARCH_ACTION_UNLOAD = 'unload';
const TOOL_SEARCH_ACTION_LIST = 'list';

export function buildToolSearchAction(tool: ToolChatMessage): ActionItem | undefined {
  if (normalizeToolName(tool.toolName) !== TOOL_SEARCH_NAME) {
    return undefined;
  }

  const payload = parseRecord(tool.rawOutput) || parseRecord(tool.content);
  const args = resolveToolCallArgs(tool);
  const action = normalizeToolSearchAction(readString(payload, 'action') || readString(args, 'action'));
  if (!action) {
    return undefined;
  }

  switch (action) {
    case TOOL_SEARCH_ACTION_SEARCH:
      return { kind: 'search', text: summarizeSearchAction(payload, args) };
    case TOOL_SEARCH_ACTION_LOAD:
      return { kind: 'load', text: summarizeNamedAction(payload, args, 'skill_names') };
    case TOOL_SEARCH_ACTION_UNLOAD:
      return { kind: 'unload', text: summarizeNamedAction(payload, args, 'skill_names') };
    case TOOL_SEARCH_ACTION_LIST:
      return { kind: 'list', text: summarizeListAction(payload) || 'skills' };
    default:
      return undefined;
  }
}

function summarizeSearchAction(payload?: RecordValue, args?: RecordValue): string {
  const query = truncateSummary(readString(args, 'query'));
  if (query) {
    return query;
  }

  const names = readItemNames(payload);
  return names.length > 0 ? truncateSummary(names.join(', ')) : 'skills';
}

function summarizeNamedAction(payload: RecordValue | undefined, args: RecordValue | undefined, key: string): string {
  const names = readItemNames(payload);
  if (names.length > 0) {
    return truncateSummary(names.join(', '));
  }

  const requested = readStringList(args, key);
  return requested.length > 0 ? truncateSummary(requested.join(', ')) : 'skills';
}

function summarizeListAction(payload?: RecordValue): string {
  const items = readItems(payload, 'items');
  if (items.length === 0) {
    return '';
  }

  const entries = items
    .map((item) => formatStatusEntry(readString(item, 'name'), readString(item, 'status')))
    .filter(Boolean);
  return truncateSummary(entries.join(', '));
}

function formatStatusEntry(name: string, status: string): string {
  if (!name) {
    return '';
  }
  if (!status) {
    return name;
  }
  return `${name}(${normalizeToolSearchAction(status)})`;
}

function readItemNames(payload?: RecordValue): string[] {
  return readItems(payload, 'items')
    .map((item) => readString(item, 'name'))
    .filter(Boolean);
}

function readItems(value: RecordValue | undefined, key: string): RecordValue[] {
  const items = value?.[key];
  if (!Array.isArray(items)) {
    return [];
  }
  return items.filter(isRecordValue);
}

function readStringList(value: RecordValue | undefined, key: string): string[] {
  const items = value?.[key];
  if (!Array.isArray(items)) {
    return [];
  }
  return items
    .filter((item): item is string => typeof item === 'string')
    .map((item) => item.trim())
    .filter(Boolean);
}

function normalizeToolSearchAction(value: string): string {
  return value.trim().toLowerCase();
}

function isRecordValue(value: unknown): value is RecordValue {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}
