'use client';

import type { FC } from 'react';
import type { ToolCardTitleMode, ToolTone } from '@/lib/chat-view/types';
import { useWebLocale } from '@/lib/i18n/provider';

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

interface ToolCardProps {
  details: string;
  isOpen: boolean;
  onToggle: () => void;
  showTerminalIcon?: boolean;
  statusLabel: string;
  title: string;
  titleMode?: ToolCardTitleMode;
  tone: ToolTone;
}

export const ToolCard: FC<ToolCardProps> = ({
  details,
  isOpen,
  onToggle,
  showTerminalIcon = true,
  statusLabel,
  title,
  titleMode = 'status',
  tone,
}) => {
  const { copy } = useWebLocale();
  const displayTitle = titleMode === 'plain' ? title : buildToolStatusTitle(tone, title, copy.chat);
  const displayStatus = buildToolStatusLabel(tone, statusLabel, copy.chat);

  return (
    <div className={`tool-card is-${tone}`}>
      <button
        type="button"
        onClick={onToggle}
        className={`tool-card-button${isOpen ? ' is-open' : ''} is-${tone}`}
      >
        <span className="tool-card-heading">
          {showTerminalIcon ? <TerminalIcon className="tool-terminal-icon" /> : null}
          <span className="tool-card-title" title={displayTitle}>{displayTitle}</span>
          <ChevronIcon expanded={isOpen} className="tool-chevron" />
        </span>
      </button>

      {isOpen ? (
        <div className={`tool-details is-${tone}`}>
          <span className="tool-details-kind">{copy.chat.toolDetailsKind}</span>
          <div className="tool-details-command">{title}</div>
          {details ? (
            <div className="tool-details-output">
              <pre>{details}</pre>
            </div>
          ) : null}
          <div className={`tool-card-status is-${tone}`}>
            {tone === 'running'
              ? <span className="tool-spinner" />
              : tone === 'error'
                ? <XIcon className="tool-status-icon" />
                : <CheckIcon className="tool-status-icon" />}
            <span className="tool-card-status-label">{displayStatus}</span>
          </div>
        </div>
      ) : null}
    </div>
  );
};

function buildToolStatusTitle(
  tone: ToolTone,
  title: string,
  chat: ReturnType<typeof useWebLocale>['copy']['chat'],
): string {
  switch (tone) {
    case 'running':
      return chat.toolRunningTitle(title);
    case 'error':
      return chat.toolErrorTitle(title);
    case 'success':
      return chat.toolSuccessTitle(title);
  }
}

function buildToolStatusLabel(
  tone: ToolTone,
  statusLabel: string,
  chat: ReturnType<typeof useWebLocale>['copy']['chat'],
): string {
  switch (tone) {
    case 'running':
      return chat.toolStatusRunning;
    case 'error':
      return chat.toolStatusError;
    case 'success':
      return chat.toolStatusSuccess || statusLabel;
  }
}
