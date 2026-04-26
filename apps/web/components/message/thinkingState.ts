import type { StreamingAssistantSegment, StreamingToolState } from '@/lib/types';

const ACTIVE_TOOL_STATUSES = new Set(['running', 'pending', 'in_progress']);

interface ThinkingStateOptions {
  loading: boolean;
  streamingThinkingText: string;
  streamingAssistantSegments: StreamingAssistantSegment[];
  streamingTools: StreamingToolState[];
}

export function shouldShowThinkingIndicator(options: ThinkingStateOptions): boolean {
  if (!options.loading) {
    return false;
  }
  if (options.streamingThinkingText.trim()) {
    return false;
  }
  if (options.streamingAssistantSegments.length > 0) {
    return false;
  }
  return options.streamingTools.every((tool) => !isToolActive(tool.toolStatus));
}

export function hasAssistantStreamedVisibleText(
  segments: StreamingAssistantSegment[],
): boolean {
  return segments.some((segment) => segment.content.trim().length > 0);
}

export function shouldExpandThinkingPanelByDefault(
  hasThinkingText: boolean,
  hadThinkingText: boolean,
): boolean {
  return hasThinkingText && !hadThinkingText;
}

interface AutoCollapseThinkingPanelOptions {
  hasThinkingText: boolean;
  hasAssistantText: boolean;
  hadAssistantText: boolean;
  alreadyAutoCollapsed: boolean;
}

export function shouldAutoCollapseThinkingPanel(options: AutoCollapseThinkingPanelOptions): boolean {
  if (!options.hasThinkingText) {
    return false;
  }
  if (options.alreadyAutoCollapsed) {
    return false;
  }
  return options.hasAssistantText && !options.hadAssistantText;
}

function isToolActive(status?: string): boolean {
  const normalized = status?.trim().toLowerCase();
  return Boolean(normalized && ACTIVE_TOOL_STATUSES.has(normalized));
}
