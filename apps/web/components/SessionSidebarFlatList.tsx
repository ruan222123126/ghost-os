'use client';

import type { FC, KeyboardEvent as ReactKeyboardEvent } from 'react';
import type { ChatCopy } from '@/lib/i18n/messages/chat';
import { compareSessionsByRecentActivity } from '@/lib/sessionSidebarSessionSort';
import type { SessionMetadata } from '@/lib/types';

interface SessionSidebarFlatListProps {
  copy: ChatCopy;
  sessions: SessionMetadata[];
  currentSessionId: string;
  resolveSessionTitle: (session: SessionMetadata) => string;
  onSelect: (id: string) => void;
  onDelete: (id: string) => void;
}

export const SessionSidebarFlatList: FC<SessionSidebarFlatListProps> = ({
  copy,
  sessions,
  currentSessionId,
  resolveSessionTitle,
  onSelect,
  onDelete,
}) => {
  return (
    <div className="space-y-0.5 pb-3">
      {sessions.map((session) => (
        <SessionRow
          key={session.id}
          copy={copy}
          session={session}
          currentSessionId={currentSessionId}
          resolveSessionTitle={resolveSessionTitle}
          onSelect={onSelect}
          onDelete={onDelete}
        />
      ))}
    </div>
  );
};

interface SessionRowProps {
  copy: ChatCopy;
  session: SessionMetadata;
  currentSessionId: string;
  resolveSessionTitle: (session: SessionMetadata) => string;
  onSelect: (id: string) => void;
  onDelete: (id: string) => void;
}

const SessionRow: FC<SessionRowProps> = ({
  copy,
  session,
  currentSessionId,
  resolveSessionTitle,
  onSelect,
  onDelete,
}) => {
  const shortID = session.id.slice(0, 8);
  const sessionTitle = resolveSessionTitle(session);
  const onRowKeyDown = (event: ReactKeyboardEvent<HTMLDivElement>) => {
    if (event.key !== 'Enter' && event.key !== ' ') {
      return;
    }
    event.preventDefault();
    onSelect(session.id);
  };

  return (
    <div
      data-session-item="true"
      data-session-id={session.id}
      role="button"
      tabIndex={0}
      onClick={() => onSelect(session.id)}
      onKeyDown={onRowKeyDown}
      onContextMenu={(event) => {
        event.preventDefault();
      }}
      className={`group relative flex w-full items-center gap-3 px-3 py-3 text-left text-xs transition-colors ${
        session.id === currentSessionId
          ? 'bg-white font-bold text-black shadow-sm ring-1 ring-black/5'
          : 'text-neutral-500 hover:bg-white hover:text-black'
      }`}
    >
      <span className="flex-1 truncate">{sessionTitle}</span>
      <button
        type="button"
        onClick={(event) => {
          event.stopPropagation();
          onDelete(session.id);
        }}
        className="opacity-0 transition-opacity group-hover:opacity-100"
        aria-label={copy.sidebarDeleteSessionAria(shortID)}
        title={copy.sessionDelete}
      >
        <span className="text-[10px] font-black uppercase tracking-widest text-black">{copy.sidebarDeleteShort}</span>
      </button>
    </div>
  );
};

export function buildFlatSessionList(
  sessions: SessionMetadata[],
  searchQuery: string,
): SessionMetadata[] {
  const query = searchQuery.trim().toLowerCase();
  return sessions
    .filter((session) => session.id.toLowerCase().includes(query))
    .sort(compareSessionsByRecentActivity);
}
