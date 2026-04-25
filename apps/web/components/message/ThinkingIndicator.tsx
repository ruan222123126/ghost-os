import type { FC } from 'react';

const THINKING_PANEL_LABEL = 'Processing Logic';

const CpuIcon: FC<{ className?: string }> = ({ className }) => (
  <svg viewBox="0 0 20 20" fill="none" aria-hidden="true" className={className}>
    <rect x="5.5" y="5.5" width="9" height="9" rx="2" stroke="currentColor" strokeWidth="1.4" />
    <rect x="8" y="8" width="4" height="4" rx="1" fill="currentColor" />
  </svg>
);

export const ThinkingIndicator: FC = () => {
  return (
    <div className="thinking-indicator">
      <CpuIcon className="thinking-indicator-cpu" />
      <span className="thinking-text">{THINKING_PANEL_LABEL}</span>
      <span className="thinking-indicator-status">running...</span>
    </div>
  );
};
