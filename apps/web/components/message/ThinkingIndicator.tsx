import type { FC } from 'react';

export const ThinkingIndicator: FC = () => (
  <div className="thinking-indicator">
    <div className="thinking-dots" aria-hidden="true">
      <span className="thinking-dot" />
      <span className="thinking-dot" />
      <span className="thinking-dot" />
    </div>
    <span className="thinking-text">Thinking...</span>
  </div>
);
