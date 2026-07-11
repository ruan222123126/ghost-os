import type { ToolChatMessage } from '@/lib/types';
import {
  type ActionItem,
  buildDirectAction,
  buildWriteChangeAction,
  normalizeInline,
  normalizeToolName,
  parseRecord,
  parseScriptExecReport,
  parseScriptExecScriptActions,
  readRecord,
  readString,
  truncateError,
} from './common';
import { buildOrchestrationDispatchCompact } from './orchestrationDispatch';
import { buildToolSearchAction } from './sfind';
import { resolveToolCallArgs } from './toolCalls';
import { buildToolDetailError } from './summary';

export function buildToolDetailCompact(tool: ToolChatMessage): string {
  const dispatchDetails = buildOrchestrationDispatchCompact(tool);
  if (dispatchDetails) {
    return dispatchDetails;
  }
  const normalizedToolName = normalizeToolName(tool.toolName);
  if (normalizedToolName === 'bash_exec' || normalizedToolName === 'codex_exec') {
    return buildBashExecCompact(tool);
  }
  const details: string[] = [];
  if (isErrorStatus(tool.toolStatus)) {
    details.push(`error: ${buildToolDetailError(tool)}`);
  }
  const steps = collectCompactSteps(tool);
  for (let index = 0; index < steps.length; index += 1) {
    details.push(`step${index + 1}: ${steps[index]}`);
  }
  return details.join('\n');
}

function collectCompactSteps(tool: ToolChatMessage): string[] {
  const reportSteps = collectScriptExecReportSteps(tool.rawOutput);
  if (reportSteps.length > 0) {
    return reportSteps;
  }

  const reportStepsInContent = collectScriptExecReportSteps(tool.content);
  if (reportStepsInContent.length > 0) {
    return reportStepsInContent;
  }

  const scriptSteps = collectScriptSteps(tool.content);
  if (scriptSteps.length > 0) {
    return scriptSteps;
  }

  const scriptStepsInOutput = collectScriptSteps(tool.rawOutput);
  if (scriptStepsInOutput.length > 0) {
    return scriptStepsInOutput;
  }

  const directStep = buildDirectToolStep(tool);
  if (directStep) {
    return [directStep];
  }
  return ['run'];
}

function collectScriptExecReportSteps(rawText?: string): string[] {
  const report = parseScriptExecReport(rawText);
  if (!report) {
    return [];
  }

  const steps: string[] = [];
  for (const step of report.steps) {
    const action = buildReportStepAction(step);
    if (action) {
      steps.push(formatActionItem(action));
    }
  }
  return steps;
}

function buildReportStepAction(step: Record<string, unknown>): ActionItem | undefined {
  const args = readRecord(step, 'args');
  const writeChange = readRecord(step, 'write_change');
  const writeAction = buildWriteChangeAction(writeChange, args);
  if (writeAction) {
    return writeAction;
  }
  const toolName = normalizeToolName(readString(step, 'tool'));
  return buildDirectAction(toolName, args);
}

function collectScriptSteps(rawText?: string): string[] {
  return parseScriptExecScriptActions(rawText).map(formatActionItem);
}

function buildDirectToolStep(tool: ToolChatMessage): string {
  const toolSearchAction = buildToolSearchAction(tool);
  if (toolSearchAction) {
    return formatActionItem(toolSearchAction);
  }

  const toolName = normalizeToolName(tool.toolName);
  if (!toolName) {
    return '';
  }
  const args = parseRecord(tool.toolInput)
    || parseRecord(tool.content)
    || parseRecord(tool.rawOutput)
    || resolveToolCallArgs(tool);
  const action = buildDirectAction(toolName, args);
  return action ? formatActionItem(action) : '';
}

function formatActionItem(action: ActionItem): string {
  return action.text ? `${action.kind} ${action.text}` : action.kind;
}

function buildBashExecCompact(tool: ToolChatMessage): string {
  const details = [isErrorStatus(tool.toolStatus) ? 'failed' : 'success'];
  details.push(...buildBashExecOutputPreview(tool));
  return details.join('\n');
}

function buildBashExecOutputPreview(tool: ToolChatMessage): string[] {
  const lines = collectBashExecOutputLines(tool);
  if (lines.length > 0) {
    return lines;
  }
  const errorText = buildToolDetailError(tool);
  return errorText ? [truncateError(errorText)] : [];
}

function collectBashExecOutputLines(tool: ToolChatMessage): string[] {
  const outputText = tool.rawOutput?.trim() || tool.content?.trim() || '';
  if (!outputText) {
    return [];
  }

  const previewLines: string[] = [];
  for (const line of outputText.split('\n')) {
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

function isErrorStatus(status?: string): boolean {
  const normalized = status?.trim().toLowerCase() || '';
  return normalized === 'error' || normalized === 'failed';
}
