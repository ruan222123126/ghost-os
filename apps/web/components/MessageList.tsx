// MessageList component used by the web console chat/session interface.

'use client';

import type { FC } from 'react';
import { useEffect, useRef } from 'react';
import { QuestionInput } from '@/components/QuestionInput';
import type { ChatMessage } from '@/lib/types';

interface MessageListProps {
  messages: ChatMessage[];
  loading: boolean;
  onAnswerQuestion: (questionId: string, answer: string) => Promise<void>;
  onCancelQuestion: (questionId: string) => Promise<void>;
}

export const MessageList: FC<MessageListProps> = ({ messages, loading, onAnswerQuestion, onCancelQuestion }) => {
  const endRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: 'smooth', block: 'end' });
  }, [messages, loading]);

  if (messages.length === 0) {
    return (
      <div className="ui-empty flex h-full min-h-[320px] items-center justify-center p-6 text-center text-sm">
        Start by configuring a model, then send a task to Ghost-OS.
      </div>
    );
  }

  return (
    <div className="ui-panel ui-scroll h-full min-h-[320px] space-y-3 overflow-y-auto p-4">
      {messages.map((message) => {
        const isUser = message.kind === 'user';
        const isError = message.kind === 'error';
        const isPendingQuestion = message.kind === 'pending_question';
        const isQuestion = message.kind === 'question';
        const isSystem = message.kind === 'system';
        const isTool = message.kind === 'tool';
        const label = isUser
          ? 'You'
          : isError
            ? 'Error'
            : isPendingQuestion || isQuestion
              ? 'Question'
              : isSystem
                ? 'System'
                : isTool
                  ? message.toolName ? `Tool · ${message.toolName}` : 'Tool'
                  : 'Assistant';

        return (
          <div key={message.id} className={`flex animate-riseSoft ${isUser ? 'justify-end' : 'justify-start'}`}>
            <div
              className={`max-w-[85%] rounded-2xl border px-4 py-3 text-sm leading-relaxed shadow-panel ${
                isUser
                  ? 'border-app-accent/35 bg-app-accent/10 text-app-text'
                  : isError
                    ? 'border-rose-300/40 bg-rose-300/10 text-rose-200'
                    : isPendingQuestion || isQuestion
                      ? 'border-amber-300/35 bg-amber-300/8 text-amber-100'
                      : isSystem
                        ? 'border-app-border/80 bg-app-field/70 text-app-muted2'
                        : isTool
                          ? 'border-sky-300/28 bg-sky-300/10 text-sky-100'
                          : 'border-app-border/90 bg-app-panel/92 text-app-text'
              }`}
            >
              <div className="ui-hint mb-1 uppercase tracking-[0.14em]">{label}</div>
              <p className="whitespace-pre-wrap">{message.content}</p>
              {isPendingQuestion && (
                <QuestionInput
                  loading={loading}
                  selectionMode={message.selectionMode}
                  options={message.options}
                  onAnswer={(answer) => onAnswerQuestion(message.questionId, answer)}
                  onCancel={() => onCancelQuestion(message.questionId)}
                />
              )}
            </div>
          </div>
        );
      })}

      {loading && (
        <div className="flex justify-start animate-riseSoft">
          <div className="ui-panel-soft px-4 py-3 text-sm text-app-muted">
            Thinking...
          </div>
        </div>
      )}
      <div ref={endRef} />
    </div>
  );
};
