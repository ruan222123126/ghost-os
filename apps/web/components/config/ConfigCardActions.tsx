'use client';

import type { MouseEvent } from 'react';
import { EditIcon, TrashIcon } from '@/components/config/configCardIcons';

interface CardPillAction {
  key: string;
  label: string;
  disabled?: boolean;
  onClick: () => void;
}

interface ConfigCardActionsProps {
  pillActions: CardPillAction[];
  editLabel: string;
  editTitle?: string;
  editDisabled?: boolean;
  onEdit: () => void;
  deleteLabel: string;
  deleteDisabled?: boolean;
  onDelete: () => void;
}

export function ConfigCardActions(props: ConfigCardActionsProps) {
  return (
    <div className="flex items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100 group-focus-within:opacity-100">
      {props.pillActions.map((action) => (
        <button
          key={action.key}
          type="button"
          disabled={action.disabled}
          onClick={(event) => stopCardAction(event, action.onClick)}
          className="rounded-full border border-[#E5E5E5] px-3 py-1.5 text-[12px] font-medium text-[#111111] transition-colors hover:bg-[#F5F5F5] disabled:cursor-not-allowed disabled:opacity-50"
        >
          {action.label}
        </button>
      ))}
      <button
        type="button"
        disabled={props.editDisabled}
        onClick={(event) => stopCardAction(event, props.onEdit)}
        className="rounded-full p-2 text-[#737373] transition-colors hover:bg-[#F5F5F5] hover:text-[#111111] disabled:cursor-not-allowed disabled:opacity-50"
        aria-label={props.editLabel}
        title={props.editTitle}
      >
        <EditIcon />
      </button>
      <button
        type="button"
        disabled={props.deleteDisabled}
        onClick={(event) => stopCardAction(event, props.onDelete)}
        className="rounded-full p-2 text-[#737373] transition-colors hover:bg-[#FEF2F2] hover:text-[#DC2626] disabled:cursor-not-allowed disabled:opacity-50"
        aria-label={props.deleteLabel}
      >
        <TrashIcon />
      </button>
    </div>
  );
}

function stopCardAction(event: MouseEvent<HTMLButtonElement>, action: () => void) {
  event.stopPropagation();
  action();
}
