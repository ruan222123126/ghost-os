import type { ChatMessage } from '@/lib/types';
import type { MessageListProjection, ToolCardViewModel } from '@/lib/chat-view/types';
import type { PostSendFocusRequest } from '@/hooks/chat/types';

export interface MessageListProps {
  view: MessageListProjection;
  assistantMarkdownEnabled: boolean;
  hasOlderHistory: boolean;
  postSendFocusRequest: PostSendFocusRequest | null;
  loadOlderHistory: () => Promise<void>;
  onAnswerQuestion: (questionId: string, answer: string) => Promise<void>;
  onCancelQuestion: (questionId: string) => Promise<void>;
}

export interface MessageRowProps {
  message: ChatMessage;
  toolCard?: ToolCardViewModel;
  assistantMarkdownEnabled: boolean;
  hasTrailingTool?: boolean;
  isToolCardOpen?: boolean;
  isThinkingPanelOpen?: boolean;
  thinkingStartedAtMs?: number | null;
  loading: boolean;
  onToggleToolCard?: (messageId: string) => void;
  onToggleThinkingPanel?: (messageId: string) => void;
  onAnswerQuestion: (questionId: string, answer: string) => Promise<void>;
  onCancelQuestion: (questionId: string) => Promise<void>;
}
