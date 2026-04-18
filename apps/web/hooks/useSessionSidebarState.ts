'use client';

import { useEffect, useRef, useState, type RefObject } from 'react';

interface UseSessionSidebarStateResult {
  isOpen: boolean;
  isSearchVisible: boolean;
  searchQuery: string;
  searchInputRef: RefObject<HTMLInputElement>;
  toggleSidebar: () => void;
  toggleSearch: () => void;
  clearSearch: () => void;
  setSearchQuery: (value: string) => void;
  openSidebar: () => void;
}

export function useSessionSidebarState(): UseSessionSidebarStateResult {
  const [isOpen, setIsOpen] = useState(true);
  const [isSearchVisible, setIsSearchVisible] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const searchInputRef = useRef<HTMLInputElement | null>(null);

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
    toggleSidebar,
    toggleSearch: () => setIsSearchVisible((visible) => !visible),
    clearSearch,
    setSearchQuery,
    openSidebar: () => setIsOpen(true),
  };
}
