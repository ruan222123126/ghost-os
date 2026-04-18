'use client';

import { type CSSProperties, type FC } from 'react';
import type { ChatCopy } from '@/lib/i18n/messages/chat';

export interface SessionContextMenuState {
  x: number;
  y: number;
  sessionID: string;
}

interface SessionContextMenuProps {
  menu?: SessionContextMenuState;
  menuStyle: CSSProperties;
  renameLabel: string;
  onClose: () => void;
  onRename: () => void;
}

export const SessionContextMenu: FC<SessionContextMenuProps> = ({
  menu,
  menuStyle,
  renameLabel,
  onClose,
  onRename,
}) => {
  if (!menu) {
    return null;
  }

  return (
    <>
      <div className="fixed inset-0 z-40" onMouseDown={onClose} />
      <div
        className="fixed z-50 w-40 border border-black/15 bg-white p-1 shadow-lg"
        role="menu"
        style={menuStyle}
        onMouseDown={(event) => event.stopPropagation()}
        onContextMenu={(event) => event.preventDefault()}
      >
        <button type="button" className="w-full px-3 py-2 text-left text-xs font-semibold hover:bg-neutral-100" onClick={onRename}>
          {renameLabel}
        </button>
      </div>
    </>
  );
};

interface SessionRenameDialogProps {
  copy: ChatCopy;
  open: boolean;
  value: string;
  error: string;
  onChange: (value: string) => void;
  onClose: () => void;
  onConfirm: () => void;
}

export const SessionRenameDialog: FC<SessionRenameDialogProps> = ({
  copy,
  open,
  value,
  error,
  onChange,
  onClose,
  onConfirm,
}) => {
  if (!open) {
    return null;
  }

  return (
    <>
      <div className="fixed inset-0 z-40 bg-black/35" onMouseDown={onClose} />
      <div className="fixed left-1/2 top-1/2 z-50 w-[320px] -translate-x-1/2 -translate-y-1/2 border border-black/10 bg-white p-4 shadow-xl">
        <p className="mb-2 text-sm font-bold text-black">{copy.sidebarSessionRenameTitle}</p>
        <input
          value={value}
          onChange={(event) => onChange(event.target.value)}
          placeholder={copy.sidebarSessionRenamePlaceholder}
          className="w-full border border-black/20 px-3 py-2 text-xs outline-none focus:border-black"
          aria-label={copy.sidebarSessionRenameTitle}
          onKeyDown={(event) => {
            if (event.key === 'Enter') {
              event.preventDefault();
              onConfirm();
            }
          }}
        />
        {error ? <p className="mt-2 text-xs text-red-700">{error}</p> : null}
        <div className="mt-4 flex justify-end gap-2">
          <button type="button" className="px-3 py-1.5 text-xs font-semibold text-neutral-600 hover:bg-neutral-100" onClick={onClose}>{copy.sidebarPartitionCancel}</button>
          <button type="button" className="bg-black px-3 py-1.5 text-xs font-semibold text-white hover:bg-neutral-800" onClick={onConfirm}>{copy.sidebarSessionRenameConfirm}</button>
        </div>
      </div>
    </>
  );
};
