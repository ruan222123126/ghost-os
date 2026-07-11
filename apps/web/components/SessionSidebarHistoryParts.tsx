'use client';

import { type CSSProperties, type FC } from 'react';
import { CloseButton } from '@/components/CloseButton';
import type { ChatCopy } from '@/lib/i18n/messages/chat';

export interface ContextMenuState {
  x: number;
  y: number;
  partitionID: string;
  partitionName: string;
}

interface SidebarHistoryContextMenuProps {
  menu?: ContextMenuState;
  menuStyle: CSSProperties;
  createLabel: string;
  renameLabel: string;
  deleteLabel: string;
  showPartitionActions: boolean;
  onClose: () => void;
  onCreate: () => void;
  onRename: () => void;
  onDelete: () => void;
}

export const SidebarHistoryContextMenu: FC<SidebarHistoryContextMenuProps> = ({
  menu,
  menuStyle,
  createLabel,
  renameLabel,
  deleteLabel,
  showPartitionActions,
  onClose,
  onCreate,
  onRename,
  onDelete,
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
        <button type="button" className="w-full px-3 py-2 text-left text-xs font-semibold hover:bg-neutral-100" onClick={onCreate}>
          {createLabel}
        </button>
        {showPartitionActions ? (
          <>
            <button type="button" className="w-full px-3 py-2 text-left text-xs font-semibold hover:bg-neutral-100" onClick={onRename}>
              {renameLabel}
            </button>
            <button type="button" className="w-full px-3 py-2 text-left text-xs font-semibold text-red-700 hover:bg-red-50" onClick={onDelete}>
              {deleteLabel}
            </button>
          </>
        ) : null}
      </div>
    </>
  );
};

interface PartitionCreateDialogProps {
  copy: ChatCopy;
  open: boolean;
  value: string;
  error: string;
  onChange: (value: string) => void;
  onClose: () => void;
  onCreate: () => void;
}

export const PartitionCreateDialog: FC<PartitionCreateDialogProps> = ({ copy, open, value, error, onChange, onClose, onCreate }) => {
  if (!open) {
    return null;
  }

  return (
    <>
      <div className="fixed inset-0 z-40 bg-black/35" onMouseDown={onClose} />
      <div className="fixed left-1/2 top-1/2 z-50 w-[320px] -translate-x-1/2 -translate-y-1/2 border border-black/10 bg-white p-4 shadow-xl">
        <div className="mb-2 flex items-center justify-between gap-3">
          <p className="text-sm font-bold text-black">{copy.sidebarPartitionCreateTitle}</p>
          <CloseButton className="shrink-0" onClick={onClose} aria-label={copy.sidebarPartitionCancel} />
        </div>
        <input
          value={value}
          onChange={(event) => onChange(event.target.value)}
          placeholder={copy.sidebarPartitionNamePlaceholder}
          className="w-full border border-black/20 px-3 py-2 text-xs outline-none focus:border-black"
          aria-label={copy.sidebarPartitionNameLabel}
          onKeyDown={(event) => {
            if (event.key === 'Enter') {
              event.preventDefault();
              onCreate();
            }
          }}
        />
        {error ? <p className="mt-2 text-xs text-red-700">{error}</p> : null}
        <div className="mt-4 flex justify-end gap-2">
          <button type="button" className="bg-black px-3 py-1.5 text-xs font-semibold text-white hover:bg-neutral-800" onClick={onCreate}>{copy.sidebarPartitionCreateConfirm}</button>
        </div>
      </div>
    </>
  );
};
