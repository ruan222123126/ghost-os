import {
  parseAgentDonePayload,
  parseAgentErrorPayload,
  parseAgentRunStartedPayload,
  parseAgentStreamMessagePayload,
} from '@/lib/api/agent/parser';
import { createToolTagStreamState } from '@/lib/toolTagText';
import type { AgentStreamEvent } from '@/lib/types';
import type { ChatStateControls, UseBridgeChatOptions } from './types';
import type { StreamRuntimeState } from './chatStreamControllerTypes';

export function createStreamRuntimeState(traceId: string, sessionId?: string): StreamRuntimeState {
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

export function syncActiveSession(options: {
  activeRunRef: ChatStateControls['activeRunRef'];
  currentSessionId: UseBridgeChatOptions['currentSessionId'];
  event: AgentStreamEvent;
  onSessionResolved: UseBridgeChatOptions['onSessionResolved'];
  setActiveRun: ChatStateControls['setActiveRun'];
  state: StreamRuntimeState;
}): void {
  const sessionId = resolveEventSessionId(options.event);
  if (!sessionId || sessionId === options.state.sessionId) {
    return;
  }

  options.state.sessionId = sessionId;
  const currentRun = options.activeRunRef.current;
  if (currentRun && currentRun.sessionId !== sessionId) {
    options.setActiveRun({ ...currentRun, sessionId });
  }
  if (sessionId !== options.currentSessionId) {
    options.onSessionResolved?.(sessionId);
  }
}

function resolveEventSessionId(event: AgentStreamEvent): string {
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
