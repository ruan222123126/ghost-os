'use client';

import { useEffect, useMemo, useState, type Dispatch, type SetStateAction } from 'react';
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

interface WorkflowAgentRuntimeCatalogSetters {
  setCatalog: Dispatch<SetStateAction<WorkflowAgentRuntimeCatalog | undefined>>;
  setError: Dispatch<SetStateAction<string>>;
  setLoading: Dispatch<SetStateAction<boolean>>;
}

interface CatalogLoadLifecycle {
  cancelled: boolean;
}

interface WorkflowAgentRuntimeCatalogLoadOptions extends WorkflowAgentRuntimeCatalogSetters {
  fallbackError: string;
  lifecycle: CatalogLoadLifecycle;
}

export function useWorkflowAgentRuntimeCatalog(): WorkflowAgentRuntimeCatalogState {
  const { copy } = useWebLocale();
  const [catalog, setCatalog] = useState<WorkflowAgentRuntimeCatalog>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useWorkflowAgentRuntimeCatalogLoad(copy.system.failedToLoadProviders, {
    setCatalog,
    setError,
    setLoading,
  });

  return useMemo(() => ({ catalog, error, loading }), [catalog, error, loading]);
}

function useWorkflowAgentRuntimeCatalogLoad(
  fallbackError: string,
  setters: WorkflowAgentRuntimeCatalogSetters,
): void {
  const { setCatalog, setError, setLoading } = setters;

  useEffect(() => {
    const lifecycle: CatalogLoadLifecycle = { cancelled: false };
    void runWorkflowAgentRuntimeCatalogLoad({
      fallbackError,
      lifecycle,
      setCatalog,
      setError,
      setLoading,
    });
    return () => {
      lifecycle.cancelled = true;
    };
  }, [fallbackError, setCatalog, setError, setLoading]);
}

export async function loadWorkflowAgentRuntimeCatalog(): Promise<WorkflowAgentRuntimeCatalog> {
  const [providerResponse, tools] = await Promise.all([getProviders(), listTools()]);

  return {
    providers: providerResponse.providers,
    activeProvider: providerResponse.active_provider,
    tools,
  };
}

async function runWorkflowAgentRuntimeCatalogLoad(
  options: WorkflowAgentRuntimeCatalogLoadOptions,
): Promise<void> {
  const { setError, setLoading } = options;
  setLoading(true);
  setError('');

  try {
    commitCatalogLoadSuccess(options, await loadWorkflowAgentRuntimeCatalog());
  } catch (cause) {
    commitCatalogLoadError(options, cause);
  } finally {
    commitCatalogLoadFinished(options);
  }
}

function commitCatalogLoadSuccess(
  options: WorkflowAgentRuntimeCatalogLoadOptions,
  catalog: WorkflowAgentRuntimeCatalog,
): void {
  if (options.lifecycle.cancelled) {
    return;
  }
  options.setCatalog(catalog);
}

function commitCatalogLoadError(options: WorkflowAgentRuntimeCatalogLoadOptions, cause: unknown): void {
  if (options.lifecycle.cancelled) {
    return;
  }
  options.setError(toErrorMessage(cause, options.fallbackError));
}

function commitCatalogLoadFinished(options: WorkflowAgentRuntimeCatalogLoadOptions): void {
  if (!options.lifecycle.cancelled) {
    options.setLoading(false);
  }
}
