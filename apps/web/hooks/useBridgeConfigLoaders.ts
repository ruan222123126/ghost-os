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

interface MountedResourceLoaderOptions<T> {
  fallbackMessage: string;
  load: () => Promise<T>;
  mountedRef: MutableRefObject<boolean>;
  setError: (error: string) => void;
  setLoading: (loading: boolean) => void;
  silent: boolean;
  store: (value: T) => void;
}

export function useBridgeConfigLoaders(
  options: UseBridgeConfigLoadersOptions,
): UseBridgeConfigLoadersResult {
  const loadConfig = useConfigLoader(options);
  const loadProviders = useProvidersLoader(options);

  useInitialBridgeConfigLoad(options.mountedRef, loadConfig, loadProviders);
  useAutoRefreshBridgeConfig(options.autoRefresh, loadConfig, loadProviders);

  return {
    loadConfig,
    loadProviders,
  };
}

function useConfigLoader(
  options: UseBridgeConfigLoadersOptions,
): UseBridgeConfigLoadersResult['loadConfig'] {
  const {
    mountedRef,
    loadConfigFallbackMessage,
    setConfig,
    setConfigLoading,
    setConfigLoadError,
  } = options;

  return useCallback(async (silent = false): Promise<void> => {
    await loadMountedResource({
      fallbackMessage: loadConfigFallbackMessage,
      load: getConfig,
      mountedRef,
      setError: setConfigLoadError,
      setLoading: setConfigLoading,
      silent,
      store: setConfig,
    });
  }, [
    loadConfigFallbackMessage,
    mountedRef,
    setConfig,
    setConfigLoadError,
    setConfigLoading,
  ]);
}

function useProvidersLoader(
  options: UseBridgeConfigLoadersOptions,
): UseBridgeConfigLoadersResult['loadProviders'] {
  const {
    mountedRef,
    loadProvidersFallbackMessage,
    setProviders,
    setModelOptionsLoading,
    setProviderLoadError,
  } = options;

  return useCallback(async (silent = false): Promise<void> => {
    await loadMountedResource({
      fallbackMessage: loadProvidersFallbackMessage,
      load: getProviders,
      mountedRef,
      setError: setProviderLoadError,
      setLoading: setModelOptionsLoading,
      silent,
      store: (loaded) => setProviders(loaded.providers),
    });
  }, [
    loadProvidersFallbackMessage,
    mountedRef,
    setModelOptionsLoading,
    setProviderLoadError,
    setProviders,
  ]);
}

function useInitialBridgeConfigLoad(
  mountedRef: MutableRefObject<boolean>,
  loadConfig: UseBridgeConfigLoadersResult['loadConfig'],
  loadProviders: UseBridgeConfigLoadersResult['loadProviders'],
): void {
  useEffect(() => {
    mountedRef.current = true;
    ignorePromise(loadBridgeConfigResources(loadConfig, loadProviders));
    return () => {
      mountedRef.current = false;
    };
  }, [loadConfig, loadProviders, mountedRef]);
}

function useAutoRefreshBridgeConfig(
  autoRefresh: boolean,
  loadConfig: UseBridgeConfigLoadersResult['loadConfig'],
  loadProviders: UseBridgeConfigLoadersResult['loadProviders'],
): void {
  useEffect(() => {
    if (!autoRefresh) {
      return;
    }

    const refresh = () => {
      ignorePromise(loadBridgeConfigResources(loadConfig, loadProviders, true));
    };
    const handleVisibilityChange = () => {
      refreshWhenVisible(refresh);
    };

    window.addEventListener('focus', refresh);
    document.addEventListener('visibilitychange', handleVisibilityChange);

    return () => {
      window.removeEventListener('focus', refresh);
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
  }, [autoRefresh, loadConfig, loadProviders]);
}

function loadBridgeConfigResources(
  loadConfig: UseBridgeConfigLoadersResult['loadConfig'],
  loadProviders: UseBridgeConfigLoadersResult['loadProviders'],
  silent = false,
): Promise<void[]> {
  return Promise.all([loadConfig(silent), loadProviders(silent)]);
}

function refreshWhenVisible(refresh: () => void): void {
  if (document.visibilityState === 'visible') {
    refresh();
  }
}

async function loadMountedResource<T>(options: MountedResourceLoaderOptions<T>): Promise<void> {
  const { fallbackMessage, load, mountedRef, setError, setLoading, silent, store } = options;

  if (!silent) {
    setLoading(true);
  }

  try {
    const loaded = await load();
    if (!mountedRef.current) {
      return;
    }
    store(loaded);
    setError('');
  } catch (error) {
    if (!mountedRef.current || silent) {
      return;
    }
    setError(toErrorMessage(error, fallbackMessage));
  } finally {
    if (mountedRef.current && !silent) {
      setLoading(false);
    }
  }
}
