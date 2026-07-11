import type { ToolChatMessage } from '@/lib/types';
import { buildToolDetailCompact } from './compact';
import {
  buildDirectAction,
  isErrorStatus,
  normalizeInline,
  normalizeToolName,
  parseRecord,
  readString,
  supportsPromotedActionTitle,
  type ActionKind,
  truncateError,
  type RecordValue,
} from './common';
import { buildToolSearchAction } from './sfind';
import { buildToolDetailError, buildToolDetailSummary } from './summary';
import { resolveToolCallArgs } from './toolCalls';

const TOOL_ACTION_LIMIT = 80;
const TOOL_ACTION_TRUNCATE_AT = 77;

interface ToolDetailFormatOptions {
  compactOutputEnabled?: boolean;
}

export interface FormattedToolAction {
  actionKind?: ActionKind;
  actionText?: string;
  text: string;
  variant: 'action' | 'command' | 'default';
}

export { formatBytes } from '@/lib/formatBytes';

export function formatToolAction(tool: ToolChatMessage): FormattedToolAction {
  const bashCommand = resolveBashExecCommand(tool);
  if (bashCommand) {
    return { text: bashCommand, variant: 'command' };
  }
  const promotedAction = resolvePromotedToolActionText(tool);
  if (promotedAction) {
    return {
      actionKind: promotedAction.kind,
      actionText: promotedAction.text,
      text: formatActionTitle(promotedAction),
      variant: 'action',
    };
  }
  if (tool.toolName) {
    return { text: tool.toolName, variant: 'default' };
  }

  const fallback = tool.content?.split('\n')[0]?.trim();
  if (!fallback) {
    return { text: 'Tool', variant: 'default' };
  }
  if (fallback.length <= TOOL_ACTION_LIMIT) {
    return { text: fallback, variant: 'default' };
  }
  return {
    text: `${fallback.slice(0, TOOL_ACTION_TRUNCATE_AT)}...`,
    variant: 'default',
  };
}

export function formatToolCardDetails(
  tool: ToolChatMessage,
  options: ToolDetailFormatOptions = {},
): string {
  if (!shouldPromoteToolActionToTitle(tool)) {
    return formatToolDetails(tool, options);
  }
  if (options.compactOutputEnabled) {
    return buildPromotedCompactDetails(tool);
  }
  return buildPromotedExpandedDetails(tool);
}

export function formatToolDetails(
  tool: ToolChatMessage,
  options: ToolDetailFormatOptions = {},
): string {
  if (options.compactOutputEnabled) {
    return buildToolDetailCompact(tool);
  }

  return buildExpandedToolDetails(tool);
}

function buildExpandedToolDetails(tool: ToolChatMessage): string {
  const details: string[] = [];
  const errorLine = buildToolDetailError(tool);
  if (errorLine) {
    details.push(`error: ${errorLine}`);
  }

  const summaryLine = buildToolDetailSummary(tool);
  if (summaryLine) {
    details.push(`summary: ${summaryLine}`);
  }

  const outputText = buildOutputText(tool);
  if (outputText) {
    details.push(outputText);
  }
  return details.join('\n');
}

function buildPromotedExpandedDetails(tool: ToolChatMessage): string {
  const details: string[] = [];
  const errorLine = buildToolDetailError(tool);
  if (errorLine) {
    details.push(`error: ${errorLine}`);
  }
  const outputText = buildOutputText(tool);
  if (outputText) {
    details.push(outputText);
  }
  return details.join('\n');
}

function buildPromotedCompactDetails(tool: ToolChatMessage): string {
  const details = [isErrorStatus(tool.toolStatus) ? 'failed' : 'success'];
  if (isErrorStatus(tool.toolStatus)) {
    const errorLine = buildToolDetailError(tool);
    if (errorLine) {
      details.push(errorLine);
    }
    return details.join('\n');
  }
  details.push(...buildCompactOutputPreview(tool));
  return details.join('\n');
}

function buildOutputText(tool: ToolChatMessage): string {
  const outputParts = collectUniqueOutputParts([tool.content, tool.rawOutput]);
  return outputParts.join('\n\n');
}

function collectUniqueOutputParts(values: Array<string | undefined>): string[] {
  const outputParts: string[] = [];
  for (const value of values) {
    if (!isNonEmptyText(value)) {
      continue;
    }
    if (outputParts.some((current) => current.trim() === value.trim())) {
      continue;
    }
    outputParts.push(value);
  }
  return outputParts;
}

function isNonEmptyText(value: string | undefined): value is string {
  return Boolean(value && value.trim());
}

function buildCompactOutputPreview(tool: ToolChatMessage): string[] {
  const previewSource = collectUniqueOutputParts([tool.content, tool.rawOutput])
    .find((value) => !looksStructuredOutput(value));
  if (!previewSource) {
    return [];
  }

  const previewLines: string[] = [];
  for (const line of previewSource.split('\n')) {
    const normalized = normalizeInline(line);
    if (!normalized || previewLines.includes(normalized)) {
      continue;
    }
    previewLines.push(truncateError(normalized));
    if (previewLines.length >= 2) {
      break;
    }
  }
  return previewLines;
}

function resolveBashExecCommand(tool: ToolChatMessage): string {
  const normalizedToolName = normalizeToolName(tool.toolName);
  if (normalizedToolName !== 'bash_exec' && normalizedToolName !== 'codex_exec') {
    return '';
  }

  const directCommand = readBashExecCommand(tool.toolInput)
    || readBashExecCommand(tool.content)
    || readBashExecCommand(tool.rawOutput);
  if (directCommand) {
    return directCommand;
  }
  return readBashExecCommandArgs(resolveToolCallArgs(tool));
}

function readBashExecCommand(rawText?: string): string {
  const parsedCommand = readBashExecCommandArgs(parseRecord(rawText));
  if (parsedCommand) {
    return parsedCommand;
  }
  const matched = rawText?.match(/"(?:command|cmd)"\s*:\s*"([^"]*)/s);
  return normalizeInline(matched?.[1]);
}

function readBashExecCommandArgs(args?: RecordValue): string {
  return normalizeInline(readString(args, 'command') || readString(args, 'cmd'));
}

function resolvePromotedToolActionText(tool: ToolChatMessage) {
  const action = resolvePromotedToolAction(tool);
  if (!action) {
    return undefined;
  }
  return action;
}

function resolvePromotedToolAction(tool: ToolChatMessage) {
  const sfindAction = buildToolSearchAction(tool);
  if (sfindAction) {
    return sfindAction;
  }

  const toolName = normalizeToolName(tool.toolName);
  if (!supportsPromotedActionTitle(toolName) || toolName === 'bash_exec') {
    return undefined;
  }

  const args = parseRecord(tool.toolInput)
    || parseRecord(tool.content)
    || parseRecord(tool.rawOutput)
    || resolveToolCallArgs(tool);
  return buildDirectAction(toolName, args);
}

function shouldPromoteToolActionToTitle(tool: ToolChatMessage): boolean {
  return formatToolAction(tool).variant !== 'default';
}

function formatActionTitle(action: { kind: ActionKind; text: string }): string {
  return action.text ? `${action.kind} ${action.text}` : action.kind;
}

function looksStructuredOutput(value: string): boolean {
  const trimmed = value.trim();
  return (trimmed.startsWith('{') && trimmed.endsWith('}'))
    || (trimmed.startsWith('[') && trimmed.endsWith(']'));
}
