import {
  clearStreamingAssistantState,
  clearStreamingThinkingState,
  clearStreamingToolState,
  STREAMING_ASSISTANT_ORDER_PREFIX,
  STREAMING_THINKING_ORDER_PREFIX,
  type StreamingAssistantState,
  type StreamingThinkingState,
  type StreamingToolTableState,
} from '@/lib/chatStream';
import type { SessionTurnDraft, StreamingToolState } from '@/lib/types';

interface DraftHydratedState {
  streamingAssistantState: StreamingAssistantState;
  streamingThinkingState: StreamingThinkingState;
  streamingItemOrder: string[];
  streamingToolState: StreamingToolTableState;
}

export function buildDraftHydratedState(draft: SessionTurnDraft | null | undefined): DraftHydratedState {
  if (!draft) {
    return {
      streamingAssistantState: clearStreamingAssistantState(),
      streamingThinkingState: clearStreamingThinkingState(),
      streamingItemOrder: [],
      streamingToolState: clearStreamingToolState(),
    };
  }

  return {
    streamingAssistantState: buildSegmentState(
      draft.assistant_segments,
      draft.item_order,
      STREAMING_ASSISTANT_ORDER_PREFIX,
      clearStreamingAssistantState(),
    ),
    streamingThinkingState: buildSegmentState(
      draft.thinking_segments,
      draft.item_order,
      STREAMING_THINKING_ORDER_PREFIX,
      clearStreamingThinkingState(),
    ),
    streamingItemOrder: [...draft.item_order],
    streamingToolState: buildToolState(draft),
  };
}

function buildSegmentState<TSegment extends { id: string; content: string }>(
  segments: TSegment[],
  itemOrder: string[],
  orderPrefix: string,
  initialState: StreamingAssistantState | StreamingThinkingState,
): StreamingAssistantState | StreamingThinkingState {
  const segmentsById = Object.fromEntries(segments.map((segment) => [segment.id, { ...segment }]));
  const activeSegmentId = resolveActiveSegmentId(itemOrder, orderPrefix, segmentsById);
  return {
    ...initialState,
    activeSegmentId,
    nextSegmentSeq: resolveNextSegmentSeq(segments),
    order: segments.map((segment) => segment.id),
    segmentsById,
  };
}

function resolveActiveSegmentId(
  itemOrder: string[],
  orderPrefix: string,
  segmentsById: Record<string, { id: string; content: string }>,
): string {
  const lastItem = itemOrder[itemOrder.length - 1];
  if (!lastItem?.startsWith(orderPrefix)) {
    return '';
  }

  const segmentId = lastItem.slice(orderPrefix.length);
  return segmentsById[segmentId] ? segmentId : '';
}

function resolveNextSegmentSeq(segments: { id: string }[]): number {
  let maxSeq = 0;
  for (const segment of segments) {
    const suffix = Number(segment.id.split(':').at(-1));
    if (Number.isInteger(suffix) && suffix > maxSeq) {
      maxSeq = suffix;
    }
  }
  return maxSeq + 1 || 1;
}

function buildToolState(draft: SessionTurnDraft): StreamingToolTableState {
  const tools: StreamingToolState[] = draft.tools.map((tool) => ({
    id: tool.id,
    content: tool.content,
    toolInput: tool.tool_input,
    toolName: tool.tool_name,
    toolStatus: tool.tool_status,
    toolCallId: tool.tool_call_id,
    traceId: tool.trace_id,
  }));

  return {
    order: tools.map((tool) => tool.id),
    toolsById: Object.fromEntries(tools.map((tool) => [tool.id, tool])),
  };
}
