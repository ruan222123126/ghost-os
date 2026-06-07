import type {
  ChatMessage,
  PendingQuestionMessage,
  StreamingAssistantSegment,
  StreamingThinkingSegment,
  StreamingToolState,
} from '@/lib/types';

export type ToolTone = 'running' | 'success' | 'error';
export type ToolCardTitleMode = 'status' | 'plain';

export interface ToolCardViewModel {
  title: string;
  tone: ToolTone;
  statusLabel: string;
  details: string;
  titleMode?: ToolCardTitleMode;
  showTerminalIcon?: boolean;
}

export interface ToolCardViewModelOptions {
  compactOutputEnabled?: boolean;
  fallbackTitle?: string;
  preparingDetails?: string;
}

export interface ChatViewInput {
  committedMessages: ChatMessage[];
  showSystemPromptMessages: boolean;
  streamingAssistantSegments: StreamingAssistantSegment[];
  streamingThinkingSegments: StreamingThinkingSegment[];
  activeStreamingThinkingId?: string | null;
  streamingItemOrder: string[];
  streamingTools: StreamingToolState[];
  pendingQuestions: PendingQuestionMessage[];
  loading: boolean;
}

export interface StreamingMessageRow {
  key: string;
  message: ChatMessage;
}

export interface MessageListMessageRow {
  key: string;
  kind: 'message';
  message: ChatMessage;
  toolCard?: ToolCardViewModel;
}

export type MessageListRow =
  | MessageListMessageRow
  | {
    key: 'processing-timer';
    kind: 'processing_timer';
  }
  | {
    key: 'history-loading';
    kind: 'history_loading';
  }
  | {
    key: 'thinking-indicator';
    kind: 'thinking_indicator';
  };

export interface ChatViewProjection {
  visibleCommittedMessages: ChatMessage[];
  streamingRows: StreamingMessageRow[];
  showThinkingIndicator: boolean;
  latestStreamingThinkingId: string | null;
  hasThinkingText: boolean;
  hasAssistantText: boolean;
  shouldAutoCollapseLatestThinkingPanel: boolean;
  visibleMessagesForPostSendOverflow: ChatMessage[];
}

export interface MessageListProjectionInput extends ChatViewInput {
  loadingOlderHistory: boolean;
  toolCard?: ToolCardViewModelOptions;
}

export interface MessageListProjection extends ChatViewProjection {
  estimatedRowSize: number;
  rows: MessageListRow[];
  rowCount: number;
  loading: boolean;
  loadingOlderHistory: boolean;
}
