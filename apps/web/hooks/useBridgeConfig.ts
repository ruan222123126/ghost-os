// React hook for reading and updating bridge runtime configuration.

import { useCallback, useMemo, useRef, useState } from 'react';
import { updateConfig } from '@/lib/api/config/api';
import { toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { BridgeConfig, ConfigUpdate, ProviderConfig, ProviderModelOption } from '@/lib/types';
import {
  buildProviderModelOptions,
  resolveActiveModelOption,
  stringsEqualIgnoreCase,
} from '@/hooks/bridgeConfigModelOptions';
import { useBridgeConfigLoaders } from '@/hooks/useBridgeConfigLoaders';

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
  const { copy } = useWebLocale();
  const { autoRefresh = false, refreshIntervalMs = defaultRefreshIntervalMs } = options;
  const [config, setConfig] = useState<BridgeConfig | null>(null);
  const [providers, setProviders] = useState<ProviderConfig[]>([]);
  const [configLoading, setConfigLoading] = useState(true);
  const [modelOptionsLoading, setModelOptionsLoading] = useState(true);
  const [savingConfig, setSavingConfig] = useState(false);
  const [configLoadError, setConfigLoadError] = useState('');
  const [providerLoadError, setProviderLoadError] = useState('');
  const [saveConfigError, setSaveConfigError] = useState('');
  const mountedRef = useRef(true);
  const loaders = useBridgeConfigLoaders({
    autoRefresh,
    refreshIntervalMs,
    mountedRef,
    loadConfigFallbackMessage: copy.system.failedToLoadConfig,
    loadProvidersFallbackMessage: copy.system.failedToLoadProviders,
    setConfig,
    setProviders,
    setConfigLoading,
    setModelOptionsLoading,
    setConfigLoadError,
    setProviderLoadError,
  });

  const configError = useMemo(() => {
    if (saveConfigError) {
      return saveConfigError;
    }
    if (configLoadError) {
      return configLoadError;
    }
    return providerLoadError;
  }, [configLoadError, providerLoadError, saveConfigError]);

  const saveConfig = useCallback(async (update: ConfigUpdate): Promise<boolean> => {
    setSavingConfig(true);
    setSaveConfigError('');
    try {
      const updated = await updateConfig(update);
      if (!mountedRef.current) {
        return false;
      }
      setConfig(updated);
      return true;
    } catch (error) {
      if (mountedRef.current) {
        setSaveConfigError(toErrorMessage(error, copy.system.failedToSaveConfig));
      }
      return false;
    } finally {
      if (mountedRef.current) {
        setSavingConfig(false);
      }
    }
  }, [copy.system.failedToSaveConfig]);

  const modelSelectionEnabled = config?.model_selection_enabled ?? true;

  const selectModel = useCallback((model: string) => {
    if (!modelSelectionEnabled || !config || model === config.model) {
      return;
    }
    void saveConfig({ model });
  }, [config, modelSelectionEnabled, saveConfig]);

  const refreshConfig = useCallback(async (): Promise<void> => {
    await Promise.all([loaders.loadConfig(), loaders.loadProviders()]);
  }, [loaders]);

  const modelValue = useMemo(() => config?.model ?? defaultModel, [config]);
  const modelOptions = useMemo(() => buildProviderModelOptions(config, providers), [config, providers]);
  const activeModelOption = useMemo(
    () => resolveActiveModelOption(config, modelOptions),
    [config, modelOptions],
  );

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

    await loaders.loadProviders(true);
    return true;
  }, [config, loaders, modelSelectionEnabled, saveConfig]);

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
