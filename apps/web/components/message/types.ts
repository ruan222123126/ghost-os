import type {
  ChatMessage,
  PendingQuestionMessage,
  StreamingAssistantSegment,
  StreamingThinkingSegment,
  StreamingToolState,
} from '@/lib/types';

export interface MessageListProps {
  committedMessages: ChatMessage[];
  showSystemPromptMessages: boolean;
  assistantMarkdownEnabled: boolean;
  toolCallCompactOutputEnabled: boolean;
  streamingAssistantSegments: StreamingAssistantSegment[];
  streamingThinkingSegments: StreamingThinkingSegment[];
  streamingItemOrder: string[];
  streamingTools: StreamingToolState[];
  pendingQuestions: PendingQuestionMessage[];
  loading: boolean;
  loadingOlderHistory: boolean;
  hasOlderHistory: boolean;
  loadOlderHistory: () => Promise<void>;
  onAnswerQuestion: (questionId: string, answer: string) => Promise<void>;
  onCancelQuestion: (questionId: string) => Promise<void>;
}

export interface MessageRowProps {
  message: ChatMessage;
  assistantMarkdownEnabled: boolean;
  toolCallCompactOutputEnabled: boolean;
  isToolCardOpen?: boolean;
  isThinkingPanelOpen?: boolean;
  loading: boolean;
  onToggleToolCard?: (messageId: string) => void;
  onToggleThinkingPanel?: (messageId: string) => void;
  onAnswerQuestion: (questionId: string, answer: string) => Promise<void>;
  onCancelQuestion: (questionId: string) => Promise<void>;
}

export type MessageListRow =
  | {
    key: string;
    kind: 'message';
    message: ChatMessage;
  }
  | {
    key: 'history-loading';
    kind: 'history_loading';
  }
  | {
    key: 'thinking-indicator';
    kind: 'thinking_indicator';
  };
