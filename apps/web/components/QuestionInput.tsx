// QuestionInput component used by the web console chat/session interface.

'use client';

import type { FC } from 'react';
import { useState } from 'react';
import { ignorePromise } from '@/lib/errors';

interface QuestionInputProps {
  loading: boolean;
  onAnswer: (answer: string) => Promise<void>;
}

export const QuestionInput: FC<QuestionInputProps> = ({ loading, onAnswer }) => {
  const [value, setValue] = useState('');

  async function submit() {
    const answer = value.trim();
    if (!answer || loading) {
      return;
    }

    setValue('');
    await onAnswer(answer);
  }

  return (
    <div className="mt-3 rounded-xl border border-amber-300/35 bg-amber-300/10 p-3">
      <textarea
        value={value}
        disabled={loading}
        aria-label="Answer question"
        onChange={(event) => setValue(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === 'Enter' && !event.shiftKey) {
            event.preventDefault();
            ignorePromise(submit());
          }
        }}
        rows={2}
        placeholder="Type your answer..."
        className="mono w-full resize-none rounded-lg border border-amber-200/20 bg-[#120f06] px-3 py-2 text-sm text-amber-100 outline-none transition focus:border-amber-300/50"
      />
      <div className="mt-2 flex items-center justify-between">
        <span className="text-xs text-amber-200/70">Enter to submit answer</span>
        <button
          type="button"
          onClick={() => {
            ignorePromise(submit());
          }}
          disabled={loading || value.trim().length === 0}
          className="rounded-lg border border-amber-300/40 bg-amber-300/20 px-3 py-1.5 text-sm font-medium text-amber-100 transition hover:bg-amber-300/30 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {loading ? 'Submitting...' : 'Submit'}
        </button>
      </div>
    </div>
  );
};
