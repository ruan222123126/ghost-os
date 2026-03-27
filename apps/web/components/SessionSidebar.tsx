// SessionSidebar component used by the web console chat/session interface.

'use client';

import { useEffect, useMemo, useRef, useState, type FC } from 'react';
import { SidebarSettingsButton } from '@/components/SidebarSettingsButton';
import type { SessionMetadata } from '@/lib/types';

interface SessionSidebarProps {
  sessions: SessionMetadata[];
  currentSessionId: string;
  loading: boolean;
  error: string;
  onSelect: (id: string) => void;
  onDelete: (id: string) => void;
  onNewChat: () => void;
  onOpenSettings: () => void;
}

const IconPanelLeftClose: FC<{ size?: number }> = ({ size = 20 }) => (
  <svg width={size} height={size} viewBox="0 0 20 20" fill="none" aria-hidden="true">
    <path d="M3.5 4.5h13" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
    <path d="M3.5 15.5h13" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
    <path d="M8.2 4.5v11" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
    <path d="M12.7 7.2l-2.5 2.8 2.5 2.8" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
  </svg>
);

const IconPanelLeftOpen: FC<{ size?: number }> = ({ size = 20 }) => (
  <svg width={size} height={size} viewBox="0 0 20 20" fill="none" aria-hidden="true">
    <path d="M3.5 4.5h13" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
    <path d="M3.5 15.5h13" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
    <path d="M8.2 4.5v11" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
    <path d="M10.2 7.2l2.5 2.8-2.5 2.8" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
  </svg>
);

const IconSearch: FC<{ size?: number }> = ({ size = 20 }) => (
  <svg width={size} height={size} viewBox="0 0 20 20" fill="none" aria-hidden="true">
    <path d="M8.9 14.7a5.8 5.8 0 1 1 0-11.6 5.8 5.8 0 0 1 0 11.6Z" stroke="currentColor" strokeWidth="1.5" />
    <path d="M13.3 13.3 17 17" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
  </svg>
);

const IconPlus: FC<{ size?: number }> = ({ size = 20 }) => (
  <svg width={size} height={size} viewBox="0 0 20 20" fill="none" aria-hidden="true">
    <path d="M10 4v12" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
    <path d="M4 10h12" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
  </svg>
);

const IconX: FC<{ size?: number }> = ({ size = 14 }) => (
  <svg width={size} height={size} viewBox="0 0 20 20" fill="none" aria-hidden="true">
    <path d="M5 5l10 10" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" />
    <path d="M15 5 5 15" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" />
  </svg>
);

export const SessionSidebar: FC<SessionSidebarProps> = ({
  sessions,
  currentSessionId,
  loading,
  error,
  onSelect,
  onDelete,
  onNewChat,
  onOpenSettings,
}) => {
  const [isOpen, setIsOpen] = useState(true);
  const [isSearchVisible, setIsSearchVisible] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const searchInputRef = useRef<HTMLInputElement | null>(null);

  const filteredSessions = useMemo(() => {
    const query = searchQuery.trim().toLowerCase();
    if (!query) return sessions;
    return sessions.filter((session) => session.id.toLowerCase().includes(query));
  }, [searchQuery, sessions]);

  const toggleSidebar = () => {
    setIsOpen((open) => !open);
    if (isOpen) {
      setIsSearchVisible(false);
      setSearchQuery('');
    }
  };

  const toggleSearch = () => {
    setIsSearchVisible((visible) => !visible);
  };

  useEffect(() => {
    if (isOpen && isSearchVisible) {
      searchInputRef.current?.focus();
    }
  }, [isOpen, isSearchVisible]);

  return (
    <aside
      className={`sidebar flex flex-col bg-neutral-50 text-black transition-all duration-300 ease-[cubic-bezier(0.2,0,0,1)] ${
        isOpen ? 'w-72' : 'w-16'
      }`}
    >
      <div className={`flex h-16 items-center p-4 ${isOpen ? 'justify-between' : 'justify-center'}`}>
        <button
          type="button"
          onClick={toggleSidebar}
          className="flex items-center justify-center p-2 transition-colors hover:bg-white"
          title={isOpen ? 'Collapse' : 'Expand'}
          aria-label={isOpen ? 'Collapse sidebar' : 'Expand sidebar'}
        >
          {isOpen ? <IconPanelLeftClose /> : <IconPanelLeftOpen />}
        </button>

        {isOpen ? (
          <button
            type="button"
            onClick={toggleSearch}
            className={`p-2 transition-colors ${isSearchVisible ? 'bg-black text-white' : 'hover:bg-white'}`}
            title="Search"
            aria-label="Search sessions"
          >
            <IconSearch />
          </button>
        ) : null}
      </div>

      <div
        className={`px-3 overflow-hidden transition-all duration-300 ${
          isSearchVisible && isOpen ? 'mb-2 h-12 opacity-100' : 'mb-0 h-0 opacity-0'
        }`}
      >
        <div className="relative border-b border-black/10">
          <input
            ref={searchInputRef}
            type="text"
            placeholder="SEARCH..."
            value={searchQuery}
            onChange={(event) => setSearchQuery(event.target.value)}
            className="w-full bg-transparent py-2 pl-1 pr-8 text-xs font-medium uppercase tracking-widest outline-none placeholder:text-neutral-300"
          />
          <button
            type="button"
            onClick={() => {
              setSearchQuery('');
              setIsSearchVisible(false);
            }}
            className="absolute right-0 top-1/2 -translate-y-1/2 text-black"
            aria-label="Clear search"
          >
            <IconX />
          </button>
        </div>
      </div>

      <div className="mt-2 flex flex-col gap-2 px-3">
        <button
          type="button"
          onClick={() => {
            onNewChat();
            if (!isOpen) setIsOpen(true);
          }}
          className={`group flex items-center justify-center gap-2 bg-transparent text-black transition-colors hover:bg-white ${
            isOpen ? 'w-full px-4 py-3' : 'mx-auto h-10 w-10'
          }`}
        >
          <IconPlus />
          {isOpen ? (
            <span className="text-xs font-bold uppercase tracking-tighter text-neutral-500 group-hover:text-black">New Chat</span>
          ) : null}
        </button>
      </div>

      <div className="mt-6 flex-1 overflow-y-auto px-3">
        {isOpen ? (
          <>
            <div className="mb-4 flex items-center gap-2 border-b border-black/5 px-1 pb-1">
              <span className="text-[10px] font-black uppercase tracking-[0.2em]">History</span>
            </div>

            {error && !loading ? (
              <div className="mb-3 border border-black/10 bg-white px-3 py-2 text-xs text-neutral-700">{error}</div>
            ) : null}

            {loading ? (
              <div className="space-y-2">
                {Array.from({ length: 6 }).map((_, index) => (
                  <div key={`session-skeleton-${index}`} className="h-10 w-full bg-white" />
                ))}
              </div>
            ) : filteredSessions.length === 0 ? (
              <div className="border border-black/10 bg-white px-3 py-3 text-xs text-neutral-600">No sessions.</div>
            ) : (
              <div className="space-y-0.5">
                {filteredSessions.map((session) => {
                  const shortID = session.id.slice(0, 8);
                  const isActive = session.id === currentSessionId;

                  return (
                    <div
                      key={session.id}
                      onClick={() => onSelect(session.id)}
                      role="button"
                      tabIndex={0}
                      onKeyDown={(event) => {
                        if (event.key === 'Enter' || event.key === ' ') {
                          event.preventDefault();
                          onSelect(session.id);
                        }
                      }}
                      className={`group relative flex w-full items-center gap-3 px-3 py-3 text-left text-xs transition-colors ${
                        isActive
                          ? 'bg-white font-bold text-black shadow-sm ring-1 ring-black/5'
                          : 'text-neutral-500 hover:bg-white hover:text-black'
                      }`}
                    >
                      <span className="flex-1 truncate uppercase tracking-tight">Session {shortID}</span>
                      <button
                        type="button"
                        onClick={(event) => {
                          event.stopPropagation();
                          onDelete(session.id);
                        }}
                        className="opacity-0 transition-opacity group-hover:opacity-100"
                        aria-label={`Delete session ${shortID}`}
                        title="Delete"
                      >
                        <span className="text-[10px] font-black uppercase tracking-widest text-black">Del</span>
                      </button>
                    </div>
                  );
                })}
              </div>
            )}
          </>
        ) : null}
      </div>

      <SidebarSettingsButton collapsed={!isOpen} onClick={onOpenSettings} />
    </aside>
  );
};
