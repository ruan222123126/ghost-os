'use client';

import {
  type CSSProperties,
  type FC,
  type RefObject,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { IconChevronRight, IconSearch } from '@/components/sessionSidebarIcons';
import { useSessionSearch } from '@/hooks/useSessionSearch';
import { useWebLocale } from '@/lib/i18n/provider';
import { compareSessionsByRecentActivity } from '@/lib/sessionSidebarSessionSort';
import type { ChatCopy } from '@/lib/i18n/messages/chat';
import type { SessionMetadata } from '@/lib/types';
import type { SessionPartitionView } from '@/lib/sessionSidebarPartitions';

const SEARCH_DIALOG_ANIMATION_MS = 300;
const SEARCH_DIALOG_CLOSED_SCALE = 0.18;
const SEARCH_DIALOG_OPEN_TRANSFORM = 'translate3d(0, 0, 0) scale(1)';
const SEARCH_DIALOG_DEFAULT_MOTION_TARGET: SearchDialogMotionTarget = {
  closedTransform: `translate3d(0, -56px, 0) scale(${SEARCH_DIALOG_CLOSED_SCALE})`,
  transformOrigin: 'center center',
};

interface SessionSearchDialogProps {
  open: boolean;
  query: string;
  inputRef: RefObject<HTMLInputElement>;
  triggerRef: RefObject<HTMLElement>;
  sessions: SessionMetadata[];
  partitionViews?: SessionPartitionView[];
  currentSessionId: string;
  resolveSessionTitle: (session: SessionMetadata) => string;
  onQueryChange: (value: string) => void;
  onClose: () => void;
  onSelect: (id: string) => void;
}

export const SessionSearchDialog: FC<SessionSearchDialogProps> = ({
  open,
  query,
  inputRef,
  triggerRef,
  sessions,
  partitionViews,
  currentSessionId,
  resolveSessionTitle,
  onQueryChange,
  onClose,
  onSelect,
}) => {
  const { copy, locale } = useWebLocale();
  const dialogRef = useRef<HTMLDivElement | null>(null);
  const [mounted, setMounted] = useState(open);
  const [expanded, setExpanded] = useState(false);
  const [motionTarget, setMotionTarget] = useState<SearchDialogMotionTarget>(SEARCH_DIALOG_DEFAULT_MOTION_TARGET);
  const sessionSearch = useSessionSearch({
    open,
    query,
    fallbackMessage: copy.system.genericRequestFailed,
  });
  const localResults = useMemo(() => {
    return buildSessionSearchResults({
      sessions,
      partitionViews,
      query,
      resolveSessionTitle,
    });
  }, [partitionViews, query, resolveSessionTitle, sessions]);
  const results = useMemo(() => {
    return mergeSessionSearchResults(sessionSearch.results, localResults);
  }, [localResults, sessionSearch.results]);

  useLayoutEffect(() => {
    if (!mounted || !open) {
      return;
    }

    setMotionTarget(buildSearchDialogMotionTarget({
      dialogRect: readElementLayoutRect(dialogRef.current),
      triggerRect: triggerRef.current?.getBoundingClientRect(),
    }));
  }, [mounted, open, triggerRef]);

  useEffect(() => {
    if (open) {
      setMounted(true);
      const frameID = window.requestAnimationFrame(() => {
        setExpanded(true);
      });
      return () => window.cancelAnimationFrame(frameID);
    }

    setExpanded(false);
    const timerID = window.setTimeout(() => {
      setMounted(false);
    }, SEARCH_DIALOG_ANIMATION_MS);
    return () => window.clearTimeout(timerID);
  }, [open]);

  useEffect(() => {
    if (!open || !mounted) {
      return;
    }
    const timerID = window.setTimeout(() => {
      inputRef.current?.focus();
    }, 0);
    return () => window.clearTimeout(timerID);
  }, [inputRef, mounted, open]);

  useEffect(() => {
    if (!open) {
      return;
    }
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose();
      }
    };
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [onClose, open]);

  if (!mounted) {
    return null;
  }

  return (
    <div
      className={`fixed inset-0 z-50 flex items-start justify-center bg-gray-300/45 px-4 pt-24 transition-opacity duration-300 ease-in-out ${
        expanded ? 'pointer-events-auto opacity-100' : 'pointer-events-none opacity-0'
      }`}
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) {
          onClose();
        }
      }}
    >
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-label={copy.chat.sidebarSearchAria}
        className={`w-full max-w-[640px] overflow-hidden rounded-2xl bg-white shadow-xl transition-all duration-300 ease-in-out ${
          expanded
            ? 'opacity-100'
            : 'pointer-events-none opacity-0'
        }`}
        style={buildSearchDialogMotionStyle({ expanded, target: motionTarget })}
        onMouseDown={(event) => event.stopPropagation()}
      >
        <div className="flex h-[64px] items-center px-4">
          <span className="ml-2 mr-3 flex h-5 w-5 flex-shrink-0 items-center justify-center text-gray-500">
            <IconSearch />
          </span>
          <input
            ref={inputRef}
            type="text"
            className="h-full w-full bg-transparent text-[15px] text-gray-800 outline-none placeholder:text-[#9ca3af]"
            placeholder={copy.chat.sidebarSearchDialogPlaceholder}
            value={query}
            onChange={(event) => onQueryChange(event.target.value)}
            autoComplete="off"
          />
        </div>

        <div className="h-px w-full bg-[#f3f4f6]" />

        <div className="max-h-[min(60vh,520px)] space-y-1 overflow-y-auto p-2 pb-3 pt-2">
          {sessionSearch.error ? (
            <div className="mx-1 px-5 py-[10px] text-[13px] text-red-600">
              {copy.chat.sidebarSearchFailed}: {sessionSearch.error}
            </div>
          ) : null}
          {results.length > 0 ? (
            results.map((session) => (
              <SearchResultRow
                key={session.id}
                copy={copy.chat}
                locale={locale}
                session={session}
                title={resolveSessionTitle(session)}
                active={session.id === currentSessionId}
                onSelect={() => {
                  onSelect(session.id);
                  onClose();
                }}
              />
            ))
          ) : sessionSearch.loading ? (
            <div className="mx-1 px-5 py-[12px] text-[14px] text-[#9ca3af]">
              {copy.chat.sidebarSearchLoading}
            </div>
          ) : (
            <div className="mx-1 px-5 py-[12px] text-[14px] text-[#9ca3af]">
              {copy.chat.sidebarNoSessions}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

interface SearchDialogRect {
  left: number;
  top: number;
  width: number;
  height: number;
}

interface SearchDialogMotionTarget {
  closedTransform: string;
  transformOrigin: string;
}

export function buildSearchDialogMotionTarget(input: {
  dialogRect?: SearchDialogRect | null;
  triggerRect?: SearchDialogRect | null;
}): SearchDialogMotionTarget {
  if (!input.dialogRect || !input.triggerRect) {
    return SEARCH_DIALOG_DEFAULT_MOTION_TARGET;
  }

  const dialogCenterX = input.dialogRect.left + input.dialogRect.width / 2;
  const dialogCenterY = input.dialogRect.top + input.dialogRect.height / 2;
  const triggerCenterX = input.triggerRect.left + input.triggerRect.width / 2;
  const triggerCenterY = input.triggerRect.top + input.triggerRect.height / 2;
  return {
    closedTransform: `translate3d(${triggerCenterX - dialogCenterX}px, ${triggerCenterY - dialogCenterY}px, 0) scale(${SEARCH_DIALOG_CLOSED_SCALE})`,
    transformOrigin: 'center center',
  };
}

function buildSearchDialogMotionStyle(input: {
  expanded: boolean;
  target: SearchDialogMotionTarget;
}): CSSProperties {
  return {
    transform: input.expanded ? SEARCH_DIALOG_OPEN_TRANSFORM : input.target.closedTransform,
    transformOrigin: input.target.transformOrigin,
  };
}

function readElementLayoutRect(element: HTMLElement | null): SearchDialogRect | null {
  if (!element) {
    return null;
  }

  const offsetParentRect = element.offsetParent?.getBoundingClientRect();
  return {
    left: (offsetParentRect?.left ?? 0) + element.offsetLeft,
    top: (offsetParentRect?.top ?? 0) + element.offsetTop,
    width: element.offsetWidth,
    height: element.offsetHeight,
  };
}

export function buildSessionSearchResults(input: {
  sessions: SessionMetadata[];
  partitionViews?: SessionPartitionView[];
  query: string;
  resolveSessionTitle: (session: SessionMetadata) => string;
  limit?: number;
}): SessionMetadata[] {
  const normalizedQuery = input.query.trim().toLowerCase();
  const resultsByID = new Map<string, SessionMetadata>();
  for (const session of input.sessions) {
    if (matchesSessionSearch(session, input.resolveSessionTitle(session), normalizedQuery)) {
      resultsByID.set(session.id, session);
    }
  }
  if (normalizedQuery && input.partitionViews) {
    for (const session of collectPartitionViewSessions(input.partitionViews)) {
      resultsByID.set(session.id, session);
    }
  }
  const results = [...resultsByID.values()].sort(compareSessionsByRecentActivity);
  if (input.limit === undefined) {
    return results;
  }
  return results.slice(0, Math.max(0, input.limit));
}

function collectPartitionViewSessions(partitionViews: SessionPartitionView[]): SessionMetadata[] {
  const sessions: SessionMetadata[] = [];
  for (const partition of partitionViews) {
    sessions.push(...partition.sessions);
    if (partition.childPartitions) {
      sessions.push(...collectPartitionViewSessions(partition.childPartitions));
    }
  }
  return sessions;
}

export function mergeSessionSearchResults(
  primary: SessionMetadata[],
  secondary: SessionMetadata[],
): SessionMetadata[] {
  const seen = new Set<string>();
  const merged: SessionMetadata[] = [];
  for (const session of [...primary, ...secondary]) {
    if (seen.has(session.id)) {
      continue;
    }
    seen.add(session.id);
    merged.push(session);
  }
  return merged;
}

const SearchResultRow: FC<{
  copy: ChatCopy;
  locale: string;
  session: SessionMetadata;
  title: string;
  active: boolean;
  onSelect: () => void;
}> = ({ copy, locale, session, title, active, onSelect }) => (
  <button
    type="button"
    onClick={onSelect}
    className={`mx-1 flex w-[calc(100%-0.5rem)] cursor-pointer items-center justify-between rounded-[14px] px-5 py-[12px] text-left transition-colors ${
      active ? 'bg-[#fdfbea]' : 'hover:bg-gray-50'
    }`}
  >
    <span className={`min-w-0 flex-1 truncate text-[15px] ${active ? 'text-[#785c32]' : 'text-[#1f2937]'}`}>
      {title}
    </span>
    {active ? (
      <span className="ml-4 flex h-[18px] w-[18px] flex-shrink-0 items-center justify-center text-[#a8916d]">
        <IconChevronRight size={18} />
      </span>
    ) : (
      <span className="ml-4 flex-shrink-0 text-[14px] text-[#9ca3af]">
        {formatSearchSessionTime(session, locale, copy)}
      </span>
    )}
  </button>
);

function matchesSessionSearch(
  session: SessionMetadata,
  title: string,
  normalizedQuery: string,
): boolean {
  if (!normalizedQuery) {
    return true;
  }
  return [
    session.id,
    session.title,
    title,
  ].some((value) => value.toLowerCase().includes(normalizedQuery));
}

function formatSearchSessionTime(
  session: SessionMetadata,
  locale: string,
  copy: ChatCopy,
): string {
  const timestamp = Date.parse(session.updated_at || session.created_at);
  if (Number.isNaN(timestamp)) {
    return session.updated_at || session.created_at;
  }

  const diffSeconds = Math.max(0, Math.floor((Date.now() - timestamp) / 1000));
  if (diffSeconds < 60) {
    return copy.sessionRelativeNow;
  }
  if (diffSeconds < 3600) {
    return copy.sessionRelativeMinutes(Math.floor(diffSeconds / 60));
  }
  if (diffSeconds < 86400) {
    return copy.sessionRelativeHours(Math.floor(diffSeconds / 3600));
  }
  if (diffSeconds < 604800) {
    return copy.sessionRelativeDays(Math.floor(diffSeconds / 86400));
  }
  return new Date(timestamp).toLocaleDateString(locale);
}
