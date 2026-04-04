import type { StreamingAssistantSegment, StreamingToolState } from '@/lib/types';

const ACTIVE_TOOL_STATUSES = new Set(['running', 'pending', 'in_progress']);

interface ThinkingStateOptions {
  loading: boolean;
  streamingAssistantSegments: StreamingAssistantSegment[];
  streamingTools: StreamingToolState[];
}

export function shouldShowThinkingIndicator(options: ThinkingStateOptions): boolean {
  if (!options.loading) {
    return false;
  }
  if (options.streamingAssistantSegments.length > 0) {
    return false;
  }
  return options.streamingTools.every((tool) => !isToolActive(tool.toolStatus));
}

function isToolActive(status?: string): boolean {
  const normalized = status?.trim().toLowerCase();
  return Boolean(normalized && ACTIVE_TOOL_STATUSES.has(normalized));
}
