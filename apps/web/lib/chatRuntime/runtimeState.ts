import { createToolTagStreamState, type ToolTagStreamState } from '@/lib/toolTagText';
import type { SessionTurnDraft } from '@/lib/types';

const NO_ACTIVE_THINKING_SEGMENT_INDEX = -1;

export interface ChatRuntimeState {
  assistantBuffer: string;
  assistantMessageId: string;
  assistantRawBuffer: string;
  nextStructuredToolPreviewSeq: number;
  thinkingActiveSegmentIndex: number;
  thinkingBuffers: string[];
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
    assistantRawBuffer: '',
    nextStructuredToolPreviewSeq: 1,
    thinkingActiveSegmentIndex: NO_ACTIVE_THINKING_SEGMENT_INDEX,
    thinkingBuffers: [],
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

export function createChatRuntimeStateFromDraft(
  draft: SessionTurnDraft,
  sessionId?: string,
): ChatRuntimeState {
  const runtime = createChatRuntimeState(draft.trace_id, sessionId);
  runtime.assistantBuffer = draft.assistant_segments.map((segment) => segment.content).join('');
  runtime.assistantRawBuffer = runtime.assistantBuffer;
  runtime.thinkingBuffers = draft.thinking_segments.map((segment) => segment.content);
  runtime.thinkingActiveSegmentIndex = resolveActiveThinkingSegmentIndex(draft);

  for (const tool of draft.tools) {
    if (tool.tool_call_id?.trim()) {
      runtime.toolMessageIds.set(tool.tool_call_id.trim(), tool.id);
    }
    if (tool.tool_name?.trim()) {
      runtime.previewToolNamesByMessageId.set(tool.id, tool.tool_name.trim());
    }
    if (tool.content) {
      runtime.previewToolArgs.set(tool.id, tool.content);
    }

    hydrateStructuredPreviewLookup(runtime, tool.id, tool.tool_call_id);
    hydrateToolTagLookup(runtime, tool.id, tool.tool_name);
  }

  return runtime;
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

function resolveActiveThinkingSegmentIndex(draft: SessionTurnDraft): number {
  const lastItem = draft.item_order[draft.item_order.length - 1];
  if (!lastItem?.startsWith('thinking:')) {
    return NO_ACTIVE_THINKING_SEGMENT_INDEX;
  }

  const activeId = lastItem.slice('thinking:'.length);
  const index = draft.thinking_segments.findIndex((segment) => segment.id === activeId);
  return index >= 0 ? index : NO_ACTIVE_THINKING_SEGMENT_INDEX;
}

function hydrateStructuredPreviewLookup(
  runtime: ChatRuntimeState,
  messageId: string,
  toolCallId?: string,
): void {
  const previewMatch = messageId.match(/:preview:(\d+):index:(\d+)$/);
  if (!previewMatch) {
    return;
  }

  const toolCallIndex = Number(previewMatch[2]);
  if (!Number.isInteger(toolCallIndex)) {
    return;
  }
  runtime.previewStructuredToolMessageIdsByIndex.set(toolCallIndex, messageId);
  if (toolCallId?.trim()) {
    runtime.previewStructuredToolCallIdsByIndex.set(toolCallIndex, toolCallId.trim());
  } else {
    runtime.pendingPreviewQueue.push(messageId);
  }
}

function hydrateToolTagLookup(
  runtime: ChatRuntimeState,
  messageId: string,
  toolName?: string,
): void {
  const tagMatch = messageId.match(/stream-tag-tool:[^:]+:(\d+)$/);
  if (!tagMatch) {
    return;
  }

  const callSeq = Number(tagMatch[1]);
  if (!Number.isInteger(callSeq)) {
    return;
  }
  runtime.previewToolCallSeqToID.set(callSeq, messageId);
  runtime.toolTagState.nextCallSeq = Math.max(runtime.toolTagState.nextCallSeq, callSeq + 1);
  if (toolName?.startsWith('tool#')) {
    runtime.previewToolIDByMessageId.set(messageId, toolName.slice('tool#'.length));
  }
}
