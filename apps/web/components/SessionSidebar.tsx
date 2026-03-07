// SessionSidebar component used by the web console chat/session interface.

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
      <div className="ui-panel animate-riseSoft flex h-full min-h-[520px] flex-col p-3">
        <div className="mb-3 flex items-center justify-between">
          <button
            type="button"
            onClick={() => setOpen((value) => !value)}
            className="ui-btn-secondary px-2 py-1 text-xs text-app-muted hover:text-app-text"
          >
            Menu
          </button>

          {open && (
            <button
              type="button"
              onClick={onNewChat}
              className="ui-btn px-3 py-1.5 text-xs"
            >
              New Chat
            </button>
          )}
        </div>

        {open && <h2 className="mb-2 text-sm font-semibold tracking-wide text-app-text">Sessions</h2>}

        {open ? (
          <div className="ui-scroll overflow-y-auto pr-1">
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
          <div className="ui-hint flex flex-1 items-center justify-center">Open</div>
        )}
      </div>
    </aside>
  );
};
