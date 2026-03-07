// SessionItem component used by the web console chat/session interface.

'use client';

import type { FC, MouseEvent } from 'react';
import type { SessionMetadata } from '@/lib/types';

interface SessionItemProps {
  session: SessionMetadata;
  isActive: boolean;
  onSelect: (id: string) => void;
  onDelete: (id: string) => void;
}

function formatRelativeTime(value: string): string {
  const timestamp = Date.parse(value);
  if (Number.isNaN(timestamp)) {
    return value;
  }

  const diffSeconds = Math.max(0, Math.floor((Date.now() - timestamp) / 1000));
  if (diffSeconds < 60) {
    return 'just now';
  }
  if (diffSeconds < 3600) {
    return `${Math.floor(diffSeconds / 60)}m ago`;
  }
  if (diffSeconds < 86400) {
    return `${Math.floor(diffSeconds / 3600)}h ago`;
  }
  if (diffSeconds < 604800) {
    return `${Math.floor(diffSeconds / 86400)}d ago`;
  }
  return new Date(timestamp).toLocaleDateString();
}

export const SessionItem: FC<SessionItemProps> = ({ session, isActive, onSelect, onDelete }) => {
  const shortID = session.id.slice(0, 8);

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={() => onSelect(session.id)}
      onKeyDown={(event) => {
        if (event.key === 'Enter' || event.key === ' ') {
          event.preventDefault();
          onSelect(session.id);
        }
      }}
      className={`group ui-panel-soft w-full cursor-pointer px-3 py-2 text-left outline-none transition focus-visible:ring-2 focus-visible:ring-app-ring/20 ${
        isActive
          ? 'border-app-accent/70 bg-app-accent/10 shadow-lift'
          : 'hover:border-app-fieldBorderHover/80 hover:shadow-lift focus-visible:border-app-fieldBorderHover/80'
      }`}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="mono truncate text-sm text-app-text">{shortID}</p>
          <p className="mt-1 text-xs text-app-muted">{formatRelativeTime(session.created_at)}</p>
        </div>
        <button
          type="button"
          onClick={(event: MouseEvent<HTMLButtonElement>) => {
            event.stopPropagation();
            onDelete(session.id);
          }}
          className="ui-btn-secondary border-rose-400/30 px-2 py-1 text-xs text-rose-200 opacity-0 hover:border-rose-300/60 hover:text-rose-100 group-hover:opacity-100 group-focus-within:opacity-100"
          aria-label={`Delete session ${shortID}`}
        >
          Delete
        </button>
      </div>
    </div>
  );
};
