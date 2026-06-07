import type { Dispatch, MutableRefObject, SetStateAction } from 'react';
import type { ActiveAgentRun } from '@/lib/chat-stream/types';
import type { ChatRuntimeAction } from '@/lib/chatRuntime/actions';
import type {
  ChatMessage,
  ChatSendInput,
  PendingQuestionMessage,
  SessionImageContent,
  SessionTurnDraft,
  StreamingAssistantSegment,
  StreamingThinkingSegment,
  StreamingToolState,
} from '@/lib/types';

export type { ActiveAgentRun } from '@/lib/chat-stream/types';

export interface UseBridgeChatResult {
  committedMessages: ChatMessage[];
  streamingAssistantSegments: StreamingAssistantSegment[];
  streamingThinkingSegments: StreamingThinkingSegment[];
  activeStreamingThinkingId: string | null;
  streamingItemOrder: string[];
  streamingTools: StreamingToolState[];
  pendingQuestions: PendingQuestionMessage[];
  loading: boolean;
  historySyncing: boolean;
  historyLoading: boolean;
  loadingOlderHistory: boolean;
  chatError: string;
  hasPendingQuestion: boolean;
  canStop: boolean;
  hasOlderHistory: boolean;
  sendChatMessage: (input: ChatSendInput) => Promise<void>;
  stopCurrentRun: () => Promise<void>;
  answerQuestion: (questionId: string, answer: string) => Promise<void>;
  cancelQuestion: (questionId: string) => Promise<void>;
  loadSessionHistory: (sessionId: string) => Promise<void>;
  loadOlderHistory: () => Promise<void>;
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
  committedMessages: ChatMessage[];
  streamingAssistantSegments: StreamingAssistantSegment[];
  streamingThinkingSegments: StreamingThinkingSegment[];
  activeStreamingThinkingId: string | null;
  streamingItemOrder: string[];
  streamingTools: StreamingToolState[];
  pendingQuestions: PendingQuestionMessage[];
  loading: boolean;
  historySyncing: boolean;
  historyLoading: boolean;
  loadingOlderHistory: boolean;
  chatError: string;
  activeRun: ActiveAgentRun | null;
  stopPending: boolean;
  hasOlderHistory: boolean;
  nextHistoryBefore: number | null;
  activeRunRef: MutableRefObject<ActiveAgentRun | null>;
  stopPendingRef: MutableRefObject<boolean>;
  setCommittedMessages: Dispatch<SetStateAction<ChatMessage[]>>;
  setLoading: (value: boolean) => void;
  beginHistorySync: () => void;
  endHistorySync: () => void;
  setHistoryLoading: (value: boolean) => void;
  setLoadingOlderHistory: (value: boolean) => void;
  setChatError: (value: string) => void;
  setActiveRun: (value: ActiveAgentRun | null) => void;
  setStopPending: (value: boolean) => void;
  setHasOlderHistory: (value: boolean) => void;
  setNextHistoryBefore: (value: number | null) => void;
  applyRuntimeActions: (actions: ChatRuntimeAction[]) => void;
  appendCommittedMessages: (nextMessages: ChatMessage[]) => void;
  clearChatError: () => void;
  replaceWithErrorMessage: (messageText: string) => void;
  appendErrorMessage: (messageText: string) => void;
  appendStreamingAssistantText: (text: string) => void;
  clearStreamingAssistantText: () => void;
  clearStreamingThinkingText: () => void;
  clearStreamingState: () => void;
  upsertStreamingTool: (tool: StreamingToolState) => void;
  clearStreamingTools: () => void;
  upsertPendingQuestion: (question: PendingQuestionMessage) => void;
  removePendingQuestion: (questionId: string) => void;
  clearPendingQuestions: () => void;
  hydrateTurnDraft: (sessionId: string, draft: SessionTurnDraft | null | undefined) => void;
  clearMessages: () => void;
}
