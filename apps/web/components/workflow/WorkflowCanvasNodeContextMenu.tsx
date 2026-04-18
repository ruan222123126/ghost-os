'use client';

import { useEffect, useMemo, type CSSProperties } from 'react';
import { useWebLocale } from '@/lib/i18n/provider';

const MENU_WIDTH = 144;
const MENU_HEIGHT = 96;
const MENU_OFFSET_X = 8;
const MENU_OFFSET_Y = 8;
const MENU_VIEWPORT_GAP = 8;

export interface WorkflowNodeContextMenuState {
  nodeID: string;
  clientX: number;
  clientY: number;
}

interface WorkflowCanvasNodeContextMenuProps {
  menu?: WorkflowNodeContextMenuState;
  onCopyNode: (nodeID: string) => void;
  onDeleteNode: (nodeID: string) => void;
  onClose: () => void;
}

export function WorkflowCanvasNodeContextMenu(props: WorkflowCanvasNodeContextMenuProps) {
  const { copy } = useWebLocale();
  const { menu, onCopyNode, onDeleteNode, onClose } = props;
  useCloseOnEscape(Boolean(menu), onClose);
  const menuStyle = useMemo(() => buildMenuStyle(menu), [menu]);

  if (!menu) {
    return null;
  }

  return (
    <>
      <div className="workflow-arch-node-menu-backdrop" onMouseDown={onClose} />
      <div
        className="workflow-arch-node-menu"
        role="menu"
        style={menuStyle}
        onMouseDown={(event) => event.stopPropagation()}
        onContextMenu={(event) => event.preventDefault()}
      >
        <button
          type="button"
          className="workflow-arch-node-menu-button"
          onClick={() => {
            onCopyNode(menu.nodeID);
            onClose();
          }}
        >
          {copy.workflow.contextCopy}
        </button>
        <button
          type="button"
          className="workflow-arch-node-menu-button workflow-arch-node-menu-button--danger"
          onClick={() => {
            onDeleteNode(menu.nodeID);
            onClose();
          }}
        >
          {copy.workflow.contextDelete}
        </button>
      </div>
    </>
  );
}

function useCloseOnEscape(enabled: boolean, onClose: () => void) {
  useEffect(() => {
    if (!enabled) {
      return;
    }
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [enabled, onClose]);
}

function buildMenuStyle(menu?: WorkflowNodeContextMenuState): CSSProperties {
  if (!menu) {
    return {};
  }

  const rawLeft = menu.clientX + MENU_OFFSET_X;
  const rawTop = menu.clientY + MENU_OFFSET_Y;
  if (typeof window === 'undefined') {
    return { left: rawLeft, top: rawTop };
  }

  const left = clamp(rawLeft, MENU_VIEWPORT_GAP, window.innerWidth - MENU_WIDTH - MENU_VIEWPORT_GAP);
  const top = clamp(rawTop, MENU_VIEWPORT_GAP, window.innerHeight - MENU_HEIGHT - MENU_VIEWPORT_GAP);
  return { left, top };
}

function clamp(value: number, min: number, max: number): number {
  if (max < min) {
    return min;
  }
  return Math.min(Math.max(value, min), max);
}
