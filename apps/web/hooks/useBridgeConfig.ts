// React hook for reading and updating bridge runtime configuration.

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { getConfig, getProviders, updateConfig } from '@/lib/api/config/api';
import { ignorePromise, toErrorMessage } from '@/lib/errors';
import type { BridgeConfig, ConfigUpdate, ProviderConfig, ProviderModelOption } from '@/lib/types';

const defaultModel = 'gpt-4o';
const defaultRefreshIntervalMs = 3000;

interface UseBridgeConfigOptions {
  autoRefresh?: boolean;
  refreshIntervalMs?: number;
}

interface UseBridgeConfigResult {
  config: BridgeConfig | null;
  configLoading: boolean;
  savingConfig: boolean;
  configError: string;
  modelOptionsLoading: boolean;
  modelValue: string;
  activeModelOption: ProviderModelOption | null;
  modelOptions: ProviderModelOption[];
  saveConfig: (update: ConfigUpdate) => Promise<boolean>;
  selectModel: (model: string) => void;
  selectActiveModel: (option: ProviderModelOption) => Promise<boolean>;
  refreshConfig: () => Promise<void>;
}

export function useBridgeConfig(options: UseBridgeConfigOptions = {}): UseBridgeConfigResult {
  const { autoRefresh = false, refreshIntervalMs = defaultRefreshIntervalMs } = options;
  const [config, setConfig] = useState<BridgeConfig | null>(null);
  const [providers, setProviders] = useState<ProviderConfig[]>([]);
  const [configLoading, setConfigLoading] = useState(true);
  const [modelOptionsLoading, setModelOptionsLoading] = useState(true);
  const [savingConfig, setSavingConfig] = useState(false);
  const [configError, setConfigError] = useState('');
  const mountedRef = useRef(true);

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
      setConfigError(toErrorMessage(error, 'failed to load config'));
    } finally {
      if (mountedRef.current && !silent) {
        setConfigLoading(false);
      }
    }
  }, []);

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
  }, []);

  useEffect(() => {
    mountedRef.current = true;
    ignorePromise(Promise.all([loadConfig(), loadProviders()]));
    return () => {
      mountedRef.current = false;
    };
  }, [loadConfig, loadProviders]);

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

  const saveConfig = useCallback(async (update: ConfigUpdate): Promise<boolean> => {
    setSavingConfig(true);
    setConfigError('');
    try {
      const updated = await updateConfig(update);
      if (!mountedRef.current) {
        return false;
      }
      setConfig(updated);
      return true;
    } catch (error) {
      if (mountedRef.current) {
        setConfigError(toErrorMessage(error, 'failed to save config'));
      }
      return false;
    } finally {
      if (mountedRef.current) {
        setSavingConfig(false);
      }
    }
  }, []);

  const modelSelectionEnabled = config?.model_selection_enabled ?? true;

  const selectModel = useCallback(
    (model: string) => {
      if (!modelSelectionEnabled || !config || model === config.model) {
        return;
      }

      ignorePromise(saveConfig({ model }));
    },
    [config, modelSelectionEnabled, saveConfig],
  );

  const refreshConfig = useCallback(async (): Promise<void> => {
    await Promise.all([loadConfig(), loadProviders()]);
  }, [loadConfig, loadProviders]);

  const modelValue = useMemo(() => config?.model ?? defaultModel, [config]);

  const modelOptions = useMemo(() => buildProviderModelOptions(config, providers), [config, providers]);

  const activeModelOption = useMemo(() => {
    if (!config) {
      return null;
    }

    return modelOptions.find((option) => {
      return stringsEqualIgnoreCase(option.providerName, config.provider)
        && stringsEqualIgnoreCase(option.model, config.model);
    }) ?? {
      providerName: config.provider,
      providerType: config.provider_type,
      model: config.model,
    };
  }, [config, modelOptions]);

  const selectActiveModel = useCallback(async (option: ProviderModelOption): Promise<boolean> => {
    if (!modelSelectionEnabled) {
      return false;
    }
    if (!option.model.trim()) {
      return false;
    }

    if (
      config
      && stringsEqualIgnoreCase(option.providerName, config.provider)
      && stringsEqualIgnoreCase(option.model, config.model)
    ) {
      return true;
    }

    const saved = await saveConfig({ provider: option.providerName, model: option.model });
    if (!saved) {
      return false;
    }

    await loadProviders(true);
    return true;
  }, [config, loadProviders, modelSelectionEnabled, saveConfig]);

  return {
    config,
    configLoading,
    savingConfig,
    configError,
    modelOptionsLoading,
    modelValue,
    activeModelOption,
    modelOptions,
    saveConfig,
    selectModel,
    selectActiveModel,
    refreshConfig,
  };
}

function buildProviderModelOptions(config: BridgeConfig | null, providers: ProviderConfig[]): ProviderModelOption[] {
  const options: ProviderModelOption[] = [];
  const seen = new Set<string>();
  const currentProvider = config?.provider ?? '';
  const orderedProviders = [...providers].sort((left, right) => {
    const leftIsActive = stringsEqualIgnoreCase(left.name, currentProvider);
    const rightIsActive = stringsEqualIgnoreCase(right.name, currentProvider);
    if (leftIsActive === rightIsActive) {
      return 0;
    }
    return leftIsActive ? -1 : 1;
  });

  const pushOption = (provider: ProviderConfig | null, model: string) => {
    const trimmedModel = model.trim();
    if (!trimmedModel) {
      return;
    }

    const providerName = provider?.name ?? config?.provider ?? 'runtime';
    const key = `${providerName.toLowerCase()}::${trimmedModel.toLowerCase()}`;
    if (seen.has(key)) {
      return;
    }
    seen.add(key);
    options.push({
      providerName,
      providerType: provider?.type ?? config?.provider_type ?? 'custom',
      model: trimmedModel,
    });
  };

  for (const provider of orderedProviders) {
    for (const model of provider.models ?? []) {
      pushOption(provider, model);
    }

    // Active runtime model must stay selectable even if the provider card has no explicit model list yet.
    if (config && stringsEqualIgnoreCase(provider.name, config.provider)) {
      pushOption(provider, config.model);
    }
  }

  if (options.length === 0 && config?.model) {
    pushOption(null, config.model);
  }

  return options;
}

function stringsEqualIgnoreCase(left: string, right: string): boolean {
  return left.trim().toLowerCase() === right.trim().toLowerCase();
}
