// SessionList component used by the web console chat/session interface.

'use client';

import type { FC } from 'react';
import { SessionItem } from '@/components/SessionItem';
import type { SessionMetadata } from '@/lib/types';

interface SessionListProps {
  sessions: SessionMetadata[];
  currentSessionId: string;
  loading: boolean;
  error: string;
  onSelect: (id: string) => void;
  onDelete: (id: string) => void;
}

export const SessionList: FC<SessionListProps> = ({ sessions, currentSessionId, loading, error, onSelect, onDelete }) => {
  if (loading) {
    return (
      <div className="space-y-2">
        {Array.from({ length: 4 }).map((_, index) => (
          <div
            key={`session-skeleton-${index}`}
            className="h-[58px] animate-pulse rounded-xl border border-app-border/70 bg-[#0b1220]"
          />
        ))}
      </div>
    );
  }

  if (sessions.length === 0) {
    return <p className="rounded-xl border border-dashed border-app-border/70 p-3 text-sm text-app-muted">No sessions yet.</p>;
  }

  return (
    <div className="space-y-2">
      {error && <p className="rounded-lg border border-rose-300/30 bg-rose-300/10 px-3 py-2 text-xs text-rose-200">{error}</p>}
      {sessions.map((session) => (
        <SessionItem
          key={session.id}
          session={session}
          isActive={session.id === currentSessionId}
          onSelect={onSelect}
          onDelete={onDelete}
        />
      ))}
    </div>
  );
};
