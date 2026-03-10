// ComposerMetaRow renders hint and status for the chat composer.

'use client';

import type { FC, ReactNode } from 'react';

interface ComposerMetaRowProps {
  hint?: ReactNode;
  status?: ReactNode;
}

export const ComposerMetaRow: FC<ComposerMetaRowProps> = ({ hint, status }) => {
  const hasSideContent = status !== undefined && status !== null;

  return (
    <div className="composer-meta">
      <span className="field-hint">{hint}</span>
      {hasSideContent ? (
        <div className="composer-meta-side">
          <span className="composer-status">{status}</span>
        </div>
      ) : null}
    </div>
  );
};
