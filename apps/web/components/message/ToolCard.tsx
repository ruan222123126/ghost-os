'use client';

import type { FC } from 'react';
import { useWebLocale } from '@/lib/i18n/provider';
import type { ToolChatMessage } from '@/lib/types';
import { formatToolAction, formatToolDetails } from './format';

type ToolTone = 'running' | 'success' | 'error';

const RUNNING_STATUSES = new Set(['running', 'pending', 'in_progress']);
const ERROR_STATUSES = new Set(['error', 'failed']);

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

const XIcon: FC<{ className?: string }> = ({ className }) => (
  <svg viewBox="0 0 20 20" fill="none" aria-hidden="true" className={className}>
    <path d="M6.8 6.8L13.2 13.2" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
    <path d="M13.2 6.8L6.8 13.2" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
  </svg>
);

const ChevronIcon: FC<{ expanded: boolean; className?: string }> = ({ expanded, className }) => (
  <svg viewBox="0 0 20 20" fill="none" aria-hidden="true" className={className}>
    {expanded ? (
      <path d="M6.2 8.2L10 12l3.8-3.8" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
    ) : (
      <path d="M8.2 6.2L12 10l-3.8 3.8" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
    )}
  </svg>
);

const TerminalIcon: FC<{ className?: string }> = ({ className }) => (
  <svg viewBox="0 0 20 20" fill="none" aria-hidden="true" className={className}>
    <rect x="3.5" y="4.5" width="13" height="11" rx="2" stroke="currentColor" strokeWidth="1.3" />
    <path d="M7 8L9 10L7 12" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round" />
    <path d="M10.5 12H13.5" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" />
  </svg>
);

function normalizeStatus(status?: string): string {
  return status?.trim().toLowerCase() || '';
}

function getToolTone(status?: string): ToolTone {
  const normalized = normalizeStatus(status);
  if (!normalized || RUNNING_STATUSES.has(normalized)) {
    return 'running';
  }
  if (ERROR_STATUSES.has(normalized)) {
    return 'error';
  }
  return 'success';
}

function getStatusLabel(tone: ToolTone, status?: string): string {
  if (tone === 'running') {
    return 'RUNNING';
  }
  if (tone === 'error') {
    return 'FAILED';
  }
  const normalized = normalizeStatus(status);
  if (!normalized || normalized === 'ok' || normalized === 'done' || normalized === 'completed') {
    return 'SUCCESS';
  }
  return normalized.replaceAll('_', ' ').toUpperCase();
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
  const tone = getToolTone(tool.toolStatus);
  const statusLabel = getStatusLabel(tone, tool.toolStatus);

  return (
    <div className={`tool-card is-${tone}`}>
      <button
        type="button"
        onClick={onToggle}
        className={`tool-card-button${isOpen ? ' is-open' : ''} is-${tone}`}
      >
        <span className="tool-card-heading">
          <ChevronIcon expanded={isOpen} className="tool-chevron" />
          <TerminalIcon className="tool-terminal-icon" />
          <span className="tool-card-title" title={action}>{action}</span>
        </span>
        <span className={`tool-card-status is-${tone}`}>
          {tone === 'running'
            ? <span className="tool-spinner" />
            : tone === 'error'
              ? <XIcon className="tool-status-icon" />
              : <CheckIcon className="tool-status-icon" />}
          <span className="tool-card-status-label">{statusLabel}</span>
        </span>
      </button>

      {isOpen ? (
        <div className={`tool-details is-${tone}`}>
          <pre>{details || copy.chat.toolPreparingOutput}</pre>
        </div>
      ) : null}
    </div>
  );
};
