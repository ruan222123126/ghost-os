import type { ChatRuntimeAction } from './actions';
import { TOOL_PENDING_STATUS } from './constants';
import { markRuntimeThinkingBoundary, type ChatRuntimeState } from './runtimeState';
import type { ToolTagStreamEvent, ToolTagStreamUnit } from '@/lib/toolTagText';
import { normalizeToolName } from '@/lib/toolNames';
import { formatToolIDName } from './toolPreviewState';

interface PendingToolActionInput {
  content: string;
  id: string;
  toolName?: string;
  traceId: string;
}

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
    return buildPendingToolAction({
      id: messageId,
      content: args,
      toolName: formatToolIDName(event.toolId),
      traceId,
    });
  }

  if (event.type === 'tool_args') {
    const messageId = runtime.previewToolCallSeqToID.get(event.callSeq);
    if (!messageId) {
      return null;
    }

    const current = runtime.previewToolArgs.get(messageId) || '';
    const next = `${current}${event.argsDelta}`;
    runtime.previewToolArgs.set(messageId, next);
    return buildPendingToolAction({
      id: messageId,
      content: next,
      toolName: formatToolIDName(runtime.previewToolIDByMessageId.get(messageId)),
      traceId,
    });
  }

  const messageId = ensurePreviewMessageID(runtime, event.callSeq, event.toolId);
  runtime.previewToolArgs.set(messageId, event.argsText);
  if (!runtime.pendingPreviewQueue.includes(messageId)) {
    runtime.pendingPreviewQueue.push(messageId);
  }
  return buildPendingToolAction({
    id: messageId,
    content: event.argsText,
    toolName: formatToolIDName(event.toolId),
    traceId,
  });
}

function buildPendingToolAction(input: PendingToolActionInput): ChatRuntimeAction {
  const { content, id, toolName, traceId } = input;
  const normalizedToolName = normalizeToolName(toolName);
  const toolInput = normalizedToolName === 'bash_exec' || normalizedToolName === 'codex_exec'
    ? content
    : '';
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
