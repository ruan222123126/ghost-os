import type { FC } from 'react';
import { useWebLocale } from '@/lib/i18n/provider';

export const ThinkingIndicator: FC = () => {
  const { copy } = useWebLocale();

  return (
    <div className="thinking-indicator">
      <span className="thinking-text thinking-sweep-text">{copy.chat.thinkingPanelTitle}</span>
    </div>
  );
};
