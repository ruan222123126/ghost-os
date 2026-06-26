// SessionSidebar component used by the web console chat/session interface.

'use client';

import { useCallback, useMemo, useRef, useState, type FC } from 'react';
import { SessionSearchDialog } from '@/components/SessionSearchDialog';
import { SessionSidebarHistory } from '@/components/SessionSidebarHistory';
import { SidebarSettingsButton } from '@/components/SidebarSettingsButton';
import {
  IconPanelLeftClose,
  IconPanelLeftOpen,
  IconPlus,
  IconSearch,
} from '@/components/sessionSidebarIcons';
import type { UseSessionSidebarAliasesResult } from '@/hooks/useSessionSidebarAliases';
import { useSessionSidebarPartitions } from '@/hooks/useSessionSidebarPartitions';
import { useSessionSidebarSessionSources } from '@/hooks/useSessionSidebarSessionSources';
import { useSessionSidebarState } from '@/hooks/useSessionSidebarState';
import { useWebLocale } from '@/lib/i18n/provider';
import { mergeSessionSourcePartitionViews } from '@/lib/sessionSidebarSessionSources';
import type { SessionMetadata } from '@/lib/types';

interface SessionSidebarProps {
  sessions: SessionMetadata[];
  currentSessionId: string;
  backgroundCompletedSessionIds: ReadonlySet<string>;
  loading: boolean;
  error: string;
  onSelect: (id: string) => void;
  onDelete: (id: string) => void;
  onNewChat: () => void;
  onOpenSettings: () => void;
  resolveSessionTitle: UseSessionSidebarAliasesResult['resolveSessionTitle'];
  renameSession: UseSessionSidebarAliasesResult['renameSession'];
}

export const SessionSidebar: FC<SessionSidebarProps> = ({
  sessions,
  currentSessionId,
  backgroundCompletedSessionIds,
  loading,
  error,
  onSelect,
  onDelete,
  onNewChat,
  onOpenSettings,
  resolveSessionTitle,
  renameSession,
}) => {
  const { copy } = useWebLocale();
  const sidebarState = useSessionSidebarState();
  const searchButtonRef = useRef<HTMLButtonElement | null>(null);
  const [focusSessionId, setFocusSessionId] = useState('');
  const activeSearchQuery = sidebarState.isSearchVisible ? sidebarState.searchQuery : '';
  const showBlockingLoading = loading && sessions.length === 0;
  const openSidebar = sidebarState.openSidebar;
  const matchesSessionSearch = useCallback((session: SessionMetadata, normalizedQuery: string) => {
    if (!normalizedQuery) {
      return true;
    }
    return [
      session.id,
      session.title,
      resolveSessionTitle(session),
    ].some((value) => value.trim().toLowerCase().includes(normalizedQuery));
  }, [resolveSessionTitle]);
  const partitionModel = useSessionSidebarPartitions({
    sessions,
    sessionsLoaded: !showBlockingLoading,
    searchQuery: activeSearchQuery,
    unclassifiedName: copy.chat.sidebarPartitionUnclassified,
    requestFailedText: copy.system.genericRequestFailed,
    matchesSearch: matchesSessionSearch,
  });
  const sessionSources = useSessionSidebarSessionSources({
    sessions,
    requestFailedText: copy.system.genericRequestFailed,
  });
  const visiblePartitionViews = useMemo(() => {
    return mergeSessionSourcePartitionViews({
      manualViews: partitionModel.partitionViews,
      sessions,
      sourceAssignments: sessionSources.assignments,
      hiddenSessionIDs: sessionSources.hiddenSessionIDs,
      searchQuery: activeSearchQuery,
      copy: copy.chat,
      matchesSearch: matchesSessionSearch,
    });
  }, [
    activeSearchQuery,
    copy.chat,
    matchesSessionSearch,
    partitionModel.partitionViews,
    sessionSources.assignments,
    sessionSources.hiddenSessionIDs,
    sessions,
  ]);
  const createNewChat = useCallback(() => {
    setFocusSessionId('');
    onNewChat();
    openSidebar();
  }, [onNewChat, openSidebar]);
  const selectSidebarSession = useCallback((id: string) => {
    setFocusSessionId('');
    onSelect(id);
  }, [onSelect]);
  const selectSearchResult = useCallback((id: string) => {
    setFocusSessionId(id);
    onSelect(id);
  }, [onSelect]);
  const iconButtonClassName = 'h-10 w-10 shrink-0 p-0';

  return (
    <aside
      className={`sidebar flex flex-col bg-neutral-50 text-black transition-all duration-300 ease-[cubic-bezier(0.2,0,0,1)] ${
        sidebarState.isOpen ? 'w-[19rem]' : 'w-16'
      }`}
    >
      <div className={`flex h-16 items-center ${sidebarState.isOpen ? 'justify-between p-4' : 'justify-center p-3'}`}>
        <button
          type="button"
          onClick={sidebarState.toggleSidebar}
          className={`flex items-center justify-center transition-colors hover:bg-white ${
            sidebarState.isOpen ? 'p-2' : iconButtonClassName
          }`}
          title={sidebarState.isOpen ? copy.chat.sidebarCollapseTitle : copy.chat.sidebarExpandTitle}
          aria-label={sidebarState.isOpen ? copy.chat.sidebarCollapseAria : copy.chat.sidebarExpandAria}
        >
          {sidebarState.isOpen ? <IconPanelLeftClose /> : <IconPanelLeftOpen />}
        </button>

        {sidebarState.isOpen ? (
          <button
            ref={searchButtonRef}
            type="button"
            onClick={sidebarState.toggleSearch}
            className="p-2 text-black transition-colors hover:bg-white focus-visible:bg-white focus-visible:outline-none"
            title={copy.chat.sidebarSearchTitle}
            aria-label={copy.chat.sidebarSearchAria}
            aria-expanded={sidebarState.isSearchVisible}
          >
            <IconSearch />
          </button>
        ) : null}
      </div>

      <div className="mt-2 flex flex-col gap-2 px-3">
        <button
          type="button"
          onClick={createNewChat}
          className={`group flex items-center justify-center gap-2 overflow-hidden bg-transparent text-black transition-colors hover:bg-white ${
            sidebarState.isOpen ? 'w-full px-4 py-3' : `mx-auto ${iconButtonClassName}`
          }`}
        >
          <span className="grid flex-shrink-0 place-items-center">
            <IconPlus />
          </span>
          <span
            className={`whitespace-nowrap text-xs font-bold uppercase tracking-tighter text-neutral-500 transition-all duration-300 group-hover:text-black ${
              sidebarState.isOpen ? 'max-w-[100px] opacity-100' : 'max-w-0 opacity-0'
            }`}
          >
            {copy.chat.sidebarNewChat}
          </span>
        </button>
      </div>

      <SessionSidebarHistory
        isOpen={sidebarState.isOpen}
        sessions={sessions}
        searchQuery={activeSearchQuery}
        currentSessionId={currentSessionId}
        backgroundCompletedSessionIds={backgroundCompletedSessionIds}
        focusSessionId={focusSessionId}
        loading={loading}
        error={error}
        onSelect={selectSidebarSession}
        onDelete={onDelete}
        resolveSessionTitle={resolveSessionTitle}
        renameSession={renameSession}
        partitionModel={partitionModel}
        visiblePartitionViews={visiblePartitionViews}
        sessionSourcesError={sessionSources.error}
      />

      <SidebarSettingsButton collapsed={!sidebarState.isOpen} onClick={onOpenSettings} />

      <SessionSearchDialog
        open={sidebarState.isSearchVisible}
        query={sidebarState.searchQuery}
        inputRef={sidebarState.searchInputRef}
        triggerRef={searchButtonRef}
        sessions={sessions}
        partitionViews={visiblePartitionViews}
        currentSessionId={currentSessionId}
        resolveSessionTitle={resolveSessionTitle}
        onQueryChange={sidebarState.setSearchQuery}
        onClose={sidebarState.closeSearch}
        onSelect={selectSearchResult}
      />
    </aside>
  );
};
