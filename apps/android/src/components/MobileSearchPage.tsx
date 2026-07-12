import { useEffect, useMemo, useRef, useState } from "react";
import type { RefObject } from "react";
import { Loader2 } from "lucide-react";
import type { SidebarHistoryItem } from "./MobileChatHome";
import type { SessionMetadata } from "../mobileTypes";
import { errorMessage } from "../lib/bridgeBus";
import { compareHistoryItems } from "./mobileChat/historyItems";
import { UiIcon } from "./mobileChat/icons";
import "./MobileSearchPage.css";

const SEARCH_DELAY_MS = 500;
const SEARCH_RESULT_VISIBLE_BATCH = 30;
const SEARCH_RESULT_LOAD_MORE_THRESHOLD_PX = 180;

interface MobileSearchPageProps {
  open: boolean;
  bridgeConnected: boolean;
  historyItems: SidebarHistoryItem[];
  onClose: () => void;
  onSearchSessions: (query: string) => Promise<SessionMetadata[]>;
  onSelectHistory: (sessionId: string) => void;
}

export function MobileSearchPage(props: MobileSearchPageProps) {
  const { bridgeConnected, historyItems, onClose, onSearchSessions, onSelectHistory, open } = props;
  const [searchText, setSearchText] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [searchError, setSearchError] = useState("");
  const searchRunIdRef = useRef(0);
  const searchBodyRef = useRef<HTMLDivElement | null>(null);
  const normalizedSearchText = searchText.trim().toLocaleLowerCase("zh-CN");
  const sortedHistoryItems = useMemo(
    () => [...historyItems].sort(compareHistoryItems),
    [historyItems],
  );
  const historyById = useMemo(
    () => new Map(historyItems.map((item) => [item.id, item])),
    [historyItems],
  );
  const [results, setResults] = useState(sortedHistoryItems);
  const resultsResetKey = useMemo(
    () => `${open ? "open" : "closed"}:${normalizedSearchText}:${results.length}:${results[0]?.id ?? ""}`,
    [normalizedSearchText, open, results],
  );
  const [visibleResultCount, handleResultsScroll] = useVisibleSearchResultCount({
    open,
    resetKey: resultsResetKey,
    resultsCount: results.length,
    scrollElementRef: searchBodyRef,
  });
  const visibleResults = useMemo(
    () => results.slice(0, visibleResultCount),
    [results, visibleResultCount],
  );

  useEffect(() => {
    if (!open) {
      searchRunIdRef.current += 1;
      setSearchText("");
      setIsLoading(false);
      setSearchError("");
      setResults(sortedHistoryItems);
      return undefined;
    }

    if (!normalizedSearchText) {
      searchRunIdRef.current += 1;
      setResults(sortedHistoryItems);
      setIsLoading(false);
      setSearchError("");
      return undefined;
    }

    if (!bridgeConnected) {
      searchRunIdRef.current += 1;
      setResults([]);
      setIsLoading(false);
      setSearchError("需要先连接电脑端");
      return undefined;
    }

    const query = searchText.trim();
    const searchRunId = searchRunIdRef.current + 1;
    searchRunIdRef.current = searchRunId;
    setIsLoading(true);
    setSearchError("");
    const timeout = window.setTimeout(() => {
      void onSearchSessions(query)
        .then((metadata) => {
          if (searchRunIdRef.current !== searchRunId) {
            return;
          }
          setResults(metadata.map((item) => sessionMetadataToHistoryItem(item, historyById)));
          setIsLoading(false);
        })
        .catch((error: unknown) => {
          if (searchRunIdRef.current !== searchRunId) {
            return;
          }
          setResults([]);
          setSearchError(`搜索失败：${errorMessage(error)}`);
          setIsLoading(false);
        });
    }, SEARCH_DELAY_MS);

    return () => {
      window.clearTimeout(timeout);
      if (searchRunIdRef.current === searchRunId) {
        searchRunIdRef.current += 1;
      }
    };
  }, [bridgeConnected, historyById, normalizedSearchText, onSearchSessions, open, searchText, sortedHistoryItems]);

  if (!open) {
    return null;
  }

  function closeOrClear(): void {
    if (searchText.trim()) {
      setSearchText("");
      return;
    }
    onClose();
  }

  function selectHistory(sessionId: string): void {
    onSelectHistory(sessionId);
    onClose();
  }

  return (
    <section className="mobile-search-page" role="dialog" aria-modal="true" aria-label="搜索对话">
      <header className="mobile-search-header">
        <UiIcon name="search" />
        <input
          type="search"
          value={searchText}
          placeholder="搜索对话"
          autoFocus
          onChange={(event) => setSearchText(event.target.value)}
        />
        <button
          className="mobile-search-close"
          type="button"
          aria-label={searchText.trim() ? "清空搜索" : "关闭搜索"}
          title={searchText.trim() ? "清空搜索" : "关闭搜索"}
          onClick={closeOrClear}
        >
          <UiIcon name="x" />
        </button>
      </header>

      <div className="mobile-search-body" ref={searchBodyRef} onScroll={handleResultsScroll}>
        {isLoading ? (
          <div className="mobile-search-loading" role="status" aria-label="正在搜索">
            <Loader2 aria-hidden="true" />
          </div>
        ) : (
          <div className="mobile-search-results">
            {!normalizedSearchText ? <h2>近期对话</h2> : null}
            {results.length > 0 ? (
              <ul>
                {visibleResults.map((item) => (
                  <li key={item.id}>
                    <button type="button" onClick={() => selectHistory(item.id)}>
                      <span>{item.title}</span>
                      <time dateTime={item.updatedAt}>{formatHistoryDate(item.updatedAt)}</time>
                    </button>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="mobile-search-empty">
                {searchError || (normalizedSearchText ? "没有找到相关对话" : "暂无可搜索对话")}
              </p>
            )}
          </div>
        )}
      </div>
    </section>
  );
}

function useVisibleSearchResultCount(options: {
  open: boolean;
  resetKey: string;
  resultsCount: number;
  scrollElementRef: RefObject<HTMLDivElement | null>;
}): [number, () => void] {
  const { open, resetKey, resultsCount, scrollElementRef } = options;
  const [visibleCount, setVisibleCount] = useState(() => resolveInitialSearchResultCount(resultsCount));
  const scrollFrameRef = useRef<number | null>(null);

  useEffect(() => {
    setVisibleCount(resolveInitialSearchResultCount(resultsCount));
    const scrollElement = scrollElementRef.current;
    if (scrollElement) {
      scrollElement.scrollTop = 0;
    }
  }, [resetKey, resultsCount, scrollElementRef]);

  useEffect(() => {
    return () => {
      if (scrollFrameRef.current !== null) {
        window.cancelAnimationFrame(scrollFrameRef.current);
      }
    };
  }, []);

  function handleScroll(): void {
    if (!open || scrollFrameRef.current !== null) {
      return;
    }

    scrollFrameRef.current = window.requestAnimationFrame(() => {
      scrollFrameRef.current = null;
      const scrollElement = scrollElementRef.current;
      if (!shouldLoadMoreSearchResults(scrollElement, visibleCount, resultsCount)) {
        return;
      }
      setVisibleCount((current) => resolveNextSearchResultCount(current, resultsCount));
    });
  }

  return [visibleCount, handleScroll];
}

function shouldLoadMoreSearchResults(
  scrollElement: HTMLDivElement | null,
  visibleCount: number,
  resultsCount: number,
): boolean {
  if (
    !scrollElement
    || visibleCount >= resultsCount
    || scrollElement.clientHeight <= 0
    || scrollElement.scrollHeight <= 0
  ) {
    return false;
  }

  return scrollElement.scrollHeight - scrollElement.scrollTop - scrollElement.clientHeight
    <= SEARCH_RESULT_LOAD_MORE_THRESHOLD_PX;
}

function resolveInitialSearchResultCount(resultsCount: number): number {
  if (resultsCount <= 0) {
    return 0;
  }
  return Math.min(resultsCount, SEARCH_RESULT_VISIBLE_BATCH);
}

function resolveNextSearchResultCount(current: number, resultsCount: number): number {
  if (current >= resultsCount) {
    return current;
  }
  return Math.min(resultsCount, current + SEARCH_RESULT_VISIBLE_BATCH);
}

function sessionMetadataToHistoryItem(
  metadata: SessionMetadata,
  historyById: Map<string, SidebarHistoryItem>,
): SidebarHistoryItem {
  const existing = historyById.get(metadata.id);
  return {
    id: metadata.id,
    title: metadata.title.trim() || existing?.title || fallbackSessionTitle(metadata.id),
    updatedAt: metadata.updated_at,
    pinned: existing?.pinned ?? false,
    status: existing?.status,
    unread: existing?.unread,
  };
}

function fallbackSessionTitle(sessionId: string): string {
  const shortId = sessionId.trim().slice(0, 8);
  return shortId ? `会话 ${shortId}` : "新会话";
}

function formatHistoryDate(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }

  const today = startOfDay(new Date());
  const target = startOfDay(date);
  const dayOffset = Math.round((today.getTime() - target.getTime()) / 86_400_000);

  if (dayOffset === 0) {
    return "今天";
  }
  if (dayOffset === 1) {
    return "昨天";
  }
  return new Intl.DateTimeFormat("zh-CN", { day: "numeric", month: "numeric" }).format(date);
}

function startOfDay(date: Date): Date {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate());
}
