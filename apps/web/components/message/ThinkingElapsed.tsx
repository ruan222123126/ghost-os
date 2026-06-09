import type { FC } from 'react';
import { useEffect, useState } from 'react';
import { useWebLocale } from '@/lib/i18n/provider';

const THINKING_ELAPSED_UPDATE_MS = 1000;
const THINKING_ELAPSED_NEXT_TICK_BUFFER_MS = 16;

interface ThinkingElapsedProps {
  className: string;
  startedAtMs: number | null;
}

export const ThinkingElapsed: FC<ThinkingElapsedProps> = ({ className, startedAtMs }) => {
  const { copy } = useWebLocale();
  const elapsedSeconds = useThinkingElapsedSeconds(startedAtMs);

  if (startedAtMs === null) {
    return null;
  }

  return <span className={className} aria-hidden="true">{copy.chat.thinkingElapsed(elapsedSeconds)}</span>;
};

function useThinkingElapsedSeconds(startedAtMs: number | null): number {
  const [nowMs, setNowMs] = useState(() => Date.now());

  useEffect(() => {
    if (startedAtMs === null) {
      return;
    }

    let timeoutId: number | null = null;
    const update = () => {
      const now = Date.now();
      setNowMs(now);
      const elapsedMs = Math.max(0, now - startedAtMs);
      const delayMs = THINKING_ELAPSED_UPDATE_MS
        - (elapsedMs % THINKING_ELAPSED_UPDATE_MS)
        + THINKING_ELAPSED_NEXT_TICK_BUFFER_MS;
      timeoutId = window.setTimeout(update, delayMs);
    };

    update();

    return () => {
      if (timeoutId !== null) {
        window.clearTimeout(timeoutId);
      }
    };
  }, [startedAtMs]);

  if (startedAtMs === null) {
    return 0;
  }

  return Math.max(0, Math.floor((nowMs - startedAtMs) / THINKING_ELAPSED_UPDATE_MS));
}
