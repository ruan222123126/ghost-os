// Hook for loading and regenerating the latest persisted RSS briefing.

'use client';

import { useCallback, useEffect, useState } from 'react';
import { buildRSSBriefing, getRSSBriefing } from '@/lib/api/rss/api';
import { ignorePromise, toErrorMessage } from '@/lib/errors';
import type { RSSBriefing } from '@/lib/types';

interface UseRSSBriefingResult {
  briefing: RSSBriefing | null;
  loading: boolean;
  generating: boolean;
  error: string;
  loadBriefing: () => Promise<void>;
  generateBriefing: () => Promise<void>;
}

export function useRSSBriefing(): UseRSSBriefingResult {
  const [briefing, setBriefing] = useState<RSSBriefing | null>(null);
  const [loading, setLoading] = useState(false);
  const [generating, setGenerating] = useState(false);
  const [error, setError] = useState('');

  const loadBriefing = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const latest = await getRSSBriefing();
      setBriefing(latest);
    } catch (error) {
      setError(toErrorMessage(error));
    } finally {
      setLoading(false);
    }
  }, []);

  const generateBriefing = useCallback(async () => {
    setGenerating(true);
    setError('');
    try {
      const next = await buildRSSBriefing({
        window_hours: 24,
        group_limit: 10,
        item_limit: 200,
        items_per_group: 3,
        highlights_limit: 5,
      });
      setBriefing(next);
    } catch (error) {
      setError(toErrorMessage(error));
    } finally {
      setGenerating(false);
    }
  }, []);

  useEffect(() => {
    ignorePromise(loadBriefing());
  }, [loadBriefing]);

  return {
    briefing,
    loading,
    generating,
    error,
    loadBriefing,
    generateBriefing,
  };
}
