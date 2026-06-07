import type { ChatRuntimeAction } from './actions';
import { TOOL_PENDING_STATUS } from './constants';
import { markRuntimeThinkingBoundary, type ChatRuntimeState } from './runtimeState';
import type { ToolTagStreamEvent, ToolTagStreamUnit } from '@/lib/toolTagText';
import { normalizeToolName } from '@/lib/toolNames';

export function projectToolTagUnits(
  runtime: ChatRuntimeState,
  traceId: string,
  units: ToolTagStreamUnit[],
): ChatRuntimeAction[] {
  const actions: ChatRuntimeAction[] = [];
  for (const unit of units) {
    if (unit.type === 'text') {
      if (!unit.text) {
        continue;
      }
      markRuntimeThinkingBoundary(runtime);
      runtime.assistantBuffer = `${runtime.assistantBuffer}${unit.text}`;
      actions.push({
        type: 'append_streaming_assistant_text',
        text: unit.text,
      });
      continue;
    }

    const action = projectToolTagEvent(runtime, traceId, unit);
    if (action) {
      actions.push(action);
    }
  }
  return actions;
}

function projectToolTagEvent(
  runtime: ChatRuntimeState,
  traceId: string,
  event: ToolTagStreamEvent,
): ChatRuntimeAction | null {
  markRuntimeThinkingBoundary(runtime);
  if (event.type === 'tool_open') {
    const messageId = ensurePreviewMessageID(runtime, event.callSeq, event.toolId);
    const args = runtime.previewToolArgs.get(messageId) || '';
    return buildPendingToolAction(messageId, args, formatToolIDName(event.toolId), traceId);
  }

  if (event.type === 'tool_args') {
    const messageId = runtime.previewToolCallSeqToID.get(event.callSeq);
    if (!messageId) {
      return null;
    }

    const current = runtime.previewToolArgs.get(messageId) || '';
    const next = `${current}${event.argsDelta}`;
    runtime.previewToolArgs.set(messageId, next);
    return buildPendingToolAction(
      messageId,
      next,
      formatToolIDName(runtime.previewToolIDByMessageId.get(messageId)),
      traceId,
    );
  }

  const messageId = ensurePreviewMessageID(runtime, event.callSeq, event.toolId);
  runtime.previewToolArgs.set(messageId, event.argsText);
  if (!runtime.pendingPreviewQueue.includes(messageId)) {
    runtime.pendingPreviewQueue.push(messageId);
  }
  return buildPendingToolAction(messageId, event.argsText, formatToolIDName(event.toolId), traceId);
}

function buildPendingToolAction(
  id: string,
  content: string,
  toolName: string | undefined,
  traceId: string,
): ChatRuntimeAction {
  const toolInput = normalizeToolName(toolName) === 'bash_exec' ? content : '';
  return {
    type: 'upsert_streaming_tool',
    tool: {
      id,
      content,
      ...(toolInput ? { toolInput } : {}),
      toolName,
      toolStatus: TOOL_PENDING_STATUS,
      traceId,
    },
  };
}

function ensurePreviewMessageID(runtime: ChatRuntimeState, callSeq: number, toolId: string): string {
  const existing = runtime.previewToolCallSeqToID.get(callSeq);
  if (existing) {
    return existing;
  }

  const messageId = `stream-tag-tool:${runtime.traceId}:${callSeq}`;
  runtime.previewToolCallSeqToID.set(callSeq, messageId);
  runtime.previewToolIDByMessageId.set(messageId, toolId);
  if (!runtime.previewToolArgs.has(messageId)) {
    runtime.previewToolArgs.set(messageId, '');
  }
  return messageId;
}

function formatToolIDName(toolId?: string): string | undefined {
  const trimmed = toolId?.trim();
  if (!trimmed) {
    return undefined;
  }
  return `tool#${trimmed}`;
}
