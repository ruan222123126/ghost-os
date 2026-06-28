import type { SetStateAction } from 'react';
import type { ActiveAgentRun } from '@/lib/chat-stream/types';
import type { ChatRuntimeAction } from '@/lib/chatRuntime/actions';
import type {
  AgentRuntimeType,
  ChatMessage,
  ChatSendInput,
  ExternalAgentMode,
  ExternalCodexPermissionMode,
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
  clearMessages: (sessionId?: string) => void;
  backgroundCompletedSessionIds: ReadonlySet<string>;
  clearBackgroundCompletion: (sessionId: string) => void;
  dropSessionState: (sessionId: string) => void;
  shouldLoadSessionHistory: (sessionId: string) => boolean;
}

export interface UseBridgeChatOptions {
  currentSessionId: string;
  externalCodexPermissionMode?: ExternalCodexPermissionMode;
  externalProjectRoot?: string;
  onSessionResolved?: (sessionId: string) => void;
}

export interface StreamAgentRunInput {
  agentRuntime?: AgentRuntimeType;
  codexMode?: ExternalAgentMode;
  images?: SessionImageContent[];
  message: string;
  mode?: ChatSendInput['mode'];
  model?: string;
  permissionMode?: ExternalCodexPermissionMode;
  projectRoot?: string;
  sessionId?: string;
  signal?: AbortSignal;
  traceId: string;
}

export interface ChatStreamRunResult {
  sessionId: string;
  terminalType: 'awaiting_human' | 'done' | '';
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
  backgroundCompletedSessionIds: ReadonlySet<string>;
  getActiveRun: (sessionId: string) => ActiveAgentRun | null;
  getStopPending: (sessionId: string) => boolean;
  resolveActiveRunSessionId: (traceId: string, fallbackSessionId: string) => string;
  getNextHistoryBefore: (sessionId: string) => number | null;
  hasPendingQuestionInSession: (sessionId: string) => boolean;
  shouldLoadSessionHistory: (sessionId: string) => boolean;
  setCommittedMessages: (sessionId: string, updater: SetStateAction<ChatMessage[]>) => void;
  setLoading: (sessionId: string, value: boolean) => void;
  beginHistorySync: (sessionId: string) => void;
  endHistorySync: (sessionId: string) => void;
  setHistoryLoading: (sessionId: string, value: boolean) => void;
  setLoadingOlderHistory: (sessionId: string, value: boolean) => void;
  setChatError: (sessionId: string, value: string) => void;
  setActiveRun: (sessionId: string, value: ActiveAgentRun | null) => void;
  setStopPending: (sessionId: string, value: boolean) => void;
  setHasOlderHistory: (sessionId: string, value: boolean) => void;
  setNextHistoryBefore: (sessionId: string, value: number | null) => void;
  applyRuntimeActions: (sessionId: string, actions: ChatRuntimeAction[]) => void;
  appendCommittedMessages: (sessionId: string, nextMessages: ChatMessage[]) => void;
  clearChatError: (sessionId: string) => void;
  replaceWithErrorMessage: (sessionId: string, messageText: string) => void;
  appendErrorMessage: (sessionId: string, messageText: string) => void;
  appendStreamingAssistantText: (sessionId: string, text: string) => void;
  clearStreamingAssistantText: (sessionId: string) => void;
  clearStreamingThinkingText: (sessionId: string) => void;
  clearStreamingState: (sessionId: string) => void;
  upsertStreamingTool: (sessionId: string, tool: StreamingToolState) => void;
  clearStreamingTools: (sessionId: string) => void;
  upsertPendingQuestion: (sessionId: string, question: PendingQuestionMessage) => void;
  removePendingQuestion: (sessionId: string, questionId: string) => void;
  clearPendingQuestions: (sessionId: string) => void;
  hydrateTurnDraft: (sessionId: string, draft: SessionTurnDraft | null | undefined) => void;
  clearMessages: (sessionId: string) => void;
  migrateSessionState: (fromSessionId: string, toSessionId: string) => void;
  markBackgroundCompleted: (sessionId: string) => void;
  clearBackgroundCompletion: (sessionId: string) => void;
  dropSessionState: (sessionId: string) => void;
}
