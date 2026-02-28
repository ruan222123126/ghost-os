'use client';

import type { FC } from 'react';
import { useState } from 'react';
import { SessionList } from '@/components/SessionList';
import type { SessionMetadata } from '@/lib/types';

interface SessionSidebarProps {
  sessions: SessionMetadata[];
  currentSessionId: string;
  loading: boolean;
  error: string;
  onSelect: (id: string) => void;
  onDelete: (id: string) => void;
  onNewChat: () => void;
}

export const SessionSidebar: FC<SessionSidebarProps> = ({
  sessions,
  currentSessionId,
  loading,
  error,
  onSelect,
  onDelete,
  onNewChat,
}) => {
  const [open, setOpen] = useState(true);

  return (
    <aside className={`shrink-0 transition-all duration-300 ${open ? 'w-[280px]' : 'w-14'}`}>
      <div className="flex h-full min-h-[520px] flex-col rounded-2xl border border-app-border bg-app-panel/85 p-3 shadow-xl backdrop-blur">
        <div className="mb-3 flex items-center justify-between">
          <button
            type="button"
            onClick={() => setOpen((value) => !value)}
            className="rounded-lg border border-app-border px-2 py-1 text-xs text-app-muted transition hover:border-app-accent/45 hover:text-app-text"
          >
            Menu
          </button>

          {open && (
            <button
              type="button"
              onClick={onNewChat}
              className="rounded-lg border border-app-accent/40 bg-app-accent/20 px-3 py-1.5 text-xs font-medium text-app-text transition hover:bg-app-accent/30"
            >
              New Chat
            </button>
          )}
        </div>

        {open && <h2 className="mb-2 text-sm font-semibold tracking-wide text-app-text">Sessions</h2>}

        {open ? (
          <div className="overflow-y-auto">
            <SessionList
              sessions={sessions}
              currentSessionId={currentSessionId}
              loading={loading}
              error={error}
              onSelect={onSelect}
              onDelete={onDelete}
            />
          </div>
        ) : (
          <div className="flex flex-1 items-center justify-center text-xs text-app-muted">Open</div>
        )}
      </div>
    </aside>
  );
};
