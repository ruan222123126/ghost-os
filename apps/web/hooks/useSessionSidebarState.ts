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

interface SidebarToggleTransition {
  nextOpen: boolean;
  shouldClearSearch: boolean;
}

export function useSessionSidebarState(): UseSessionSidebarStateResult {
  const [isOpen, setIsOpen] = useState(true);
  const [isSearchVisible, setIsSearchVisible] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const searchInputRef = useRef<HTMLInputElement | null>(null);

  useEffect(() => {
    focusSearchInput(isOpen, isSearchVisible, searchInputRef);
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
    toggleSearchVisibility(isSearchVisible, openSearch, closeSearch);
  };

  const toggleSidebar = () => {
    toggleSidebarVisibility(isOpen, setIsOpen, clearSearch);
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

function focusSearchInput(
  isOpen: boolean,
  isSearchVisible: boolean,
  searchInputRef: RefObject<HTMLInputElement>,
): void {
  if (isOpen && isSearchVisible) {
    searchInputRef.current?.focus();
  }
}

function toggleSidebarVisibility(
  isOpen: boolean,
  setIsOpen: (value: boolean) => void,
  clearSearch: () => void,
): void {
  const transition = resolveSidebarToggle(isOpen);
  setIsOpen(transition.nextOpen);
  clearSearchWhenNeeded(transition.shouldClearSearch, clearSearch);
}

function toggleSearchVisibility(
  isSearchVisible: boolean,
  openSearch: () => void,
  closeSearch: () => void,
): void {
  if (isSearchVisible) {
    closeSearch();
    return;
  }
  openSearch();
}

function clearSearchWhenNeeded(shouldClearSearch: boolean, clearSearch: () => void): void {
  if (shouldClearSearch) {
    clearSearch();
  }
}

function resolveSidebarToggle(isOpen: boolean): SidebarToggleTransition {
  return {
    nextOpen: !isOpen,
    shouldClearSearch: isOpen,
  };
}
