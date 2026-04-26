// SessionSidebar component used by the web console chat/session interface.

'use client';

import type { FC } from 'react';
import { SessionSidebarHistory } from '@/components/SessionSidebarHistory';
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
  const sidebarState = useSessionSidebarState();

  return (
    <aside
      className={`sidebar flex flex-col bg-neutral-50 text-black transition-all duration-300 ease-[cubic-bezier(0.2,0,0,1)] ${
        sidebarState.isOpen ? 'w-[19rem]' : 'w-16'
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

      <SessionSidebarHistory
        isOpen={sidebarState.isOpen}
        sessions={sessions}
        searchQuery={sidebarState.searchQuery}
        currentSessionId={currentSessionId}
        loading={loading}
        error={error}
        onSelect={onSelect}
        onDelete={onDelete}
      />

      <SidebarSettingsButton collapsed={!sidebarState.isOpen} onClick={onOpenSettings} />
    </aside>
  );
};
