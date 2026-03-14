import type { ToolChatMessage } from '@/lib/types';

const BYTES_PER_KB = 1024;
const BYTES_PER_MB = BYTES_PER_KB * 1024;
const BYTES_PER_GB = BYTES_PER_MB * 1024;
const TOOL_ACTION_LIMIT = 80;
const TOOL_ACTION_TRUNCATE_AT = 77;

export function formatBytes(bytes?: number): string {
  if (!bytes || bytes <= 0) {
    return '';
  }
  if (bytes < BYTES_PER_KB) {
    return `${bytes} B`;
  }
  if (bytes < BYTES_PER_MB) {
    return `${(bytes / BYTES_PER_KB).toFixed(1)} KB`;
  }
  if (bytes < BYTES_PER_GB) {
    return `${(bytes / BYTES_PER_MB).toFixed(1)} MB`;
  }
  return `${(bytes / BYTES_PER_GB).toFixed(1)} GB`;
}

export function formatToolAction(tool: ToolChatMessage): string {
  if (tool.toolName) {
    return tool.toolName;
  }

  const fallback = tool.content?.split('\n')[0]?.trim();
  if (!fallback) {
    return 'Tool';
  }
  if (fallback.length <= TOOL_ACTION_LIMIT) {
    return fallback;
  }
  return `${fallback.slice(0, TOOL_ACTION_TRUNCATE_AT)}...`;
}

export function formatToolDetails(tool: ToolChatMessage): string {
  const details = buildToolMetadata(tool);
  const outputParts = [tool.content, tool.rawOutput].filter(isNonEmptyText);
  if (outputParts.length > 0) {
    details.push(outputParts.join('\n\n'));
  }
  return details.join('\n');
}

function buildToolMetadata(tool: ToolChatMessage): string[] {
  const details: string[] = [];
  if (tool.toolStatus) {
    details.push(`status: ${tool.toolStatus}`);
  }
  if (tool.traceId) {
    details.push(`trace_id: ${tool.traceId}`);
  }
  if (tool.toolCallId) {
    details.push(`tool_call_id: ${tool.toolCallId}`);
  }
  return details;
}

function isNonEmptyText(value: string | undefined): value is string {
  return Boolean(value && value.trim());
}
