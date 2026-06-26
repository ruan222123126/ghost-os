import type { AgentStreamEvent } from '@/lib/types';
import type { ChatRuntimeState } from './runtimeState';

interface StructuredToolCallLookupInput {
  messageId: string;
  runtime: ChatRuntimeState;
  toolCallId?: string;
  toolCallIndex: number;
}

export function resolveToolMessageID(
  runtime: ChatRuntimeState,
  event: AgentStreamEvent,
  toolCallId?: string,
): string {
  const trimmedToolCallID = toolCallId?.trim();
  if (trimmedToolCallID) {
    return resolveToolMessageIDWithCallID(runtime, trimmedToolCallID);
  }

  return resolveToolMessageIDWithFallback(runtime, event.step_id.trim() || event.id.trim());
}

export function resolvePreviewToolMessageID(
  runtime: ChatRuntimeState,
  toolCallIndex?: number,
  toolCallId?: string,
): string {
  if (toolCallIndex === undefined) {
    return ensureToolMessageIDWithCallID(runtime, toolCallId);
  }

  const existing = runtime.previewStructuredToolMessageIdsByIndex.get(toolCallIndex);
  if (existing) {
    rememberStructuredToolCallID({
      runtime,
      toolCallIndex,
      toolCallId,
      messageId: existing,
    });
    return existing;
  }

  const generated = ensureToolMessageIDWithCallID(runtime, toolCallId)
    || createStructuredPreviewMessageID(runtime, toolCallIndex);
  runtime.previewStructuredToolMessageIdsByIndex.set(toolCallIndex, generated);
  rememberStructuredToolCallID({
    runtime,
    toolCallIndex,
    toolCallId,
    messageId: generated,
  });
  return generated;
}

export function resolveStructuredToolCallID(runtime: ChatRuntimeState, toolCallIndex: number): string | undefined {
  return runtime.previewStructuredToolCallIdsByIndex.get(toolCallIndex)?.trim() || undefined;
}

export function enqueuePendingPreviewMessage(runtime: ChatRuntimeState, messageId: string): void {
  if (!runtime.pendingPreviewQueue.includes(messageId)) {
    runtime.pendingPreviewQueue.push(messageId);
  }
}

export function clearStructuredPreviewLookups(runtime: ChatRuntimeState): void {
  runtime.previewStructuredToolCallIdsByIndex.clear();
  runtime.previewStructuredToolMessageIdsByIndex.clear();
}

export function resolvePreviewToolCallID(runtime: ChatRuntimeState, messageId: string): string | undefined {
  for (const [toolCallId, currentMessageID] of runtime.toolMessageIds.entries()) {
    if (currentMessageID === messageId) {
      return toolCallId;
    }
  }
  return undefined;
}

export function resolvePreviewToolName(runtime: ChatRuntimeState, messageId: string): string | undefined {
  const previewToolName = runtime.previewToolNamesByMessageId.get(messageId)?.trim();
  if (previewToolName) {
    return previewToolName;
  }
  return formatToolIDName(runtime.previewToolIDByMessageId.get(messageId));
}

export function formatToolIDName(toolId?: string): string | undefined {
  const trimmed = toolId?.trim();
  if (!trimmed) {
    return undefined;
  }
  return `tool#${trimmed}`;
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

function rememberStructuredToolCallID(input: StructuredToolCallLookupInput): void {
  const { messageId, runtime, toolCallId, toolCallIndex } = input;
  const trimmedToolCallID = toolCallId?.trim();
  if (!trimmedToolCallID) {
    return;
  }

  runtime.toolMessageIds.set(trimmedToolCallID, messageId);
  runtime.previewStructuredToolCallIdsByIndex.set(toolCallIndex, trimmedToolCallID);
}

function createStructuredPreviewMessageID(runtime: ChatRuntimeState, toolCallIndex: number): string {
  const previewSeq = runtime.nextStructuredToolPreviewSeq;
  runtime.nextStructuredToolPreviewSeq += 1;
  return `stream-tool:${runtime.traceId}:preview:${previewSeq}:index:${toolCallIndex}`;
}
