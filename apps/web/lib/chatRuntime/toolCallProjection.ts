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
import {
  clearStructuredPreviewLookups,
  enqueuePendingPreviewMessage,
  resolvePreviewToolCallID,
  resolvePreviewToolMessageID,
  resolvePreviewToolName,
  resolveStructuredToolCallID,
  resolveToolMessageID,
} from './toolPreviewState';

interface PreviewToolActionInput {
  messageId: string;
  runtime: ChatRuntimeState;
  toolStatus: string;
  traceId: string;
}

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
  const outputPreview = trimmedText(payload.output) || argsPreview;
  const toolStatus = resolveFinishedToolStatus(payload);
  const toolInput = resolveStreamingToolInput(toolName, argsPreview);

  markRuntimeThinkingBoundary(runtime);
  return [
    { type: 'mark_streaming_thinking_boundary' },
    {
      type: 'upsert_streaming_tool',
      tool: {
        id: messageId,
        content: resolveFinishedToolContent(payload, outputPreview, toolName),
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
    buildPreviewToolAction({
      runtime,
      messageId,
      traceId,
      toolStatus: TOOL_PENDING_STATUS,
    }),
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
  return [buildPreviewToolAction({
    runtime,
    messageId,
    traceId,
    toolStatus: TOOL_PENDING_STATUS,
  })];
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
  return [buildPreviewToolAction({
    runtime,
    messageId,
    traceId,
    toolStatus: TOOL_PENDING_STATUS,
  })];
}

function buildPreviewToolAction(input: PreviewToolActionInput): ChatRuntimeAction {
  const { messageId, runtime, toolStatus, traceId } = input;
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

function resolveStreamingToolInput(toolName: string | undefined, inputText: string): string {
  return inputText && supportsPromotedActionTitle(toolName) ? inputText : '';
}

function resolveFinishedToolStatus(payload: AgentToolCallFinishedPayload): string {
  const status = trimmedText(payload.status);
  if (status) {
    return status;
  }
  return payload.error ? TOOL_ERROR_STATUS : TOOL_SUCCESS_STATUS;
}

function resolveFinishedToolContent(
  payload: AgentToolCallFinishedPayload,
  outputPreview: string,
  toolName: string | undefined,
): string {
  const errorText = trimmedText(payload.error);
  if (errorText) {
    return errorText;
  }
  if (outputPreview) {
    return outputPreview;
  }
  return `${toolName || 'Tool'} finished`;
}

function trimmedText(value?: string): string {
  return value?.trim() ?? '';
}
