import type { ToolChatMessage } from '@/lib/types';
import { normalizeToolName, type RecordValue } from './toolDetailCommon';

interface ParsedToolCall {
  id: string;
  name: string;
  arguments: RecordValue;
}

export function resolveToolCallArgs(tool: ToolChatMessage): RecordValue | undefined {
  const parsedToolCalls = parseToolCalls(tool.toolCalls);
  const matchedByID = findToolCallByID(parsedToolCalls, tool.toolCallId);
  if (matchedByID) {
    return matchedByID.arguments;
  }
  const matchedByName = findToolCallByName(parsedToolCalls, tool.toolName);
  return matchedByName?.arguments;
}

function parseToolCalls(toolCalls?: unknown[]): ParsedToolCall[] {
  if (!Array.isArray(toolCalls) || toolCalls.length === 0) {
    return [];
  }

  const parsed: ParsedToolCall[] = [];
  for (const toolCall of toolCalls) {
    const record = readRecordValue(toolCall);
    const argumentsValue = readRecordValue(record?.arguments);
    if (!record || !argumentsValue) {
      continue;
    }
    parsed.push({
      id: readStringValue(record.id),
      name: readStringValue(record.name),
      arguments: argumentsValue,
    });
  }
  return parsed;
}

function findToolCallByID(toolCalls: ParsedToolCall[], toolCallId?: string): ParsedToolCall | undefined {
  const normalizedID = readStringValue(toolCallId);
  if (!normalizedID) {
    return undefined;
  }
  return toolCalls.find((toolCall) => toolCall.id === normalizedID);
}

function findToolCallByName(toolCalls: ParsedToolCall[], toolName?: string): ParsedToolCall | undefined {
  const normalizedName = normalizeToolName(toolName);
  if (!normalizedName) {
    return undefined;
  }
  return toolCalls.find((toolCall) => normalizeToolName(toolCall.name) === normalizedName);
}

function readRecordValue(value: unknown): RecordValue | undefined {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) {
    return undefined;
  }
  return value as RecordValue;
}

function readStringValue(value: unknown): string {
  return typeof value === 'string' ? value.trim() : '';
}
