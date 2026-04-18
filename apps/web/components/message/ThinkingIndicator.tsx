import type { FC } from 'react';
import { useWebLocale } from '@/lib/i18n/provider';

export const ThinkingIndicator: FC = () => {
  const { copy } = useWebLocale();

  return (
    <div className="thinking-indicator">
      <div className="thinking-dots" aria-hidden="true">
        <span className="thinking-dot" />
        <span className="thinking-dot" />
        <span className="thinking-dot" />
      </div>
      <span className="thinking-text">{copy.chat.thinkingText}</span>
    </div>
  );
};
