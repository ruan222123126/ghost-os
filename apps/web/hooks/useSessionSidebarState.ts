'use client';

import { useEffect, useMemo, useRef, useState, type RefObject } from 'react';
import type { SessionMetadata } from '@/lib/types';

interface UseSessionSidebarStateOptions {
  sessions: SessionMetadata[];
}

interface UseSessionSidebarStateResult {
  isOpen: boolean;
  isSearchVisible: boolean;
  searchQuery: string;
  searchInputRef: RefObject<HTMLInputElement>;
  filteredSessions: SessionMetadata[];
  toggleSidebar: () => void;
  toggleSearch: () => void;
  clearSearch: () => void;
  setSearchQuery: (value: string) => void;
  openSidebar: () => void;
}

export function useSessionSidebarState(options: UseSessionSidebarStateOptions): UseSessionSidebarStateResult {
  const { sessions } = options;
  const [isOpen, setIsOpen] = useState(true);
  const [isSearchVisible, setIsSearchVisible] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const searchInputRef = useRef<HTMLInputElement | null>(null);

  const filteredSessions = useMemo(() => {
    const query = searchQuery.trim().toLowerCase();
    if (!query) {
      return sessions;
    }
    return sessions.filter((session) => session.id.toLowerCase().includes(query));
  }, [searchQuery, sessions]);

  useEffect(() => {
    if (isOpen && isSearchVisible) {
      searchInputRef.current?.focus();
    }
  }, [isOpen, isSearchVisible]);

  const clearSearch = () => {
    setSearchQuery('');
    setIsSearchVisible(false);
  };

  const toggleSidebar = () => {
    setIsOpen((open) => {
      const nextOpen = !open;
      if (!nextOpen) {
        clearSearch();
      }
      return nextOpen;
    });
  };

  return {
    isOpen,
    isSearchVisible,
    searchQuery,
    searchInputRef,
    filteredSessions,
    toggleSidebar,
    toggleSearch: () => setIsSearchVisible((visible) => !visible),
    clearSearch,
    setSearchQuery,
    openSidebar: () => setIsOpen(true),
  };
}
