'use client';

import { type MutableRefObject, useEffect, useRef } from 'react';

type SaveHandlerRef = MutableRefObject<() => void>;

export function useWorkflowSaveShortcut(onSave: () => void): void {
  const onSaveRef = useRef(onSave);

  useEffect(function syncLatestSaveHandler() {
    onSaveRef.current = onSave;
  }, [onSave]);

  useEffect(function bindSaveShortcut() {
    return registerWorkflowSaveShortcut(onSaveRef);
  }, []);
}

function registerWorkflowSaveShortcut(onSaveRef: SaveHandlerRef): () => void {
  function handleKeyDown(event: KeyboardEvent) {
    handleSaveShortcutKeyDown(event, onSaveRef.current);
  }

  window.addEventListener('keydown', handleKeyDown, true);
  return function unregisterWorkflowSaveShortcut() {
    window.removeEventListener('keydown', handleKeyDown, true);
  };
}

function handleSaveShortcutKeyDown(event: KeyboardEvent, onSave: () => void): void {
  if (!isWorkflowSaveShortcut(event)) {
    return;
  }
  event.preventDefault();
  onSave();
}

function isWorkflowSaveShortcut(event: KeyboardEvent): boolean {
  return event.key.toLowerCase() === 's' && (event.metaKey || event.ctrlKey);
}
