import {
  parseRecord,
  readPositiveInt,
  readRecord,
  readRecordArray,
  readString,
  type RecordValue,
} from './records';

const SUMMARY_LIMIT = 80;
const SUMMARY_TRUNCATE_AT = 77;
const ERROR_LIMIT = 160;
const ERROR_TRUNCATE_AT = 157;
const WINDOWS_DRIVE_ROOT_PATTERN = /^[A-Za-z]:\/$/;
const TOOL_PREFIX = 'tools.';
const SCRIPT_EXEC_TOOL = 'script_exec';
const ERROR_STATUSES = new Set(['error', 'failed']);
const PROMOTED_ACTION_TOOL_NAMES = new Set([
  'apply_diff',
  'bash_exec',
  'list_files',
  'read_file',
  'search_files',
  'sfind',
  'write_file',
]);
const SCRIPT_EXEC_HELPERS = [
  'apply_diff',
  'bash_exec',
  'fetch_webpage',
  'list_files',
  'read_file',
  'search_files',
  'write_file',
] as const;

export const ACTION_ORDER = ['list', 'load', 'unload', 'read', 'write', 'edit', 'run', 'web', 'search'] as const;
export type ActionKind = typeof ACTION_ORDER[number];
export interface ActionItem { kind: ActionKind; text: string; }
interface LineDelta { added?: number; removed?: number; }

export {
  parseRecord,
  readPositiveInt,
  readRecord,
  readString,
  type RecordValue,
} from './records';

const TOOL_ALIASES: Record<string, string> = {
  apply_diff: 'apply_diff',
  bash: 'bash_exec',
  bash_exec: 'bash_exec',
  edit: 'apply_diff',
  fetch_web: 'fetch_webpage',
  fetch_webpage: 'fetch_webpage',
  patch: 'apply_diff',
  read: 'read_file',
  read_file: 'read_file',
  run: 'bash_exec',
  script: SCRIPT_EXEC_TOOL,
  script_exec: SCRIPT_EXEC_TOOL,
  search: 'search_files',
  search_files: 'search_files',
  web: 'fetch_webpage',
  write: 'write_file',
  write_file: 'write_file',
};

export function normalizeToolName(rawName?: string): string {
  const normalized = rawName?.trim().toLowerCase() || '';
  if (!normalized) return '';
  const withoutPrefix = normalized.startsWith(TOOL_PREFIX)
    ? normalized.slice(TOOL_PREFIX.length)
    : normalized;
  if (TOOL_ALIASES[withoutPrefix]) return TOOL_ALIASES[withoutPrefix];
  for (const suffix of ['_exec', '_file']) {
    if (!withoutPrefix.endsWith(suffix)) continue;
    const baseName = withoutPrefix.slice(0, withoutPrefix.length - suffix.length);
    if (TOOL_ALIASES[baseName]) return TOOL_ALIASES[baseName];
  }
  return withoutPrefix;
}

export function supportsPromotedActionTitle(toolName?: string): boolean {
  return PROMOTED_ACTION_TOOL_NAMES.has(normalizeToolName(toolName));
}

export function buildDirectAction(toolName: string, args?: RecordValue): ActionItem | undefined {
  if (!toolName) return undefined;
  if (toolName === SCRIPT_EXEC_TOOL) {
    return { kind: 'run', text: truncateSummary(readString(args, 'script')) };
  }
  if (toolName === 'read_file') {
    return { kind: 'read', text: formatPath(readString(args, 'path')) };
  }
  if (toolName === 'list_files') {
    return { kind: 'list', text: formatPath(readString(args, 'path')) || '.' };
  }
  if (toolName === 'write_file') {
    const delta = resolveWriteDelta(undefined, args);
    return { kind: 'write', text: appendDelta(formatPath(readString(args, 'path')), delta) };
  }
  if (toolName === 'apply_diff') {
    const delta = resolveEditDelta(undefined, args);
    return { kind: 'edit', text: appendDelta(formatPath(readString(args, 'path')), delta) };
  }
  if (toolName === 'bash_exec') {
    return { kind: 'run', text: truncateSummary(readString(args, 'command') || readString(args, 'cmd')) };
  }
  if (toolName === 'fetch_webpage') {
    return { kind: 'web', text: truncateSummary(formatWebTarget(readString(args, 'url'))) };
  }
  if (toolName === 'search_files') {
    return { kind: 'search', text: truncateSummary(readString(args, 'query')) };
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
  return {
    added: readPositiveInt(writeChange, 'added_lines') || readPositiveInt(diffSummary, 'added_lines'),
    removed: readPositiveInt(writeChange, 'removed_lines') || readPositiveInt(diffSummary, 'removed_lines'),
  };
}

export function parseInlineArgs(toolName: string, argsText: string): RecordValue {
  const firstLiteral = readFirstQuotedLiteral(argsText);
  const args: RecordValue = {};
  if (toolName === 'list_files' || toolName === 'read_file' || toolName === 'write_file' || toolName === 'apply_diff') {
    const path = readNamedQuotedArg(argsText, 'path') || firstLiteral;
    if (path) args.path = path;
  }
  if (toolName === 'write_file') {
    const content = readNamedQuotedArg(argsText, 'content');
    if (content) args.content = content;
  }
  if (toolName === 'bash_exec') {
    args.command = readNamedQuotedArg(argsText, 'command') || firstLiteral;
  }
  if (toolName === 'fetch_webpage') {
    args.url = readNamedQuotedArg(argsText, 'url') || firstLiteral;
  }
  if (toolName === 'search_files') {
    args.query = readNamedQuotedArg(argsText, 'query') || firstLiteral;
  }
  return args;
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

function readNamedQuotedArg(argsText: string, name: string): string {
  const pattern = new RegExp(`${name}\\s*=\\s*(['"])(.*?)\\1`, 's');
  const match = argsText.match(pattern);
  return match?.[2]?.trim() || '';
}

function readFirstQuotedLiteral(argsText: string): string {
  const match = argsText.match(/^\s*(['"])(.*?)\1/s);
  return match?.[2]?.trim() || '';
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
