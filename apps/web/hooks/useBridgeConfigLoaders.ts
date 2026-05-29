'use client';

import { useCallback, useEffect, type MutableRefObject } from 'react';
import { getConfig, getProviders } from '@/lib/api/config/api';
import { ignorePromise, toErrorMessage } from '@/lib/errors';
import type { BridgeConfig, ProviderConfig } from '@/lib/types';

interface UseBridgeConfigLoadersOptions {
  autoRefresh: boolean;
  mountedRef: MutableRefObject<boolean>;
  loadConfigFallbackMessage: string;
  loadProvidersFallbackMessage: string;
  setConfig: (config: BridgeConfig) => void;
  setProviders: (providers: ProviderConfig[]) => void;
  setConfigLoading: (loading: boolean) => void;
  setModelOptionsLoading: (loading: boolean) => void;
  setConfigLoadError: (error: string) => void;
  setProviderLoadError: (error: string) => void;
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
    mountedRef,
    loadConfigFallbackMessage,
    loadProvidersFallbackMessage,
    setConfig,
    setProviders,
    setConfigLoading,
    setModelOptionsLoading,
    setConfigLoadError,
    setProviderLoadError,
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
      setConfigLoadError('');
    } catch (error) {
      if (!mountedRef.current || silent) {
        return;
      }
      setConfigLoadError(toErrorMessage(error, loadConfigFallbackMessage));
    } finally {
      if (mountedRef.current && !silent) {
        setConfigLoading(false);
      }
    }
  }, [
    loadConfigFallbackMessage,
    mountedRef,
    setConfig,
    setConfigLoadError,
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
      setProviderLoadError('');
    } catch (error) {
      if (!mountedRef.current || silent) {
        return;
      }
      setProviderLoadError(toErrorMessage(error, loadProvidersFallbackMessage));
    } finally {
      if (mountedRef.current && !silent) {
        setModelOptionsLoading(false);
      }
    }
  }, [
    loadProvidersFallbackMessage,
    mountedRef,
    setModelOptionsLoading,
    setProviderLoadError,
    setProviders,
  ]);

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

    window.addEventListener('focus', refresh);
    document.addEventListener('visibilitychange', handleVisibilityChange);

    return () => {
      window.removeEventListener('focus', refresh);
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
  }, [autoRefresh, loadConfig, loadProviders]);

  return {
    loadConfig,
    loadProviders,
  };
}
