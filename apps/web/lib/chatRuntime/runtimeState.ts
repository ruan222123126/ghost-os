import {
  parseAgentDonePayload,
  parseAgentErrorPayload,
  parseAgentRunStartedPayload,
  parseAgentStreamMessagePayload,
} from '@/lib/api/agent/parser';
import { createToolTagStreamState, type ToolTagStreamState } from '@/lib/toolTagText';
import type { AgentStreamEvent } from '@/lib/types';

export interface ChatRuntimeState {
  assistantBuffer: string;
  assistantMessageId: string;
  pendingPreviewQueue: string[];
  previewToolArgs: Map<string, string>;
  previewToolCallSeqToID: Map<number, string>;
  previewToolIDByMessageId: Map<string, string>;
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
    pendingPreviewQueue: [],
    previewToolArgs: new Map(),
    previewToolCallSeqToID: new Map(),
    previewToolIDByMessageId: new Map(),
    sessionId: sessionId?.trim() || '',
    toolMessageIds: new Map(),
    toolTagState: createToolTagStreamState(),
    traceId: trimmedTraceId,
  };
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
