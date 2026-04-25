import type { FC } from 'react';

interface ThinkingPanelProps {
  expanded: boolean;
  text: string;
  onToggleExpanded: () => void;
}

const THINKING_PANEL_LABEL = 'Processing Logic';

const ChevronIcon: FC<{ expanded: boolean; className?: string }> = ({ expanded, className }) => (
  <svg viewBox="0 0 20 20" fill="none" aria-hidden="true" className={className}>
    {expanded ? (
      <path d="M6.2 8.2L10 12l3.8-3.8" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
    ) : (
      <path d="M8.2 6.2L12 10l-3.8 3.8" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
    )}
  </svg>
);

const CpuIcon: FC<{ className?: string }> = ({ className }) => (
  <svg viewBox="0 0 20 20" fill="none" aria-hidden="true" className={className}>
    <rect x="5.5" y="5.5" width="9" height="9" rx="2" stroke="currentColor" strokeWidth="1.4" />
    <rect x="8" y="8" width="4" height="4" rx="1" fill="currentColor" />
  </svg>
);

export const ThinkingPanel: FC<ThinkingPanelProps> = ({
  expanded,
  text,
  onToggleExpanded,
}) => {
  return (
    <div className={`thinking-panel ${expanded ? 'is-expanded' : 'is-collapsed'}`}>
      <button
        type="button"
        className="thinking-panel-toggle"
        aria-expanded={expanded}
        onClick={onToggleExpanded}
      >
        <span className="thinking-panel-heading">
          <ChevronIcon expanded={expanded} className="thinking-panel-chevron" />
          <CpuIcon className="thinking-panel-cpu" />
          <span className="thinking-panel-title">{THINKING_PANEL_LABEL}</span>
        </span>
      </button>
      {expanded ? <pre className="thinking-panel-content">{text}</pre> : null}
    </div>
  );
};
