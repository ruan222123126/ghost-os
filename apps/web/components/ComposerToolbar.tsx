// ComposerToolbar renders safe, non-business toolbar content for the chat composer shell.

'use client';

import { Children } from 'react';
import type { FC, ReactNode } from 'react';

interface ComposerToolbarProps {
  children?: ReactNode;
}

export const ComposerToolbar: FC<ComposerToolbarProps> = ({ children }) => {
  const items = Children.toArray(children);
  if (items.length === 0) {
    return null;
  }

  return <div className="composer-toolbar-slot">{items}</div>;
};

interface ComposerToolbarPillProps {
  label: string;
  tone?: 'default' | 'active' | 'muted' | 'warning';
}

export const ComposerToolbarPill: FC<ComposerToolbarPillProps> = ({ label, tone = 'default' }) => {
  return <span className={`composer-pill composer-pill-${tone}`}>{label}</span>;
};
