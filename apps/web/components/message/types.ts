import type { ChatMessage } from '@/lib/types';

export interface MessageListProps {
  messages: ChatMessage[];
  loading: boolean;
  onAnswerQuestion: (questionId: string, answer: string) => Promise<void>;
  onCancelQuestion: (questionId: string) => Promise<void>;
}

export interface MessageRowProps {
  message: ChatMessage;
  loading: boolean;
  onAnswerQuestion: (questionId: string, answer: string) => Promise<void>;
  onCancelQuestion: (questionId: string) => Promise<void>;
}
