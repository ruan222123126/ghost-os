import type {
  AgentCompletionDeltaPayload,
  AgentStreamEvent,
  AgentToolCallFinishedPayload,
  AgentToolCallStartedPayload,
} from '@/lib/types';
import { supportsPromotedActionTitle } from '@/lib/toolNames';
import type { ChatRuntimeAction } from './actions';
import {
  TOOL_ERROR_STATUS,
  TOOL_PENDING_STATUS,
  TOOL_RUNNING_STATUS,
  TOOL_SUCCESS_STATUS,
} from './constants';
import { markRuntimeThinkingBoundary, type ChatRuntimeState } from './runtimeState';

export function projectToolStartedEvent(
  runtime: ChatRuntimeState,
  event: AgentStreamEvent,
  payload: AgentToolCallStartedPayload,
): ChatRuntimeAction[] {
  clearStructuredPreviewLookups(runtime);
  const messageId = resolveToolMessageID(runtime, event, payload.tool_call_id);
  const argsPreview = payload.arguments_json?.trim() || runtime.previewToolArgs.get(messageId) || '';
  if (argsPreview) {
    runtime.previewToolArgs.set(messageId, argsPreview);
  }
  const toolName = payload.tool || resolvePreviewToolName(runtime, messageId);
  const toolInput = resolveStreamingToolInput(toolName, argsPreview);

  markRuntimeThinkingBoundary(runtime);
  return [
    { type: 'mark_streaming_thinking_boundary' },
    {
      type: 'upsert_streaming_tool',
      tool: {
        id: messageId,
        content: argsPreview || `${toolName || 'Tool'} running`,
        ...(toolInput ? { toolInput } : {}),
        toolCallId: payload.tool_call_id,
        toolName,
        toolStatus: TOOL_RUNNING_STATUS,
        traceId: event.trace_id,
      },
    },
  ];
}

export function projectToolFinishedEvent(
  runtime: ChatRuntimeState,
  event: AgentStreamEvent,
  payload: AgentToolCallFinishedPayload,
): ChatRuntimeAction[] {
  clearStructuredPreviewLookups(runtime);
  const messageId = resolveToolMessageID(runtime, event, payload.tool_call_id);
  const toolName = payload.tool || resolvePreviewToolName(runtime, messageId);
  const argsPreview = runtime.previewToolArgs.get(messageId) || '';
  const outputPreview = payload.output?.trim() || argsPreview;
  const toolStatus = payload.status?.trim() || (payload.error ? TOOL_ERROR_STATUS : TOOL_SUCCESS_STATUS);
  const toolInput = resolveStreamingToolInput(toolName, argsPreview);

  markRuntimeThinkingBoundary(runtime);
  return [
    { type: 'mark_streaming_thinking_boundary' },
    {
      type: 'upsert_streaming_tool',
      tool: {
        id: messageId,
        content: payload.error?.trim() || outputPreview || `${toolName || 'Tool'} finished`,
        ...(toolInput ? { toolInput } : {}),
        toolCallId: payload.tool_call_id,
        toolName,
        toolStatus,
        traceId: event.trace_id,
      },
    },
  ];
}

export function projectToolCallStartDelta(
  runtime: ChatRuntimeState,
  traceId: string,
  payload: AgentCompletionDeltaPayload,
): ChatRuntimeAction[] {
  const messageId = resolvePreviewToolMessageID(runtime, payload.tool_call_index, payload.tool_call_id);
  if (!messageId) {
    return [];
  }

  const toolName = payload.tool_name?.trim();
  if (toolName) {
    runtime.previewToolNamesByMessageId.set(messageId, toolName);
  }

  markRuntimeThinkingBoundary(runtime);
  return [
    { type: 'mark_streaming_thinking_boundary' },
    buildPreviewToolAction(runtime, messageId, traceId, TOOL_PENDING_STATUS),
  ];
}

export function projectToolCallDelta(
  runtime: ChatRuntimeState,
  traceId: string,
  payload: AgentCompletionDeltaPayload,
): ChatRuntimeAction[] {
  const messageId = resolvePreviewToolMessageID(runtime, payload.tool_call_index, payload.tool_call_id);
  if (!messageId || !payload.arguments_fragment) {
    return [];
  }

  const nextArgs = `${runtime.previewToolArgs.get(messageId) || ''}${payload.arguments_fragment}`;
  runtime.previewToolArgs.set(messageId, nextArgs);
  return [buildPreviewToolAction(runtime, messageId, traceId, TOOL_PENDING_STATUS)];
}

export function projectToolCallEndDelta(
  runtime: ChatRuntimeState,
  traceId: string,
  payload: AgentCompletionDeltaPayload,
): ChatRuntimeAction[] {
  const messageId = resolvePreviewToolMessageID(runtime, payload.tool_call_index, payload.tool_call_id);
  if (!messageId) {
    return [];
  }
  if (payload.tool_call_index !== undefined && !resolveStructuredToolCallID(runtime, payload.tool_call_index)) {
    enqueuePendingPreviewMessage(runtime, messageId);
  }
  return [buildPreviewToolAction(runtime, messageId, traceId, TOOL_PENDING_STATUS)];
}

function resolveToolMessageID(runtime: ChatRuntimeState, event: AgentStreamEvent, toolCallId?: string): string {
  const trimmedToolCallID = toolCallId?.trim();
  if (trimmedToolCallID) {
    return resolveToolMessageIDWithCallID(runtime, trimmedToolCallID);
  }

  return resolveToolMessageIDWithFallback(runtime, event.step_id.trim() || event.id.trim());
}

function resolveToolMessageIDWithCallID(runtime: ChatRuntimeState, toolCallId: string): string {
  const existing = runtime.toolMessageIds.get(toolCallId);
  if (existing) {
    return existing;
  }
  if (runtime.pendingPreviewQueue.length > 0) {
    const previewMessageID = runtime.pendingPreviewQueue.shift();
    if (previewMessageID) {
      runtime.toolMessageIds.set(toolCallId, previewMessageID);
      return previewMessageID;
    }
  }

  const generated = `stream-tool:${runtime.traceId}:${toolCallId}`;
  runtime.toolMessageIds.set(toolCallId, generated);
  return generated;
}

function resolveToolMessageIDWithFallback(runtime: ChatRuntimeState, key: string): string {
  const existing = runtime.toolMessageIds.get(key);
  if (existing) {
    return existing;
  }

  const generated = `stream-tool:${runtime.traceId}:${key}`;
  runtime.toolMessageIds.set(key, generated);
  return generated;
}

function resolvePreviewToolMessageID(
  runtime: ChatRuntimeState,
  toolCallIndex?: number,
  toolCallId?: string,
): string {
  if (toolCallIndex === undefined) {
    return ensureToolMessageIDWithCallID(runtime, toolCallId);
  }

  const existing = runtime.previewStructuredToolMessageIdsByIndex.get(toolCallIndex);
  if (existing) {
    rememberStructuredToolCallID(runtime, toolCallIndex, toolCallId, existing);
    return existing;
  }

  const generated = ensureToolMessageIDWithCallID(runtime, toolCallId)
    || createStructuredPreviewMessageID(runtime, toolCallIndex);
  runtime.previewStructuredToolMessageIdsByIndex.set(toolCallIndex, generated);
  rememberStructuredToolCallID(runtime, toolCallIndex, toolCallId, generated);
  return generated;
}

function ensureToolMessageIDWithCallID(runtime: ChatRuntimeState, toolCallId?: string): string {
  const trimmedToolCallID = toolCallId?.trim();
  if (!trimmedToolCallID) {
    return '';
  }

  const existing = runtime.toolMessageIds.get(trimmedToolCallID);
  if (existing) {
    return existing;
  }

  const generated = `stream-tool:${runtime.traceId}:${trimmedToolCallID}`;
  runtime.toolMessageIds.set(trimmedToolCallID, generated);
  return generated;
}

function rememberStructuredToolCallID(
  runtime: ChatRuntimeState,
  toolCallIndex: number,
  toolCallId: string | undefined,
  messageId: string,
): void {
  const trimmedToolCallID = toolCallId?.trim();
  if (!trimmedToolCallID) {
    return;
  }

  runtime.toolMessageIds.set(trimmedToolCallID, messageId);
  runtime.previewStructuredToolCallIdsByIndex.set(toolCallIndex, trimmedToolCallID);
}

function resolveStructuredToolCallID(runtime: ChatRuntimeState, toolCallIndex: number): string | undefined {
  return runtime.previewStructuredToolCallIdsByIndex.get(toolCallIndex)?.trim() || undefined;
}

function enqueuePendingPreviewMessage(runtime: ChatRuntimeState, messageId: string): void {
  if (!runtime.pendingPreviewQueue.includes(messageId)) {
    runtime.pendingPreviewQueue.push(messageId);
  }
}

function clearStructuredPreviewLookups(runtime: ChatRuntimeState): void {
  runtime.previewStructuredToolCallIdsByIndex.clear();
  runtime.previewStructuredToolMessageIdsByIndex.clear();
}

function createStructuredPreviewMessageID(runtime: ChatRuntimeState, toolCallIndex: number): string {
  const previewSeq = runtime.nextStructuredToolPreviewSeq;
  runtime.nextStructuredToolPreviewSeq += 1;
  return `stream-tool:${runtime.traceId}:preview:${previewSeq}:index:${toolCallIndex}`;
}

function buildPreviewToolAction(
  runtime: ChatRuntimeState,
  messageId: string,
  traceId: string,
  toolStatus: string,
): ChatRuntimeAction {
  const toolName = resolvePreviewToolName(runtime, messageId);
  const toolInput = resolveStreamingToolInput(toolName, runtime.previewToolArgs.get(messageId) || '');
  return {
    type: 'upsert_streaming_tool',
    tool: {
      id: messageId,
      content: runtime.previewToolArgs.get(messageId) || '',
      ...(toolInput ? { toolInput } : {}),
      toolCallId: resolvePreviewToolCallID(runtime, messageId),
      toolName,
      toolStatus,
      traceId,
    },
  };
}

function resolvePreviewToolCallID(runtime: ChatRuntimeState, messageId: string): string | undefined {
  for (const [toolCallId, currentMessageID] of runtime.toolMessageIds.entries()) {
    if (currentMessageID === messageId) {
      return toolCallId;
    }
  }
  return undefined;
}

function resolvePreviewToolName(runtime: ChatRuntimeState, messageId: string): string | undefined {
  const previewToolName = runtime.previewToolNamesByMessageId.get(messageId)?.trim();
  if (previewToolName) {
    return previewToolName;
  }
  return formatToolIDName(runtime.previewToolIDByMessageId.get(messageId));
}

function formatToolIDName(toolId?: string): string | undefined {
  const trimmed = toolId?.trim();
  if (!trimmed) {
    return undefined;
  }
  return `tool#${trimmed}`;
}

function resolveStreamingToolInput(toolName: string | undefined, inputText: string): string {
  return inputText && supportsPromotedActionTitle(toolName) ? inputText : '';
}
