'use client';

import { useCallback, useEffect, type MutableRefObject } from 'react';
import { getConfig, getProviders } from '@/lib/api/config/api';
import { ignorePromise, toErrorMessage } from '@/lib/errors';
import type { BridgeConfig, ProviderConfig } from '@/lib/types';

interface UseBridgeConfigLoadersOptions {
  autoRefresh: boolean;
  refreshIntervalMs: number;
  mountedRef: MutableRefObject<boolean>;
  loadConfigFallbackMessage: string;
  setConfig: (config: BridgeConfig) => void;
  setProviders: (providers: ProviderConfig[]) => void;
  setConfigLoading: (loading: boolean) => void;
  setModelOptionsLoading: (loading: boolean) => void;
  setConfigError: (error: string) => void;
}

interface UseBridgeConfigLoadersResult {
  loadConfig: (silent?: boolean) => Promise<void>;
  loadProviders: (silent?: boolean) => Promise<void>;
}

export function useBridgeConfigLoaders(
  options: UseBridgeConfigLoadersOptions,
): UseBridgeConfigLoadersResult {
  const {
    autoRefresh,
    refreshIntervalMs,
    mountedRef,
    loadConfigFallbackMessage,
    setConfig,
    setProviders,
    setConfigLoading,
    setModelOptionsLoading,
    setConfigError,
  } = options;

  const loadConfig = useCallback(async (silent = false): Promise<void> => {
    if (!silent) {
      setConfigLoading(true);
    }

    try {
      const loaded = await getConfig();
      if (!mountedRef.current) {
        return;
      }
      setConfig(loaded);
      setConfigError('');
    } catch (error) {
      if (!mountedRef.current || silent) {
        return;
      }
      setConfigError(toErrorMessage(error, loadConfigFallbackMessage));
    } finally {
      if (mountedRef.current && !silent) {
        setConfigLoading(false);
      }
    }
  }, [
    loadConfigFallbackMessage,
    mountedRef,
    setConfig,
    setConfigError,
    setConfigLoading,
  ]);

  const loadProviders = useCallback(async (silent = false): Promise<void> => {
    if (!silent) {
      setModelOptionsLoading(true);
    }

    try {
      const loaded = await getProviders();
      if (!mountedRef.current) {
        return;
      }
      setProviders(loaded.providers);
    } catch {
      if (!mountedRef.current) {
        return;
      }
      if (!silent) {
        setProviders([]);
      }
    } finally {
      if (mountedRef.current && !silent) {
        setModelOptionsLoading(false);
      }
    }
  }, [mountedRef, setModelOptionsLoading, setProviders]);

  useEffect(() => {
    mountedRef.current = true;
    ignorePromise(Promise.all([loadConfig(), loadProviders()]));
    return () => {
      mountedRef.current = false;
    };
  }, [loadConfig, loadProviders, mountedRef]);

  useEffect(() => {
    if (!autoRefresh) {
      return;
    }

    const refresh = () => {
      ignorePromise(Promise.all([loadConfig(true), loadProviders(true)]));
    };
    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible') {
        refresh();
      }
    };

    refresh();
    const intervalId = window.setInterval(refresh, refreshIntervalMs);
    window.addEventListener('focus', refresh);
    document.addEventListener('visibilitychange', handleVisibilityChange);

    return () => {
      window.clearInterval(intervalId);
      window.removeEventListener('focus', refresh);
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
  }, [autoRefresh, loadConfig, loadProviders, refreshIntervalMs]);

  return {
    loadConfig,
    loadProviders,
  };
}
