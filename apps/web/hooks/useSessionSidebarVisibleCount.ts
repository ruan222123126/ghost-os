'use client';

import { useCallback, useEffect, useRef, useState, type RefObject } from 'react';

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
  }, [resetKey, totalSessions]);

  useEffect(() => {
    if (focusSessionIndex === undefined || focusSessionIndex < 0) {
      return;
    }
    setVisibleCount((current) => {
      const focusedCount = resolveFocusedVisibleSessionCount(focusSessionIndex, totalSessions);
      return focusedCount > current ? focusedCount : current;
    });
  }, [focusSessionIndex, totalSessions]);

  const maybeLoadMore = useCallback(() => {
    const scrollElement = scrollElementRef.current;
    if (!scrollElement || visibleCount >= totalSessions) {
      return;
    }

    const remainingDistance = scrollElement.scrollHeight - scrollElement.scrollTop - scrollElement.clientHeight;
    if (remainingDistance > SESSION_LOAD_MORE_THRESHOLD_PX) {
      return;
    }

    setVisibleCount((current) => {
      if (current >= totalSessions) {
        return current;
      }
      return Math.min(totalSessions, current + SESSION_VISIBLE_BATCH);
    });
  }, [scrollElementRef, totalSessions, visibleCount]);

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

  return visibleCount;
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
