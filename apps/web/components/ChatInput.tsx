// ChatInput component used by the web console chat/session interface.

'use client';

import type { FC } from 'react';
import { TextareaSubmitInput } from '@/components/TextareaSubmitInput';

interface ChatInputProps {
  loading: boolean;
  disabled: boolean;
  awaitingQuestion?: boolean;
  onSend: (message: string) => Promise<void>;
}

export const ChatInput: FC<ChatInputProps> = ({ loading, disabled, awaitingQuestion = false, onSend }) => {
  return (
    <TextareaSubmitInput
      loading={loading}
      disabled={disabled}
      onSubmit={onSend}
      ariaLabel="Message input"
      placeholder="Type a task for Ghost-OS..."
      rows={3}
      hint={
        awaitingQuestion
          ? 'Agent is asking a question, please answer above...'
          : disabled && loading
            ? 'Sending...'
          : disabled
            ? 'Waiting for runtime config...'
            : 'Enter to send, Shift+Enter for newline'
      }
      submitLabel="Send"
      submittingLabel="Sending..."
      containerClassName="ui-panel animate-riseSoft p-4"
      textareaClassName="ui-textarea mono min-h-[96px] w-full resize-none"
      actionsClassName="mt-3 flex items-center justify-between gap-3"
      hintClassName="ui-hint"
      buttonClassName="ui-btn min-w-[104px]"
    />
  );
};
