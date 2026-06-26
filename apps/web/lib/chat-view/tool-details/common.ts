import {
  parseRecord,
  readPositiveInt,
  readRecord,
  readRecordArray,
  readString,
  type RecordValue,
} from './records';
import {
  normalizeToolName,
  SCRIPT_EXEC_TOOL,
  supportsPromotedActionTitle,
} from '@/lib/toolNames';
import { parseInlineArgs } from './inlineArgs';

const SUMMARY_LIMIT = 80;
const SUMMARY_TRUNCATE_AT = 77;
const ERROR_LIMIT = 160;
const ERROR_TRUNCATE_AT = 157;
const WINDOWS_DRIVE_ROOT_PATTERN = /^[A-Za-z]:\/$/;
const ERROR_STATUSES = new Set(['error', 'failed']);
const SCRIPT_EXEC_HELPERS = [
  'apply_diff',
  'bash_exec',
  'fetch_webpage',
  'list_files',
  'read_file',
  'search_files',
  'write_file',
] as const;

export const ACTION_ORDER = ['list', 'load', 'unload', 'read', 'write', 'edit', 'run', 'screen', 'web', 'search', 'codex'] as const;
export type ActionKind = typeof ACTION_ORDER[number];
export interface ActionItem { kind: ActionKind; text: string; }
interface LineDelta { added?: number; removed?: number; }
interface DiffLineCountState { added: number; removed: number; inHunk: boolean; }
type DirectActionBuilder = (args?: RecordValue) => ActionItem | undefined;

const DIRECT_ACTION_BUILDERS: Record<string, DirectActionBuilder> = {
  [SCRIPT_EXEC_TOOL]: (args) => ({ kind: 'run', text: truncateSummary(readString(args, 'script')) }),
  codex_cli: (args) => ({ kind: 'codex', text: formatCodexTarget(args) }),
  read_file: (args) => ({ kind: 'read', text: formatPath(readString(args, 'path')) }),
  list_files: (args) => ({ kind: 'list', text: formatPath(readString(args, 'path')) || '.' }),
  write_file: (args) => ({ kind: 'write', text: appendDelta(formatPath(readString(args, 'path')), resolveWriteDelta(undefined, args)) }),
  apply_diff: (args) => ({ kind: 'edit', text: appendDelta(formatPath(readString(args, 'path')), resolveEditDelta(undefined, args)) }),
  bash_exec: (args) => ({ kind: 'run', text: truncateSummary(readString(args, 'command') || readString(args, 'cmd')) }),
  screen_action: buildScreenAction,
  screen_control: buildScreenAction,
  fetch_webpage: (args) => ({ kind: 'web', text: truncateSummary(formatWebTarget(readString(args, 'url'))) }),
  web_search: (args) => ({ kind: 'web', text: truncateSummary(readString(args, 'query')) }),
  search_files: (args) => ({ kind: 'search', text: truncateSummary(readString(args, 'query')) }),
};

export {
  parseRecord,
  readPositiveInt,
  readRecord,
  readString,
  type RecordValue,
} from './records';
export { parseInlineArgs } from './inlineArgs';
export { normalizeToolName, supportsPromotedActionTitle } from '@/lib/toolNames';

export function buildDirectAction(toolName: string, args?: RecordValue): ActionItem | undefined {
  return DIRECT_ACTION_BUILDERS[toolName]?.(args);
}

function buildScreenAction(args?: RecordValue): ActionItem | undefined {
  const action = normalizeToolName(readString(args, 'action'));
  if (action === 'screenshot') {
    return { kind: 'screen', text: '' };
  }
  return undefined;
}

export function buildWriteChangeAction(writeChange?: RecordValue, args?: RecordValue): ActionItem | undefined {
  if (!writeChange) return undefined;
  const operation = normalizeToolName(readString(writeChange, 'operation'));
  const pathText = formatPath(readString(writeChange, 'path') || readString(args, 'path'));
  if (operation === 'write_file') {
    return { kind: 'write', text: appendDelta(pathText, resolveWriteDelta(writeChange, args)) };
  }
  if (operation === 'apply_diff') {
    return { kind: 'edit', text: appendDelta(pathText, resolveEditDelta(writeChange, args)) };
  }
  return undefined;
}

function resolveWriteDelta(writeChange?: RecordValue, args?: RecordValue): LineDelta {
  const delta: LineDelta = {
    added: readPositiveInt(writeChange, 'added_lines') || readPositiveInt(writeChange, 'content_lines'),
    removed: readPositiveInt(writeChange, 'removed_lines'),
  };
  if (!delta.added) {
    delta.added = readPositiveInt(readRecord(args, 'content_summary'), 'lines') || readPositiveInt(args, 'added_lines');
  }
  if (!delta.added) {
    delta.added = countLines(readString(args, 'content'));
  }
  if (!delta.removed) {
    delta.removed = readPositiveInt(args, 'removed_lines');
  }
  return delta;
}

function resolveEditDelta(writeChange?: RecordValue, args?: RecordValue): LineDelta {
  const diffSummary = readRecord(args, 'diff_summary');
  const delta: LineDelta = {
    added: readPositiveInt(writeChange, 'added_lines') || readPositiveInt(diffSummary, 'added_lines'),
    removed: readPositiveInt(writeChange, 'removed_lines') || readPositiveInt(diffSummary, 'removed_lines'),
  };
  if (!delta.added || !delta.removed) {
    const counted = countUnifiedDiffDelta(readString(args, 'diff_text'));
    delta.added ||= counted.added;
    delta.removed ||= counted.removed;
  }
  return delta;
}

function countUnifiedDiffDelta(diffText: string): LineDelta {
  if (!diffText.trim()) return {};
  const state: DiffLineCountState = { added: 0, removed: 0, inHunk: false };
  for (const line of diffText.split(/\r?\n/)) {
    countUnifiedDiffLine(state, line);
  }
  return lineDeltaFromCounts(state);
}

function countUnifiedDiffLine(state: DiffLineCountState, line: string): void {
  if (line.startsWith('@@')) {
    state.inHunk = true;
    return;
  }
  if (shouldIgnoreDiffLine(state, line)) {
    return;
  }
  if (isAddedDiffLine(line)) {
    state.added += 1;
    return;
  }
  if (isRemovedDiffLine(line)) {
    state.removed += 1;
  }
}

function shouldIgnoreDiffLine(state: DiffLineCountState, line: string): boolean {
  return !state.inHunk || line.startsWith('\\');
}

function isAddedDiffLine(line: string): boolean {
  return line.startsWith('+');
}

function isRemovedDiffLine(line: string): boolean {
  return line.startsWith('-');
}

function lineDeltaFromCounts(state: DiffLineCountState): LineDelta {
  return {
    added: positiveCount(state.added),
    removed: positiveCount(state.removed),
  };
}

function positiveCount(value: number): number | undefined {
  return value > 0 ? value : undefined;
}

export function parseScriptExecScriptActions(rawText?: string): ActionItem[] {
  const payload = parseRecord(rawText);
  const script = readString(payload, 'script');
  if (!script) return [];

  const actions: ActionItem[] = [];
  const pattern = new RegExp(
    `(^|[^\\w.])(?:tools\\.)?(${SCRIPT_EXEC_HELPERS.join('|')})\\s*\\(([\\s\\S]*?)\\)`,
    'gm',
  );
  for (const match of script.matchAll(pattern)) {
    const toolName = normalizeToolName(match[2]);
    if (!toolName) continue;
    const action = buildDirectAction(toolName, parseInlineArgs(toolName, match[3]?.trim() || ''));
    if (action) actions.push(action);
  }
  return actions;
}

function appendDelta(baseText: string, delta: LineDelta): string {
  const parts: string[] = [];
  if (delta.added) parts.push(`+${delta.added}`);
  if (delta.removed) parts.push(`-${delta.removed}`);
  if (parts.length === 0) return baseText;
  return baseText ? `${baseText} ${parts.join(' ')}` : parts.join(' ');
}

function formatWebTarget(rawURL: string): string {
  if (!rawURL) return '';
  try {
    const parsed = new URL(rawURL);
    const pathname = parsed.pathname === '/' ? '' : parsed.pathname;
    return `${parsed.hostname}${pathname}${parsed.search}`;
  } catch {
    return rawURL;
  }
}

function formatCodexTarget(args?: RecordValue): string {
  const op = readString(args, 'op');
  const prompt = truncateSummary(readString(args, 'prompt'));
  const parts = [op, prompt].filter(Boolean);
  return truncateSummary(parts.join(' '));
}

function formatPath(rawPath: string): string {
  if (!rawPath) return '';
  const normalized = rawPath.replaceAll('\\', '/');
  if (normalized === '/') {
    return normalized;
  }
  if (WINDOWS_DRIVE_ROOT_PATTERN.test(normalized)) {
    return normalized;
  }
  return normalized.replace(/\/+$/, '');
}

function countLines(content: string): number | undefined {
  if (!content.trim()) return undefined;
  return content.split('\n').length;
}

export function parseScriptExecReport(rawText?: string): { steps: RecordValue[] } | undefined {
  const record = parseRecord(rawText);
  if (!record || !readRecord(record, 'summary')) return undefined;
  const steps = readRecordArray(record, 'steps');
  return steps ? { steps } : undefined;
}

export function readScriptExecReportError(rawText?: string): string {
  const report = parseScriptExecReport(rawText);
  if (!report) return '';
  for (const step of report.steps) {
    const errorText = normalizeInline(readString(step, 'error'));
    if (errorText) return errorText;
  }
  return '';
}

export function truncateSummary(value: string): string {
  const normalized = normalizeInline(value);
  if (!normalized) return '';
  if (normalized.length <= SUMMARY_LIMIT) return normalized;
  return `${normalized.slice(0, SUMMARY_TRUNCATE_AT)}...`;
}

export function truncateError(value: string): string {
  if (value.length <= ERROR_LIMIT) return value;
  return `${value.slice(0, ERROR_TRUNCATE_AT)}...`;
}

export function normalizeInline(value?: string): string {
  if (!value) return '';
  return value.trim().replace(/\s+/g, ' ');
}

export function isErrorStatus(status?: string): boolean {
  const normalized = status?.trim().toLowerCase() || '';
  return ERROR_STATUSES.has(normalized);
}
