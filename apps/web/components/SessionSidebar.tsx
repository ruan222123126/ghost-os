// SessionSidebar component used by the web console chat/session interface.

'use client';

import type { FC } from 'react';
import { SidebarSettingsButton } from '@/components/SidebarSettingsButton';
import {
  IconPanelLeftClose,
  IconPanelLeftOpen,
  IconPlus,
  IconSearch,
  IconX,
} from '@/components/sessionSidebarIcons';
import { useSessionSidebarState } from '@/hooks/useSessionSidebarState';
import { useWebLocale } from '@/lib/i18n/provider';
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
  const { copy } = useWebLocale();
  const sidebarState = useSessionSidebarState({ sessions });

  return (
    <aside
      className={`sidebar flex flex-col bg-neutral-50 text-black transition-all duration-300 ease-[cubic-bezier(0.2,0,0,1)] ${
        sidebarState.isOpen ? 'w-72' : 'w-16'
      }`}
    >
      <div className={`flex h-16 items-center p-4 ${sidebarState.isOpen ? 'justify-between' : 'justify-center'}`}>
        <button
          type="button"
          onClick={sidebarState.toggleSidebar}
          className="flex items-center justify-center p-2 transition-colors hover:bg-white"
          title={sidebarState.isOpen ? copy.chat.sidebarCollapseTitle : copy.chat.sidebarExpandTitle}
          aria-label={sidebarState.isOpen ? copy.chat.sidebarCollapseAria : copy.chat.sidebarExpandAria}
        >
          {sidebarState.isOpen ? <IconPanelLeftClose /> : <IconPanelLeftOpen />}
        </button>

        {sidebarState.isOpen ? (
          <button
            type="button"
            onClick={sidebarState.toggleSearch}
            className={`p-2 transition-colors ${sidebarState.isSearchVisible ? 'bg-black text-white' : 'hover:bg-white'}`}
            title={copy.chat.sidebarSearchTitle}
            aria-label={copy.chat.sidebarSearchAria}
          >
            <IconSearch />
          </button>
        ) : null}
      </div>

      <div
        className={`px-3 overflow-hidden transition-all duration-300 ${
          sidebarState.isSearchVisible && sidebarState.isOpen ? 'mb-2 h-12 opacity-100' : 'mb-0 h-0 opacity-0'
        }`}
      >
        <div className="relative border-b border-black/10">
          <input
            ref={sidebarState.searchInputRef}
            type="text"
            placeholder={copy.chat.sidebarSearchPlaceholder}
            value={sidebarState.searchQuery}
            onChange={(event) => sidebarState.setSearchQuery(event.target.value)}
            className="w-full bg-transparent py-2 pl-1 pr-8 text-xs font-medium uppercase tracking-widest outline-none placeholder:text-neutral-300"
          />
          <button
            type="button"
            onClick={sidebarState.clearSearch}
            className="absolute right-0 top-1/2 -translate-y-1/2 text-black"
            aria-label={copy.chat.sidebarClearSearchAria}
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
            sidebarState.openSidebar();
          }}
          className={`group flex items-center justify-center gap-2 bg-transparent text-black transition-colors hover:bg-white ${
            sidebarState.isOpen ? 'w-full px-4 py-3' : 'mx-auto h-10 w-10'
          }`}
        >
          <IconPlus />
          {sidebarState.isOpen ? (
            <span className="text-xs font-bold uppercase tracking-tighter text-neutral-500 group-hover:text-black">{copy.chat.sidebarNewChat}</span>
          ) : null}
        </button>
      </div>

      <div className="mt-6 flex-1 overflow-y-auto px-3">
        {sidebarState.isOpen ? (
          <>
            <div className="mb-4 flex items-center gap-2 border-b border-black/5 px-1 pb-1">
              <span className="text-[10px] font-black uppercase tracking-[0.2em]">{copy.chat.sidebarHistory}</span>
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
            ) : sidebarState.filteredSessions.length === 0 ? (
              <div className="border border-black/10 bg-white px-3 py-3 text-xs text-neutral-600">{copy.chat.sidebarNoSessions}</div>
            ) : (
              <div className="space-y-0.5">
                {sidebarState.filteredSessions.map((session) => {
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
                      <span className="flex-1 truncate uppercase tracking-tight">{copy.chat.sidebarSessionTitle(shortID)}</span>
                      <button
                        type="button"
                        onClick={(event) => {
                          event.stopPropagation();
                          onDelete(session.id);
                        }}
                        className="opacity-0 transition-opacity group-hover:opacity-100"
                        aria-label={copy.chat.sidebarDeleteSessionAria(shortID)}
                        title={copy.chat.sessionDelete}
                      >
                        <span className="text-[10px] font-black uppercase tracking-widest text-black">{copy.chat.sidebarDeleteShort}</span>
                      </button>
                    </div>
                  );
                })}
              </div>
            )}
          </>
        ) : null}
      </div>

      <SidebarSettingsButton collapsed={!sidebarState.isOpen} onClick={onOpenSettings} />
    </aside>
  );
};
