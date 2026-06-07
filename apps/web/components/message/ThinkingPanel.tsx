import type { FC } from 'react';
import { useWebLocale } from '@/lib/i18n/provider';

interface ThinkingPanelProps {
  active: boolean;
  expanded: boolean;
  text: string;
  onToggleExpanded: () => void;
}

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

export const ThinkingPanel: FC<ThinkingPanelProps> = ({
  active,
  expanded,
  text,
  onToggleExpanded,
}) => {
  const { copy } = useWebLocale();
  const title = active ? copy.chat.thinkingPanelTitle : copy.chat.thinkingPanelDoneTitle;
  const titleClassName = [
    'thinking-panel-title',
    active ? 'thinking-sweep-text' : '',
  ]
    .filter(Boolean)
    .join(' ');
  const panelClassName = [
    'thinking-panel',
    expanded ? 'is-expanded' : 'is-collapsed',
    active ? 'is-active' : 'is-complete',
  ].join(' ');

  return (
    <div className={panelClassName}>
      <button
        type="button"
        className="thinking-panel-toggle"
        aria-expanded={expanded}
        aria-busy={active}
        onClick={onToggleExpanded}
      >
        <span className="thinking-panel-heading">
          <TerminalIcon className="thinking-panel-terminal" />
          <span className={titleClassName}>{title}</span>
          <ChevronIcon expanded={expanded} className="thinking-panel-chevron" />
        </span>
      </button>
      {expanded ? <pre className="thinking-panel-content">{text}</pre> : null}
    </div>
  );
};
