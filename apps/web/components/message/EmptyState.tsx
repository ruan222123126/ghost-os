import type { FC } from 'react';

export const EmptyState: FC = () => (
  <div className="messages is-empty">
    <div className="empty-state">Start by configuring a model, then send a task to Ghost-OS.</div>
  </div>
);
