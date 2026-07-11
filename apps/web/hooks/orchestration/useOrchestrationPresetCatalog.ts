'use client';

import { useEffect, useState } from 'react';
import { listPresets } from '@/lib/api/presets/api';
import { toErrorMessage } from '@/lib/errors';
import type { PresetPayload } from '@/lib/types';

export function useOrchestrationPresetCatalog(loadErrorMessage: string) {
  const [presets, setPresets] = useState<PresetPayload[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      setLoading(true);
      setError('');
      try {
        const next = await listPresets();
        if (!cancelled) {
          setPresets(next);
        }
      } catch (cause) {
        if (!cancelled) {
          setError(toErrorMessage(cause, loadErrorMessage));
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [loadErrorMessage]);

  return { presets, loading, error };
}
