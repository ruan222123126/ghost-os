import type { Dispatch, MutableRefObject, SetStateAction } from 'react';
import type { ChatMessage, ChatSendInput, SessionImageContent } from '@/lib/types';

export interface ActiveAgentRun {
  abortController?: AbortController;
  sessionId: string;
  traceId: string;
}

export interface UseBridgeChatResult {
  messages: ChatMessage[];
  loading: boolean;
  historyLoading: boolean;
  chatError: string;
  hasPendingQuestion: boolean;
  canStop: boolean;
  sendChatMessage: (input: ChatSendInput) => Promise<void>;
  stopCurrentRun: () => Promise<void>;
  answerQuestion: (questionId: string, answer: string) => Promise<void>;
  cancelQuestion: (questionId: string) => Promise<void>;
  loadSessionHistory: (sessionId: string) => Promise<void>;
  clearMessages: () => void;
}

export interface UseBridgeChatOptions {
  currentSessionId: string;
  onSessionResolved?: (sessionId: string) => void;
}

export interface StreamAgentRunInput {
  images?: SessionImageContent[];
  message: string;
  sessionId?: string;
  signal?: AbortSignal;
  traceId: string;
}

export interface ChatStateControls {
  messages: ChatMessage[];
  loading: boolean;
  historyLoading: boolean;
  chatError: string;
  activeRun: ActiveAgentRun | null;
  stopPending: boolean;
  activeRunRef: MutableRefObject<ActiveAgentRun | null>;
  stopPendingRef: MutableRefObject<boolean>;
  setMessages: Dispatch<SetStateAction<ChatMessage[]>>;
  setLoading: (value: boolean) => void;
  setHistoryLoading: (value: boolean) => void;
  setChatError: (value: string) => void;
  setActiveRun: (value: ActiveAgentRun | null) => void;
  setStopPending: (value: boolean) => void;
  appendMessages: (nextMessages: ChatMessage[]) => void;
  clearChatError: () => void;
  replaceWithErrorMessage: (messageText: string) => void;
  appendErrorMessage: (messageText: string) => void;
  clearMessages: () => void;
}
