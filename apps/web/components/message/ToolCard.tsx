'use client';

import type { FC } from 'react';
import { useWebLocale } from '@/lib/i18n/provider';
import type { ToolChatMessage } from '@/lib/types';
import { formatToolAction, formatToolDetails } from './format';

const CheckIcon: FC<{ className?: string }> = ({ className }) => (
  <svg viewBox="0 0 20 20" fill="none" aria-hidden="true" className={className}>
    <path
      d="M5 10.5l3.4 3.4L15.5 6.8"
      stroke="currentColor"
      strokeWidth="1.6"
      strokeLinecap="round"
      strokeLinejoin="round"
    />
  </svg>
);

const ChevronIcon: FC<{ className?: string }> = ({ className }) => (
  <svg viewBox="0 0 20 20" fill="none" aria-hidden="true" className={className}>
    <path d="M6.2 8.4L10 12.2l3.8-3.8" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
  </svg>
);

function isToolRunning(status?: string): boolean {
  const normalized = status?.toLowerCase();
  return normalized === 'running' || normalized === 'pending' || normalized === 'in_progress';
}

function isToolError(status?: string): boolean {
  const normalized = status?.toLowerCase();
  return normalized === 'error' || normalized === 'failed';
}

interface ToolCardProps {
  isOpen: boolean;
  onToggle: () => void;
  tool: ToolChatMessage;
}

export const ToolCard: FC<ToolCardProps> = ({ isOpen, onToggle, tool }) => {
  const { copy } = useWebLocale();
  const actionValue = formatToolAction(tool);
  const action = actionValue === 'Tool' ? copy.chat.toolFallbackName : actionValue;
  const details = formatToolDetails(tool);
  const hasError = isToolError(tool.toolStatus);

  return (
    <div className="tool-card">
      <button
        type="button"
        onClick={onToggle}
        className={`tool-card-button${isOpen ? ' is-open' : ''}${hasError ? ' is-error' : ''}`}
      >
        <span className="tool-card-title" title={action}>
          {action}
        </span>
        <span className="tool-card-meta">
          <span className={`tool-status${hasError ? ' is-error' : ''}`}>
            {isToolRunning(tool.toolStatus) ? <span className="tool-spinner" /> : <CheckIcon className="tool-check" />}
          </span>
          <ChevronIcon className="tool-chevron" />
        </span>
      </button>

      {isOpen ? (
        <div className="tool-details">
          <pre>{details || copy.chat.toolPreparingOutput}</pre>
        </div>
      ) : null}
    </div>
  );
};
