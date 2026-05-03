import type { ToolChatMessage } from '@/lib/types';
import {
  ACTION_ORDER,
  type ActionItem,
  type ActionKind,
  buildDirectAction,
  buildWriteChangeAction,
  isErrorStatus,
  normalizeInline,
  normalizeToolName,
  parseRecord,
  parseScriptExecReport,
  parseScriptExecScriptActions,
  readRecord,
  readScriptExecReportError,
  readString,
  truncateError,
  truncateSummary,
} from './toolDetailCommon';
import { buildToolSearchAction } from './toolDetailSfind';
import { resolveToolCallArgs } from './toolDetailToolCalls';

const ACTION_SAMPLE_LIMIT = 2;

export function buildToolDetailSummary(tool: ToolChatMessage): string {
  const actions = collectSummaryActions(tool);
  if (actions.length === 0) {
    return '';
  }
  return formatActionGroups(actions);
}

export function buildToolDetailError(tool: ToolChatMessage): string {
  if (!isErrorStatus(tool.toolStatus)) {
    return '';
  }

  const reportError = readScriptExecReportError(tool.rawOutput) || readScriptExecReportError(tool.content);
  if (reportError) {
    return truncateError(reportError);
  }

  const fallback = normalizeInline(tool.rawOutput) || normalizeInline(tool.content);
  return fallback ? truncateError(fallback) : 'unknown error';
}

function collectSummaryActions(tool: ToolChatMessage): ActionItem[] {
  const outputReportActions = collectScriptExecReportActions(tool.rawOutput);
  if (outputReportActions.length > 0) {
    return outputReportActions;
  }

  const contentReportActions = collectScriptExecReportActions(tool.content);
  if (contentReportActions.length > 0) {
    return contentReportActions;
  }

  const contentScriptActions = collectScriptArgActions(tool.content);
  if (contentScriptActions.length > 0) {
    return contentScriptActions;
  }

  const rawScriptActions = collectScriptArgActions(tool.rawOutput);
  if (rawScriptActions.length > 0) {
    return rawScriptActions;
  }

  const toolSearchAction = buildToolSearchAction(tool);
  if (toolSearchAction) {
    return [toolSearchAction];
  }

  const toolName = normalizeToolName(tool.toolName);
  if (!toolName) {
    return [];
  }
  const actionArgs = parseRecord(tool.toolInput)
    || parseRecord(tool.content)
    || parseRecord(tool.rawOutput)
    || resolveToolCallArgs(tool);
  const action = buildDirectAction(toolName, actionArgs);
  return action ? [action] : [];
}

function collectScriptExecReportActions(rawText?: string): ActionItem[] {
  const report = parseScriptExecReport(rawText);
  if (!report) {
    return [];
  }

  const actions: ActionItem[] = [];
  for (const step of report.steps) {
    const action = buildScriptExecStepAction(step);
    if (action) {
      actions.push(action);
    }
  }
  return actions;
}

function buildScriptExecStepAction(step: Record<string, unknown>): ActionItem | undefined {
  const args = readRecord(step, 'args');
  const writeChange = readRecord(step, 'write_change');
  const writeAction = buildWriteChangeAction(writeChange, args);
  if (writeAction) {
    return writeAction;
  }
  const toolName = normalizeToolName(readString(step, 'tool'));
  return buildDirectAction(toolName, args);
}

function collectScriptArgActions(rawText?: string): ActionItem[] {
  return parseScriptExecScriptActions(rawText);
}

function formatActionGroups(actions: ActionItem[]): string {
  const grouped = new Map<ActionKind, string[]>();
  for (const action of actions) {
    const items = grouped.get(action.kind) || [];
    items.push(action.text);
    grouped.set(action.kind, items);
  }

  const sections: string[] = [];
  for (const kind of ACTION_ORDER) {
    const items = grouped.get(kind);
    if (!items || items.length === 0) {
      continue;
    }
    sections.push(formatActionSection(kind, items));
  }
  return sections.join(' | ');
}

function formatActionSection(kind: ActionKind, items: string[]): string {
  const visibleItems = items.filter(Boolean);
  const sample = visibleItems.slice(0, ACTION_SAMPLE_LIMIT).map((item) => truncateSummary(item));
  const hidden = visibleItems.length - sample.length;
  let section: string = kind;
  if (sample.length > 0) {
    section = `${section} ${sample.join(', ')}`;
  }
  if (hidden > 0) {
    section = `${section} +${hidden}`;
  }
  return section;
}
