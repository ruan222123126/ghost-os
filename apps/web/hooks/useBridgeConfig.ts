// React hook for reading and updating bridge runtime configuration.

import { useCallback, useMemo, useRef, useState, type MutableRefObject } from 'react';
import { updateConfig } from '@/lib/api/config/api';
import { toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { BridgeConfig, ConfigUpdate, ProviderConfig, ProviderModelOption } from '@/lib/types';
import { useBridgeModelControls } from '@/hooks/bridgeConfigModelControls';
import { useBridgeConfigLoaders } from '@/hooks/useBridgeConfigLoaders';

interface UseBridgeConfigOptions {
  autoRefresh?: boolean;
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

type BridgeConfigCopy = ReturnType<typeof useWebLocale>['copy'];
type BridgeConfigLoaders = ReturnType<typeof useBridgeConfigLoaders>;
type SaveConfigAction = UseBridgeConfigResult['saveConfig'];

interface BridgeConfigStore {
  config: BridgeConfig | null;
  configLoading: boolean;
  configLoadError: string;
  modelOptionsLoading: boolean;
  mountedRef: MutableRefObject<boolean>;
  providerLoadError: string;
  providers: ProviderConfig[];
  saveConfigError: string;
  savingConfig: boolean;
  setConfig: (config: BridgeConfig) => void;
  setConfigLoadError: (error: string) => void;
  setConfigLoading: (loading: boolean) => void;
  setModelOptionsLoading: (loading: boolean) => void;
  setProviderLoadError: (error: string) => void;
  setProviders: (providers: ProviderConfig[]) => void;
  setSaveConfigError: (error: string) => void;
  setSavingConfig: (saving: boolean) => void;
}

interface SaveBridgeConfigOptions {
  copy: BridgeConfigCopy;
  mountedRef: MutableRefObject<boolean>;
  setConfig: (config: BridgeConfig) => void;
  setSaveConfigError: (error: string) => void;
  setSavingConfig: (saving: boolean) => void;
}

export function useBridgeConfig(options: UseBridgeConfigOptions = {}): UseBridgeConfigResult {
  const { copy } = useWebLocale();
  const { autoRefresh = false } = options;
  const store = useBridgeConfigStore();
  const loaders = useBridgeConfigLoaders({
    autoRefresh,
    mountedRef: store.mountedRef,
    loadConfigFallbackMessage: copy.system.failedToLoadConfig,
    loadProvidersFallbackMessage: copy.system.failedToLoadProviders,
    setConfig: store.setConfig,
    setProviders: store.setProviders,
    setConfigLoading: store.setConfigLoading,
    setModelOptionsLoading: store.setModelOptionsLoading,
    setConfigLoadError: store.setConfigLoadError,
    setProviderLoadError: store.setProviderLoadError,
  });
  const configError = useConfigError(store);
  const saveConfig = useSaveBridgeConfig({
    copy,
    mountedRef: store.mountedRef,
    setConfig: store.setConfig,
    setSaveConfigError: store.setSaveConfigError,
    setSavingConfig: store.setSavingConfig,
  });
  const refreshConfig = useRefreshBridgeConfig(loaders);
  const model = useBridgeModelControls({
    config: store.config,
    loadProviders: loaders.loadProviders,
    providers: store.providers,
    saveConfig,
  });

  return {
    config: store.config,
    configLoading: store.configLoading,
    savingConfig: store.savingConfig,
    configError,
    modelOptionsLoading: store.modelOptionsLoading,
    modelValue: model.modelValue,
    activeModelOption: model.activeModelOption,
    modelOptions: model.modelOptions,
    saveConfig,
    selectModel: model.selectModel,
    selectActiveModel: model.selectActiveModel,
    refreshConfig,
  };
}

function useBridgeConfigStore(): BridgeConfigStore {
  const [config, setConfig] = useState<BridgeConfig | null>(null);
  const [providers, setProviders] = useState<ProviderConfig[]>([]);
  const [configLoading, setConfigLoading] = useState(true);
  const [modelOptionsLoading, setModelOptionsLoading] = useState(true);
  const [savingConfig, setSavingConfig] = useState(false);
  const [configLoadError, setConfigLoadError] = useState('');
  const [providerLoadError, setProviderLoadError] = useState('');
  const [saveConfigError, setSaveConfigError] = useState('');
  const mountedRef = useRef(true);

  return {
    config,
    configLoading,
    configLoadError,
    modelOptionsLoading,
    mountedRef,
    providerLoadError,
    providers,
    saveConfigError,
    savingConfig,
    setConfig,
    setConfigLoadError,
    setConfigLoading,
    setModelOptionsLoading,
    setProviderLoadError,
    setProviders,
    setSaveConfigError,
    setSavingConfig,
  };
}

function useConfigError(store: Pick<BridgeConfigStore, 'configLoadError' | 'providerLoadError' | 'saveConfigError'>): string {
  const { configLoadError, providerLoadError, saveConfigError } = store;

  return useMemo(() => {
    if (saveConfigError) {
      return saveConfigError;
    }
    if (configLoadError) {
      return configLoadError;
    }
    return providerLoadError;
  }, [configLoadError, providerLoadError, saveConfigError]);
}

function useSaveBridgeConfig(options: SaveBridgeConfigOptions): SaveConfigAction {
  const { copy, mountedRef, setConfig, setSaveConfigError, setSavingConfig } = options;

  return useCallback(async (update: ConfigUpdate): Promise<boolean> => {
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
  }, [copy.system.failedToSaveConfig, mountedRef, setConfig, setSaveConfigError, setSavingConfig]);
}

function useRefreshBridgeConfig(loaders: BridgeConfigLoaders): UseBridgeConfigResult['refreshConfig'] {
  const { loadConfig, loadProviders } = loaders;

  return useCallback(async (): Promise<void> => {
    await Promise.all([loadConfig(), loadProviders()]);
  }, [loadConfig, loadProviders]);
}
