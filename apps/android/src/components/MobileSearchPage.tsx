import { useEffect, useMemo, useState } from "react";
import { Loader2 } from "lucide-react";
import type { SidebarHistoryItem } from "./MobileChatHome";
import { UiIcon } from "./mobileChat/icons";
import "./MobileSearchPage.css";

const SEARCH_DELAY_MS = 500;

interface MobileSearchPageProps {
  open: boolean;
  historyItems: SidebarHistoryItem[];
  onClose: () => void;
  onSelectHistory: (sessionId: string) => void;
}

export function MobileSearchPage(props: MobileSearchPageProps) {
  const [searchText, setSearchText] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const normalizedSearchText = searchText.trim().toLocaleLowerCase("zh-CN");
  const sortedHistoryItems = useMemo(
    () => [...props.historyItems].sort(compareHistoryItems),
    [props.historyItems],
  );
  const [results, setResults] = useState(sortedHistoryItems);

  useEffect(() => {
    if (!props.open) {
      setSearchText("");
      setIsLoading(false);
      setResults(sortedHistoryItems);
      return undefined;
    }

    if (!normalizedSearchText) {
      setResults(sortedHistoryItems);
      setIsLoading(false);
      return undefined;
    }

    setIsLoading(true);
    const timeout = window.setTimeout(() => {
      setResults(
        sortedHistoryItems.filter((item) =>
          item.title.toLocaleLowerCase("zh-CN").includes(normalizedSearchText),
        ),
      );
      setIsLoading(false);
    }, SEARCH_DELAY_MS);

    return () => window.clearTimeout(timeout);
  }, [normalizedSearchText, props.open, sortedHistoryItems]);

  if (!props.open) {
    return null;
  }

  function closeOrClear(): void {
    if (searchText.trim()) {
      setSearchText("");
      return;
    }
    props.onClose();
  }

  function selectHistory(sessionId: string): void {
    props.onSelectHistory(sessionId);
    props.onClose();
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

      <div className="mobile-search-body">
        {isLoading ? (
          <div className="mobile-search-loading" role="status" aria-label="正在搜索">
            <Loader2 aria-hidden="true" />
          </div>
        ) : (
          <div className="mobile-search-results">
            {!normalizedSearchText ? <h2>近期对话</h2> : null}
            {results.length > 0 ? (
              <ul>
                {results.map((item) => (
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
                {normalizedSearchText ? "没有找到相关对话" : "暂无可搜索对话"}
              </p>
            )}
          </div>
        )}
      </div>
    </section>
  );
}

function compareHistoryItems(a: SidebarHistoryItem, b: SidebarHistoryItem): number {
  if (a.pinned !== b.pinned) {
    return a.pinned ? -1 : 1;
  }
  const updatedOrder = b.updatedAt.localeCompare(a.updatedAt);
  if (updatedOrder !== 0) {
    return updatedOrder;
  }
  return a.title.localeCompare(b.title, "zh-Hans");
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
