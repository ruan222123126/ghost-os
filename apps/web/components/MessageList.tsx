'use client';

import type { FC } from 'react';
import { useEffect, useRef } from 'react';
import { QuestionInput } from '@/components/QuestionInput';
import type { ChatMessage } from '@/lib/types';

interface MessageListProps {
  messages: ChatMessage[];
  loading: boolean;
  onAnswerQuestion: (questionId: string, answer: string) => Promise<void>;
}

export const MessageList: FC<MessageListProps> = ({ messages, loading, onAnswerQuestion }) => {
  const endRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: 'smooth', block: 'end' });
  }, [messages, loading]);

  if (messages.length === 0) {
    return (
      <div className="flex h-full min-h-[320px] items-center justify-center rounded-2xl border border-dashed border-app-border bg-app-panel/60 p-6 text-center text-app-muted">
        Start by configuring a model, then send a task to Ghost-OS.
      </div>
    );
  }

  return (
    <div className="h-full min-h-[320px] space-y-3 overflow-y-auto rounded-2xl border border-app-border/80 bg-app-panel/70 p-4 shadow-lg backdrop-blur-sm">
      {messages.map((message) => {
        const isUser = message.kind === 'user';
        const isError = message.kind === 'error';
        const isPendingQuestion = message.kind === 'pending_question';
        const label = isUser ? 'You' : isError ? 'Error' : isPendingQuestion ? 'Question' : 'Assistant';

        return (
          <div key={message.id} className={`flex animate-rise ${isUser ? 'justify-end' : 'justify-start'}`}>
            <div
              className={`max-w-[85%] rounded-2xl px-4 py-3 text-sm leading-relaxed shadow-lg ${
                isUser
                  ? 'border border-emerald-300/25 bg-emerald-300/10 text-emerald-100'
                  : isError
                    ? 'border border-rose-300/40 bg-rose-300/10 text-rose-200'
                    : isPendingQuestion
                      ? 'border border-amber-300/45 bg-amber-300/10 text-amber-100'
                    : 'border border-app-border bg-[#10192b] text-app-text'
              }`}
            >
              <div className="mb-1 text-[11px] uppercase tracking-wide text-app-muted">{label}</div>
              <p className="whitespace-pre-wrap">{message.content}</p>
              {isPendingQuestion && (
                <QuestionInput
                  loading={loading}
                  onAnswer={(answer) => onAnswerQuestion(message.questionId, answer)}
                />
              )}
            </div>
          </div>
        );
      })}

      {loading && (
        <div className="flex justify-start animate-rise">
          <div className="rounded-xl border border-app-border bg-[#10192b] px-4 py-3 text-sm text-app-muted">
            Thinking...
          </div>
        </div>
      )}
      <div ref={endRef} />
    </div>
  );
};
