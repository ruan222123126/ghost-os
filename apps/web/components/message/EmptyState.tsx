import type { FC } from 'react';
import { useWebLocale } from '@/lib/i18n/provider';

export const EmptyState: FC = () => {
  const { copy } = useWebLocale();

  return (
    <div className="messages is-empty">
      <div className="empty-state">{copy.chat.emptyState}</div>
    </div>
  );
};
