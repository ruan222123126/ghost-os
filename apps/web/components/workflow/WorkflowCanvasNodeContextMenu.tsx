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

interface NodeContextMenuAction {
  danger: boolean;
  key: 'copy' | 'delete';
  label: string;
  onSelect: () => void;
}

export function WorkflowCanvasNodeContextMenu(props: WorkflowCanvasNodeContextMenuProps) {
  const { copy } = useWebLocale();
  const { menu, onCopyNode, onDeleteNode, onClose } = props;
  useCloseOnEscape(Boolean(menu), onClose);
  const menuStyle = useMemo(() => buildMenuStyle(menu), [menu]);

  if (!menu) {
    return null;
  }

  const actions = buildContextMenuActions({ copy, menu, onClose, onCopyNode, onDeleteNode });
  return (
    <WorkflowCanvasNodeContextMenuContent actions={actions} menuStyle={menuStyle} onClose={onClose} />
  );
}

function buildContextMenuActions(options: {
  copy: ReturnType<typeof useWebLocale>['copy'];
  menu: WorkflowNodeContextMenuState;
  onClose: () => void;
  onCopyNode: (nodeID: string) => void;
  onDeleteNode: (nodeID: string) => void;
}): NodeContextMenuAction[] {
  const { copy, menu, onClose, onCopyNode, onDeleteNode } = options;

  return [
    {
      danger: false,
      key: 'copy',
      label: copy.workflow.contextCopy,
      onSelect: () => {
        onCopyNode(menu.nodeID);
        onClose();
      },
    },
    {
      danger: true,
      key: 'delete',
      label: copy.workflow.contextDelete,
      onSelect: () => {
        onDeleteNode(menu.nodeID);
        onClose();
      },
    },
  ];
}

function WorkflowCanvasNodeContextMenuContent(props: {
  actions: NodeContextMenuAction[];
  menuStyle: CSSProperties;
  onClose: () => void;
}) {
  const { actions, menuStyle, onClose } = props;

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
        {actions.map((action) => (
          <WorkflowCanvasNodeContextMenuButton action={action} key={action.key} />
        ))}
      </div>
    </>
  );
}

function WorkflowCanvasNodeContextMenuButton(props: {
  action: NodeContextMenuAction;
}) {
  const { action } = props;
  const className = action.danger
    ? 'workflow-arch-node-menu-button workflow-arch-node-menu-button--danger'
    : 'workflow-arch-node-menu-button';

  return (
    <button type="button" className={className} onClick={action.onSelect}>
      {action.label}
    </button>
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
