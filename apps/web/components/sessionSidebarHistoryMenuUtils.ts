'use client';

import { type CSSProperties, useEffect } from 'react';
import type { ChatCopy } from '@/lib/i18n/messages/chat';

const CONTEXT_MENU_WIDTH = 164;
const CONTEXT_MENU_DEFAULT_HEIGHT = 48;
const CONTEXT_MENU_OFFSET = 8;
const CONTEXT_MENU_GAP = 8;

interface MenuPoint {
  x: number;
  y: number;
}

export function useCloseOnEscape(enabled: boolean, onClose: () => void) {
  useEffect(() => {
    if (!enabled) {
      return;
    }

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose();
      }
    };

    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [enabled, onClose]);
}

export function resolveContextMenuStyle(menu?: MenuPoint, menuHeight = CONTEXT_MENU_DEFAULT_HEIGHT): CSSProperties {
  if (!menu) {
    return {};
  }

  const rawLeft = menu.x + CONTEXT_MENU_OFFSET;
  const rawTop = menu.y + CONTEXT_MENU_OFFSET;
  if (typeof window === 'undefined') {
    return { left: rawLeft, top: rawTop };
  }

  return {
    left: clamp(rawLeft, CONTEXT_MENU_GAP, window.innerWidth - CONTEXT_MENU_WIDTH - CONTEXT_MENU_GAP),
    top: clamp(rawTop, CONTEXT_MENU_GAP, window.innerHeight - menuHeight - CONTEXT_MENU_GAP),
  };
}

export function resolvePartitionNameErrorText(copy: ChatCopy, error?: 'empty' | 'duplicate'): string {
  if (error === 'duplicate') {
    return copy.sidebarPartitionNameDuplicate;
  }
  return copy.sidebarPartitionNameRequired;
}

function clamp(value: number, min: number, max: number): number {
  if (max < min) {
    return min;
  }
  return Math.min(Math.max(value, min), max);
}
