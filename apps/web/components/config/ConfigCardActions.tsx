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
    <div className="settings-config-card-actions">
      {props.pillActions.map((action) => (
        <button
          key={action.key}
          type="button"
          disabled={action.disabled}
          onClick={(event) => stopCardAction(event, action.onClick)}
          className="settings-card-action whitespace-nowrap"
        >
          {action.label}
        </button>
      ))}
      <button
        type="button"
        disabled={props.editDisabled}
        onClick={(event) => stopCardAction(event, props.onEdit)}
        className="settings-card-icon-action"
        aria-label={props.editLabel}
        title={props.editTitle}
      >
        <EditIcon />
      </button>
      <button
        type="button"
        disabled={props.deleteDisabled}
        onClick={(event) => stopCardAction(event, props.onDelete)}
        className="settings-card-icon-action is-danger"
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
