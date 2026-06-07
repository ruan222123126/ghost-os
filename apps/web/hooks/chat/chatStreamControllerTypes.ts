import type { ChatRuntimeAction } from '@/lib/chatRuntime/actions';
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
  activeRunRef: ChatStateControls['activeRunRef'];
  applyRuntimeActions: (actions: ChatRuntimeAction[]) => void;
  currentSessionId: UseBridgeChatOptions['currentSessionId'];
  endHistorySync: ChatStateControls['endHistorySync'];
  onSessionResolved: UseBridgeChatOptions['onSessionResolved'];
  setChatError: ChatStateControls['setChatError'];
  setActiveRun: ChatStateControls['setActiveRun'];
  beginHistorySync: ChatStateControls['beginHistorySync'];
  syncRecentHistory: (sessionId: string) => Promise<void>;
}

export type { ChatRuntimeState };
