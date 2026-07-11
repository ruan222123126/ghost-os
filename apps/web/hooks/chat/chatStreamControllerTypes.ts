import type { ChatRuntimeState } from '@/lib/chatRuntime/runtimeState';
import type { ChatStateControls, UseBridgeChatOptions } from './types';

export interface StreamHumanRunOptions {
  answer: string;
  cancelled?: boolean;
  questionId: string;
  sessionId: string;
  signal?: AbortSignal;
  traceId: string;
}

export interface UseChatStreamControllerOptions {
  applyRuntimeActions: ChatStateControls['applyRuntimeActions'];
  endHistorySync: ChatStateControls['endHistorySync'];
  getCurrentSessionId: () => UseBridgeChatOptions['currentSessionId'];
  migrateSessionState: ChatStateControls['migrateSessionState'];
  onSessionResolved: UseBridgeChatOptions['onSessionResolved'];
  setChatError: ChatStateControls['setChatError'];
  beginHistorySync: ChatStateControls['beginHistorySync'];
  syncRecentHistory: (sessionId: string) => Promise<void>;
}

export type { ChatRuntimeState };
