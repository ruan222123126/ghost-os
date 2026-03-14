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

export const SessionList: FC<SessionListProps> = ({ sessions, currentSessionId, loading, error: _error, onSelect, onDelete }) => {
  if (loading) {
    return (
      <div className="session-list">
        {Array.from({ length: 4 }).map((_, index) => (
          <div key={`session-skeleton-${index}`} className="session-skeleton" />
        ))}
      </div>
    );
  }

  if (sessions.length === 0) {
    return <div className="empty-state">No sessions yet. Start a new chat to create the first session.</div>;
  }

  return (
    <div className="session-list">
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
