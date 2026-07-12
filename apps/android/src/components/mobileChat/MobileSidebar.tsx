import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { Dispatch, ReactNode, RefObject, SetStateAction } from "react";
import type { ConfigPayload, HostProfile, StoredSettings } from "../../mobileTypes";
import { compareHistoryItems } from "./historyItems";
import { UiIcon } from "./icons";
import type { SidebarHistoryItem, UiIconName } from "./types";

interface MobileSidebarProps {
  open: boolean;
  host: HostProfile | undefined;
  config: ConfigPayload | undefined;
  settings: StoredSettings;
  historyItems: SidebarHistoryItem[];
  activeHistoryId: string | undefined;
  onClose: () => void;
  onNewSession: () => void;
  onOpenSearch: () => void;
  onSelectHistory: (sessionId: string) => void;
  onConnect: () => Promise<void>;
  onOpenSettings: () => void;
}

const HISTORY_VISIBLE_BATCH = 30;
const HISTORY_LOAD_MORE_THRESHOLD_PX = 160;
const HISTORY_FOCUS_CONTEXT_COUNT = 12;

export function MobileSidebar(props: MobileSidebarProps) {
  const accountName = props.host?.productName || "Ghost-OS Mobile";
  const isBridgeConnected = Boolean(props.config);
  const bridgeStatusLabel = isBridgeConnected ? "CONNECTED" : "DISCONNECTED";
  const scrollElementRef = useRef<HTMLDivElement | null>(null);
  const sortedHistoryItems = useMemo(
    () => [...props.historyItems].sort(compareHistoryItems),
    [props.historyItems],
  );
  const activeHistoryIndex = useMemo(() => {
    if (!props.activeHistoryId) {
      return -1;
    }
    return sortedHistoryItems.findIndex((item) => item.id === props.activeHistoryId);
  }, [props.activeHistoryId, sortedHistoryItems]);
  const visibleHistoryCount = useMobileSidebarVisibleCount({
    enabled: props.open,
    focusHistoryIndex: activeHistoryIndex >= 0 ? activeHistoryIndex : undefined,
    scrollElementRef,
    totalHistoryItems: sortedHistoryItems.length,
  });
  const visibleHistoryItems = useMemo(
    () => sortedHistoryItems.slice(0, visibleHistoryCount),
    [sortedHistoryItems, visibleHistoryCount],
  );

  return (
    <>
      <button
        className={`sidebar-scrim ${props.open ? "is-open" : ""}`}
        type="button"
        aria-label="关闭侧边栏"
        onClick={props.onClose}
      />
      <aside
        className={`mobile-sidebar ${props.open ? "is-open" : ""}`}
        role="dialog"
        aria-hidden={!props.open}
        aria-modal="true"
        inert={props.open ? undefined : true}
      >
        <div className="sidebar-scroll" ref={scrollElementRef}>
          <div className="sidebar-brand">
            <h2>Ghost-OS</h2>
          </div>

          <nav className="sidebar-nav" aria-label="主要操作">
            <SidebarNavButton icon="edit" label="发起新任务" onClick={props.onNewSession} />
            <SidebarNavButton icon="search" label="搜索任务内容" onClick={props.onOpenSearch} />
          </nav>

          <SidebarSection title="最近">
            <SidebarHistoryList
              activeHistoryId={props.activeHistoryId}
              historyItems={visibleHistoryItems}
              onSelectHistory={props.onSelectHistory}
              totalHistoryItems={sortedHistoryItems.length}
            />
          </SidebarSection>
        </div>

        <footer className="sidebar-footer">
          <div className="sidebar-account">
            <div className="sidebar-account-copy">
              <span>{accountName}</span>
              <strong className={isBridgeConnected ? "is-connected" : undefined}>{bridgeStatusLabel}</strong>
            </div>
          </div>
          <button
            className="sidebar-settings-button"
            type="button"
            aria-label="打开设置"
            title={props.settings.bridgeUrl}
            onClick={props.onOpenSettings}
          >
            <UiIcon name="settings" />
          </button>
        </footer>
      </aside>
    </>
  );
}

function SidebarHistoryList(props: {
  activeHistoryId: string | undefined;
  historyItems: SidebarHistoryItem[];
  onSelectHistory: (sessionId: string) => void;
  totalHistoryItems: number;
}) {
  if (props.totalHistoryItems === 0) {
    return <p className="history-empty">暂无会话</p>;
  }

  return (
    <div className="history-list">
      {props.historyItems.map((item) => (
        <SidebarHistoryButton
          key={item.id}
          active={props.activeHistoryId === item.id}
          item={item}
          onSelectHistory={props.onSelectHistory}
        />
      ))}
    </div>
  );
}

function SidebarHistoryButton(props: {
  active: boolean;
  item: SidebarHistoryItem;
  onSelectHistory: (sessionId: string) => void;
}) {
  const { active, item, onSelectHistory } = props;

  return (
    <button
      className={`history-item ${active ? "is-active" : ""} ${item.pinned ? "is-pinned" : ""}`}
      type="button"
      title={item.title}
      onClick={() => onSelectHistory(item.id)}
    >
      <span>{item.title}</span>
      <span className="history-item-indicators">
        {item.unread ? (
          <span className="history-unread-indicator" aria-label="有新回复" title="有新回复" />
        ) : null}
      </span>
      {item.pinned ? (
        <span className="history-pin-indicator" aria-label="已固定" title="已固定">
          <UiIcon name="pin" />
        </span>
      ) : null}
    </button>
  );
}

function useMobileSidebarVisibleCount(options: {
  enabled: boolean;
  focusHistoryIndex?: number;
  scrollElementRef: RefObject<HTMLDivElement | null>;
  totalHistoryItems: number;
}): number {
  const { enabled, focusHistoryIndex, scrollElementRef, totalHistoryItems } = options;
  const [visibleCount, setVisibleCount] = useState(() => resolveInitialVisibleHistoryCount(totalHistoryItems));

  useResetVisibleHistoryCount({ enabled, setVisibleCount, totalHistoryItems });
  useFocusedVisibleHistoryCount({ enabled, focusHistoryIndex, setVisibleCount, totalHistoryItems });
  useLoadMoreVisibleHistory({
    enabled,
    scrollElementRef,
    setVisibleCount,
    totalHistoryItems,
    visibleCount,
  });

  return visibleCount;
}

function useResetVisibleHistoryCount(options: {
  enabled: boolean;
  setVisibleCount: Dispatch<SetStateAction<number>>;
  totalHistoryItems: number;
}): void {
  const { enabled, setVisibleCount, totalHistoryItems } = options;

  useEffect(() => {
    setVisibleCount((current) => {
      if (!enabled) {
        return resolveInitialVisibleHistoryCount(totalHistoryItems);
      }
      return clampVisibleHistoryCount(current, totalHistoryItems);
    });
  }, [enabled, setVisibleCount, totalHistoryItems]);
}

function useFocusedVisibleHistoryCount(options: {
  enabled: boolean;
  focusHistoryIndex?: number;
  setVisibleCount: Dispatch<SetStateAction<number>>;
  totalHistoryItems: number;
}): void {
  const { enabled, focusHistoryIndex, setVisibleCount, totalHistoryItems } = options;

  useEffect(() => {
    if (!enabled || focusHistoryIndex === undefined || focusHistoryIndex < 0) {
      return;
    }
    setVisibleCount((current) => {
      const focusedCount = resolveFocusedVisibleHistoryCount(focusHistoryIndex, totalHistoryItems);
      return focusedCount > current ? focusedCount : current;
    });
  }, [enabled, focusHistoryIndex, setVisibleCount, totalHistoryItems]);
}

function useLoadMoreVisibleHistory(options: {
  enabled: boolean;
  scrollElementRef: RefObject<HTMLDivElement | null>;
  setVisibleCount: Dispatch<SetStateAction<number>>;
  totalHistoryItems: number;
  visibleCount: number;
}): void {
  const { enabled, scrollElementRef, setVisibleCount, totalHistoryItems, visibleCount } = options;

  const maybeLoadMore = useCallback(() => {
    if (!enabled) {
      return;
    }
    const scrollElement = scrollElementRef.current;
    if (!shouldLoadMoreVisibleHistory(scrollElement, visibleCount, totalHistoryItems)) {
      return;
    }

    setVisibleCount((current) => resolveNextVisibleHistoryCount(current, totalHistoryItems));
  }, [enabled, scrollElementRef, setVisibleCount, totalHistoryItems, visibleCount]);

  useEffect(() => {
    if (!enabled) {
      return;
    }
    const scrollElement = scrollElementRef.current;
    if (!scrollElement) {
      return;
    }

    scrollElement.addEventListener("scroll", maybeLoadMore, { passive: true });
    return () => {
      scrollElement.removeEventListener("scroll", maybeLoadMore);
    };
  }, [enabled, maybeLoadMore, scrollElementRef]);

  useEffect(() => {
    maybeLoadMore();
  }, [maybeLoadMore, visibleCount]);
}

function shouldLoadMoreVisibleHistory(
  scrollElement: HTMLDivElement | null,
  visibleCount: number,
  totalHistoryItems: number,
): boolean {
  if (
    !scrollElement
    || visibleCount >= totalHistoryItems
    || scrollElement.clientHeight <= 0
    || scrollElement.scrollHeight <= 0
  ) {
    return false;
  }

  return getRemainingScrollDistance(scrollElement) <= HISTORY_LOAD_MORE_THRESHOLD_PX;
}

function getRemainingScrollDistance(scrollElement: HTMLDivElement): number {
  return scrollElement.scrollHeight - scrollElement.scrollTop - scrollElement.clientHeight;
}

function resolveNextVisibleHistoryCount(current: number, totalHistoryItems: number): number {
  if (current >= totalHistoryItems) {
    return current;
  }
  return Math.min(totalHistoryItems, current + HISTORY_VISIBLE_BATCH);
}

function resolveInitialVisibleHistoryCount(totalHistoryItems: number): number {
  if (totalHistoryItems <= 0) {
    return 0;
  }
  return Math.min(totalHistoryItems, HISTORY_VISIBLE_BATCH);
}

function resolveFocusedVisibleHistoryCount(focusHistoryIndex: number, totalHistoryItems: number): number {
  if (totalHistoryItems <= 0) {
    return 0;
  }
  return Math.min(totalHistoryItems, focusHistoryIndex + 1 + HISTORY_FOCUS_CONTEXT_COUNT);
}

function clampVisibleHistoryCount(current: number, totalHistoryItems: number): number {
  if (totalHistoryItems <= 0) {
    return 0;
  }
  if (current <= 0) {
    return resolveInitialVisibleHistoryCount(totalHistoryItems);
  }
  if (current > totalHistoryItems) {
    return totalHistoryItems;
  }
  return current;
}

function SidebarSection(props: { title: string; children: ReactNode }) {
  return (
    <section className="sidebar-section">
      <h3>{props.title}</h3>
      {props.children}
    </section>
  );
}

function SidebarNavButton(props: { icon: UiIconName; label: string; onClick: () => void }) {
  return (
    <button className="sidebar-nav-button" type="button" onClick={props.onClick}>
      <UiIcon name={props.icon} />
      <span>{props.label}</span>
    </button>
  );
}
