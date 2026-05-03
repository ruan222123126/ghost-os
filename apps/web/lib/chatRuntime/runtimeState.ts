import {
  parseAgentDonePayload,
  parseAgentErrorPayload,
  parseAgentRunStartedPayload,
  parseAgentStreamMessagePayload,
} from '@/lib/api/agent/parser';
import { createToolTagStreamState, type ToolTagStreamState } from '@/lib/toolTagText';
import type { AgentStreamEvent } from '@/lib/types';

const NO_ACTIVE_THINKING_SEGMENT_INDEX = -1;

interface RuntimeThinkingSegment {
  id: string;
  content: string;
}

export interface ChatRuntimeState {
  assistantBuffer: string;
  assistantMessageId: string;
  nextStructuredToolPreviewSeq: number;
  thinkingActiveSegmentIndex: number;
  thinkingBuffers: string[];
  thinkingMessageId: string;
  pendingPreviewQueue: string[];
  previewToolArgs: Map<string, string>;
  previewToolCallSeqToID: Map<number, string>;
  previewToolIDByMessageId: Map<string, string>;
  previewToolNamesByMessageId: Map<string, string>;
  previewStructuredToolCallIdsByIndex: Map<number, string>;
  previewStructuredToolMessageIdsByIndex: Map<number, string>;
  sessionId: string;
  toolMessageIds: Map<string, string>;
  toolTagState: ToolTagStreamState;
  traceId: string;
}

export function createChatRuntimeState(traceId: string, sessionId?: string): ChatRuntimeState {
  const trimmedTraceId = traceId.trim();
  return {
    assistantBuffer: '',
    assistantMessageId: `stream-assistant:${trimmedTraceId}`,
    nextStructuredToolPreviewSeq: 1,
    thinkingActiveSegmentIndex: NO_ACTIVE_THINKING_SEGMENT_INDEX,
    thinkingBuffers: [],
    thinkingMessageId: `stream-thinking:${trimmedTraceId}`,
    pendingPreviewQueue: [],
    previewToolArgs: new Map(),
    previewToolCallSeqToID: new Map(),
    previewToolIDByMessageId: new Map(),
    previewToolNamesByMessageId: new Map(),
    previewStructuredToolCallIdsByIndex: new Map(),
    previewStructuredToolMessageIdsByIndex: new Map(),
    sessionId: sessionId?.trim() || '',
    toolMessageIds: new Map(),
    toolTagState: createToolTagStreamState(),
    traceId: trimmedTraceId,
  };
}

export function appendRuntimeThinkingDelta(runtime: ChatRuntimeState, delta: string): void {
  if (!delta) {
    return;
  }
  if (runtime.thinkingActiveSegmentIndex === NO_ACTIVE_THINKING_SEGMENT_INDEX) {
    runtime.thinkingBuffers.push(delta);
    runtime.thinkingActiveSegmentIndex = runtime.thinkingBuffers.length - 1;
    return;
  }

  const activeSegment = runtime.thinkingBuffers[runtime.thinkingActiveSegmentIndex] || '';
  runtime.thinkingBuffers[runtime.thinkingActiveSegmentIndex] = `${activeSegment}${delta}`;
}

export function markRuntimeThinkingBoundary(runtime: ChatRuntimeState): void {
  runtime.thinkingActiveSegmentIndex = NO_ACTIVE_THINKING_SEGMENT_INDEX;
}

export function clearRuntimeThinking(runtime: ChatRuntimeState): void {
  runtime.thinkingBuffers = [];
  runtime.thinkingActiveSegmentIndex = NO_ACTIVE_THINKING_SEGMENT_INDEX;
}

export function drainRuntimeThinkingSegments(runtime: ChatRuntimeState): RuntimeThinkingSegment[] {
  const committed: RuntimeThinkingSegment[] = runtime.thinkingBuffers
    .map((content, index) => ({
      id: buildThinkingSegmentId(runtime.thinkingMessageId, index),
      content: content.trim(),
    }))
    .filter((segment) => Boolean(segment.content));
  clearRuntimeThinking(runtime);
  return committed;
}

export function resolveEventSessionId(event: AgentStreamEvent): string {
  if (event.session_id?.trim()) {
    return event.session_id.trim();
  }

  switch (event.type) {
    case 'run_started':
      return parseAgentRunStartedPayload(event.payload).session_id?.trim() || '';
    case 'message':
      return parseAgentStreamMessagePayload(event.payload).session_id?.trim() || '';
    case 'done':
      return parseAgentDonePayload(event.payload).session_id?.trim() || '';
    case 'error':
      return parseAgentErrorPayload(event.payload).session_id?.trim() || '';
    default:
      return '';
  }
}

function buildThinkingSegmentId(baseId: string, index: number): string {
  if (index === 0) {
    return baseId;
  }
  return `${baseId}:${index + 1}`;
}
