'use client';

import type { FC } from 'react';
import { useWebLocale } from '@/lib/i18n/provider';

export const ChatEmptyHero: FC = () => {
  const { copy } = useWebLocale();

  return (
    <div className="chat-empty-hero">
      <h1 className="chat-empty-hero-title">{copy.chat.emptyHeroTitle}</h1>
      <p className="chat-empty-hero-subtitle">{copy.chat.emptyHeroSubtitle}</p>
    </div>
  );
};
