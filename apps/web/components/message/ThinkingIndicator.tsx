import type { FC } from 'react';
import { useWebLocale } from '@/lib/i18n/provider';
import { ThinkingElapsed } from './ThinkingElapsed';

export const ThinkingIndicator: FC<{ startedAtMs: number | null }> = ({ startedAtMs }) => {
  const { copy } = useWebLocale();

  return (
    <div className="thinking-indicator">
      <span className="thinking-text thinking-sweep-text">{copy.chat.thinkingPanelTitle}</span>
      <ThinkingElapsed className="thinking-elapsed" startedAtMs={startedAtMs} />
    </div>
  );
};
