import {
  resolveEventSessionId,
  type ChatRuntimeState,
} from '@/lib/chatRuntime/runtimeState';
import type { AgentStreamEvent } from '@/lib/types';
import type { ChatStateControls, UseBridgeChatOptions } from './types';

export function syncActiveSession(options: {
  activeRunRef: ChatStateControls['activeRunRef'];
  currentSessionId: UseBridgeChatOptions['currentSessionId'];
  event: AgentStreamEvent;
  onSessionResolved: UseBridgeChatOptions['onSessionResolved'];
  setActiveRun: ChatStateControls['setActiveRun'];
  runtime: ChatRuntimeState;
}): void {
  const sessionId = resolveEventSessionId(options.event);
  if (!sessionId || sessionId === options.runtime.sessionId) {
    return;
  }

  options.runtime.sessionId = sessionId;
  const currentRun = options.activeRunRef.current;
  if (currentRun && currentRun.sessionId !== sessionId) {
    options.setActiveRun({ ...currentRun, sessionId });
  }
  if (sessionId !== options.currentSessionId) {
    options.onSessionResolved?.(sessionId);
  }
}
