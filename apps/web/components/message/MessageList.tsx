'use client';

import type { FC } from 'react';
import { useEffect, useRef } from 'react';
import { EmptyState } from './EmptyState';
import { MessageRow } from './MessageRow';
import { ThinkingIndicator } from './ThinkingIndicator';
import type { MessageListProps } from './types';

export const MessageList: FC<MessageListProps> = ({ messages, loading, onAnswerQuestion, onCancelQuestion }) => {
  const endRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: 'smooth', block: 'end' });
  }, [loading, messages]);

  if (messages.length === 0) {
    return <EmptyState />;
  }

  return (
    <div className="messages ui-scroll" aria-live="polite">
      {messages.map((message) => (
        <MessageRow
          key={message.id}
          message={message}
          loading={loading}
          onAnswerQuestion={onAnswerQuestion}
          onCancelQuestion={onCancelQuestion}
        />
      ))}
      {loading ? <ThinkingIndicator /> : null}
      <div ref={endRef} />
    </div>
  );
};
