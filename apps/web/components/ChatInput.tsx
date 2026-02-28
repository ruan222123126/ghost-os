'use client';

import type { FC } from 'react';
import { useState } from 'react';
import { ignorePromise } from '@/lib/errors';

interface ChatInputProps {
  loading: boolean;
  disabled: boolean;
  onSend: (message: string) => Promise<void>;
}

export const ChatInput: FC<ChatInputProps> = ({ loading, disabled, onSend }) => {
  const [value, setValue] = useState('');

  async function submit() {
    const message = value.trim();
    if (!message || loading || disabled) {
      return;
    }

    setValue('');
    await onSend(message);
  }

  return (
    <div className="rounded-2xl border border-app-border/70 bg-app-panel/90 p-4 shadow-xl backdrop-blur">
      <textarea
        value={value}
        disabled={disabled}
        aria-label="Message input"
        onChange={(event) => setValue(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === 'Enter' && !event.shiftKey) {
            event.preventDefault();
            ignorePromise(submit());
          }
        }}
        placeholder="Type a task for Ghost-OS..."
        rows={3}
        className="mono w-full resize-none rounded-xl border border-app-border bg-[#070c17] px-3 py-2 text-sm text-app-text outline-none transition focus:border-app-accent"
      />
      <div className="mt-3 flex items-center justify-between">
        <span className="text-xs text-app-muted">
          {disabled ? 'Waiting for runtime config...' : 'Enter to send, Shift+Enter for newline'}
        </span>
        <button
          type="button"
          onClick={() => {
            ignorePromise(submit());
          }}
          disabled={loading || disabled || value.trim().length === 0}
          className="rounded-lg border border-app-accent/40 bg-app-accent/20 px-4 py-2 text-sm font-medium text-app-text transition hover:bg-app-accent/30 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {loading ? 'Sending...' : 'Send'}
        </button>
      </div>
    </div>
  );
};
