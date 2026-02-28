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
      className={`group w-full rounded-xl border px-3 py-2 text-left transition ${
        isActive
          ? 'border-app-accent/70 bg-app-accent/10'
          : 'border-app-border/70 bg-[#0b1220] hover:border-app-accent/35'
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
          className="rounded-md border border-transparent px-2 py-1 text-xs text-app-muted opacity-0 transition hover:border-rose-300/40 hover:text-rose-200 group-hover:opacity-100"
          aria-label={`Delete session ${shortID}`}
        >
          Delete
        </button>
      </div>
    </div>
  );
};
