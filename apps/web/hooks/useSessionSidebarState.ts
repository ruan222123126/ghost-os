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
  closeSearch: () => void;
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

  const closeSearch = () => {
    setIsSearchVisible(false);
  };

  const clearSearch = () => {
    setSearchQuery('');
    closeSearch();
  };

  const openSearch = () => {
    setIsOpen(true);
    setIsSearchVisible(true);
  };

  const toggleSearch = () => {
    if (isSearchVisible) {
      closeSearch();
      return;
    }
    openSearch();
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
    toggleSearch,
    clearSearch,
    closeSearch,
    setSearchQuery,
    openSidebar: () => setIsOpen(true),
  };
}
