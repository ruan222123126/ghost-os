import type { ChatMessage, PendingQuestionMessage, StreamingToolState } from '@/lib/types';

export type ChatRuntimeAction =
  | {
    type: 'append_streaming_assistant_text';
    text: string;
  }
  | {
    type: 'clear_streaming_assistant_text';
  }
  | {
    type: 'append_streaming_thinking_text';
    text: string;
  }
  | {
    type: 'clear_streaming_thinking_text';
  }
  | {
    type: 'append_committed_messages';
    messages: ChatMessage[];
  }
  | {
    type: 'upsert_streaming_tool';
    tool: StreamingToolState;
  }
  | {
    type: 'clear_streaming_tools';
  }
  | {
    type: 'upsert_pending_question';
    question: PendingQuestionMessage;
  }
  | {
    type: 'remove_pending_question';
    questionId: string;
  }
  | {
    type: 'clear_pending_questions';
  };
