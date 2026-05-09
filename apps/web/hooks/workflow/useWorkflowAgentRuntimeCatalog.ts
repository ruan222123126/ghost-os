'use client';

import { useEffect, useMemo, useState } from 'react';
import { getProviders } from '@/lib/api/config/api';
import { listTools } from '@/lib/api/tools/api';
import { toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { WorkflowAgentRuntimeCatalog } from '@/lib/workflow-editor';

interface WorkflowAgentRuntimeCatalogState {
  catalog?: WorkflowAgentRuntimeCatalog;
  error: string;
  loading: boolean;
}

export function useWorkflowAgentRuntimeCatalog(): WorkflowAgentRuntimeCatalogState {
  const { copy } = useWebLocale();
  const [catalog, setCatalog] = useState<WorkflowAgentRuntimeCatalog>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      setLoading(true);
      setError('');
      try {
        const [providerResponse, tools] = await Promise.all([getProviders(), listTools()]);
        if (cancelled) {
          return;
        }
        setCatalog({
          providers: providerResponse.providers,
          activeProvider: providerResponse.active_provider,
          tools,
        });
      } catch (cause) {
        if (cancelled) {
          return;
        }
        setError(toErrorMessage(cause, copy.system.failedToLoadProviders));
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };
    void load();
    return () => {
      cancelled = true;
    };
  }, [copy.system.failedToLoadProviders]);

  return useMemo(() => ({ catalog, error, loading }), [catalog, error, loading]);
}

