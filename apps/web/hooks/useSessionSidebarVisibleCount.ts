'use client';

import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type Dispatch,
  type RefObject,
  type SetStateAction,
} from 'react';

const SESSION_VISIBLE_BATCH = 30;
const SESSION_LOAD_MORE_THRESHOLD_PX = 160;
const SESSION_FOCUS_CONTEXT_COUNT = 12;

export function useSessionSidebarVisibleCount(options: {
  scrollElementRef: RefObject<HTMLDivElement>;
  totalSessions: number;
  resetKey: string;
  focusSessionIndex?: number;
}): number {
  const { scrollElementRef, totalSessions, resetKey, focusSessionIndex } = options;
  const [visibleCount, setVisibleCount] = useState(0);

  useResetVisibleSessionCount({ resetKey, setVisibleCount, totalSessions });
  useFocusedVisibleSessionCount({ focusSessionIndex, setVisibleCount, totalSessions });
  useLoadMoreVisibleSessions({
    scrollElementRef,
    setVisibleCount,
    totalSessions,
    visibleCount,
    resetKey,
  });

  return visibleCount;
}

function useResetVisibleSessionCount(options: {
  resetKey: string;
  setVisibleCount: Dispatch<SetStateAction<number>>;
  totalSessions: number;
}): void {
  const { resetKey, setVisibleCount, totalSessions } = options;
  const resetKeyRef = useRef(resetKey);

  useEffect(() => {
    const didResetKeyChange = resetKeyRef.current !== resetKey;
    resetKeyRef.current = resetKey;
    setVisibleCount((current) => {
      if (didResetKeyChange) {
        return resolveInitialVisibleSessionCount(totalSessions);
      }
      return clampVisibleSessionCount(current, totalSessions);
    });
  }, [resetKey, setVisibleCount, totalSessions]);
}

function useFocusedVisibleSessionCount(options: {
  focusSessionIndex?: number;
  setVisibleCount: Dispatch<SetStateAction<number>>;
  totalSessions: number;
}): void {
  const { focusSessionIndex, setVisibleCount, totalSessions } = options;

  useEffect(() => {
    if (focusSessionIndex === undefined || focusSessionIndex < 0) {
      return;
    }
    setVisibleCount((current) => {
      const focusedCount = resolveFocusedVisibleSessionCount(focusSessionIndex, totalSessions);
      return focusedCount > current ? focusedCount : current;
    });
  }, [focusSessionIndex, setVisibleCount, totalSessions]);
}

function useLoadMoreVisibleSessions(options: {
  resetKey: string;
  scrollElementRef: RefObject<HTMLDivElement>;
  setVisibleCount: Dispatch<SetStateAction<number>>;
  totalSessions: number;
  visibleCount: number;
}): void {
  const { resetKey, scrollElementRef, setVisibleCount, totalSessions, visibleCount } = options;

  const maybeLoadMore = useCallback(() => {
    const scrollElement = scrollElementRef.current;
    if (!shouldLoadMoreVisibleSessions(scrollElement, visibleCount, totalSessions)) {
      return;
    }

    setVisibleCount((current) => {
      return resolveNextVisibleSessionCount(current, totalSessions);
    });
  }, [scrollElementRef, setVisibleCount, totalSessions, visibleCount]);

  useEffect(() => {
    const scrollElement = scrollElementRef.current;
    if (!scrollElement) {
      return;
    }

    scrollElement.addEventListener('scroll', maybeLoadMore, { passive: true });
    return () => {
      scrollElement.removeEventListener('scroll', maybeLoadMore);
    };
  }, [maybeLoadMore, scrollElementRef, resetKey]);

  useEffect(() => {
    maybeLoadMore();
  }, [maybeLoadMore, visibleCount]);
}

function shouldLoadMoreVisibleSessions(
  scrollElement: HTMLDivElement | null,
  visibleCount: number,
  totalSessions: number,
): boolean {
  if (!scrollElement || visibleCount >= totalSessions) {
    return false;
  }

  return getRemainingScrollDistance(scrollElement) <= SESSION_LOAD_MORE_THRESHOLD_PX;
}

function getRemainingScrollDistance(scrollElement: HTMLDivElement): number {
  return scrollElement.scrollHeight - scrollElement.scrollTop - scrollElement.clientHeight;
}

function resolveNextVisibleSessionCount(current: number, totalSessions: number): number {
  if (current >= totalSessions) {
    return current;
  }
  return Math.min(totalSessions, current + SESSION_VISIBLE_BATCH);
}

function resolveInitialVisibleSessionCount(totalSessions: number): number {
  if (totalSessions <= 0) {
    return 0;
  }
  return Math.min(totalSessions, SESSION_VISIBLE_BATCH);
}

function resolveFocusedVisibleSessionCount(
  focusSessionIndex: number,
  totalSessions: number,
): number {
  if (totalSessions <= 0) {
    return 0;
  }
  return Math.min(totalSessions, focusSessionIndex + 1 + SESSION_FOCUS_CONTEXT_COUNT);
}

function clampVisibleSessionCount(
  current: number,
  totalSessions: number,
): number {
  if (totalSessions <= 0) {
    return 0;
  }
  if (current <= 0) {
    return resolveInitialVisibleSessionCount(totalSessions);
  }
  if (current > totalSessions) {
    return totalSessions;
  }
  return current;
}
