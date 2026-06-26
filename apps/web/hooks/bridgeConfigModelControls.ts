import { useCallback, useMemo } from 'react';
import type { BridgeConfig, ConfigUpdate, ProviderConfig, ProviderModelOption } from '@/lib/types';
import {
  buildProviderModelOptions,
  resolveActiveModelOption,
  stringsEqualIgnoreCase,
} from '@/hooks/bridgeConfigModelOptions';

const defaultModel = 'gpt-4o';

type SaveConfigAction = (update: ConfigUpdate) => Promise<boolean>;

interface BridgeModelControls {
  activeModelOption: ProviderModelOption | null;
  modelOptions: ProviderModelOption[];
  modelValue: string;
  selectActiveModel: (option: ProviderModelOption) => Promise<boolean>;
  selectModel: (model: string) => void;
}

interface BridgeModelControlOptions {
  config: BridgeConfig | null;
  loadProviders: (silent?: boolean) => Promise<void>;
  providers: ProviderConfig[];
  saveConfig: SaveConfigAction;
}

interface BridgeModelState {
  activeModelOption: ProviderModelOption | null;
  modelOptions: ProviderModelOption[];
  modelSelectionEnabled: boolean;
  modelValue: string;
}

interface BridgeModelActionOptions {
  config: BridgeConfig | null;
  loadProviders: BridgeModelControlOptions['loadProviders'];
  modelSelectionEnabled: boolean;
  saveConfig: SaveConfigAction;
}

export function useBridgeModelControls(options: BridgeModelControlOptions): BridgeModelControls {
  const { config, loadProviders, providers, saveConfig } = options;
  const state = useBridgeModelState(config, providers);
  const actions = useBridgeModelActions({
    config,
    loadProviders,
    modelSelectionEnabled: state.modelSelectionEnabled,
    saveConfig,
  });

  return {
    activeModelOption: state.activeModelOption,
    modelOptions: state.modelOptions,
    modelValue: state.modelValue,
    selectActiveModel: actions.selectActiveModel,
    selectModel: actions.selectModel,
  };
}

function useBridgeModelState(
  config: BridgeConfig | null,
  providers: ProviderConfig[],
): BridgeModelState {
  const modelSelectionEnabled = config?.model_selection_enabled ?? true;
  const modelValue = useMemo(() => config?.model ?? defaultModel, [config]);
  const modelOptions = useMemo(() => buildProviderModelOptions(config, providers), [config, providers]);
  const activeModelOption = useMemo(
    () => resolveActiveModelOption(config, modelOptions),
    [config, modelOptions],
  );

  return {
    activeModelOption,
    modelOptions,
    modelSelectionEnabled,
    modelValue,
  };
}

function useBridgeModelActions(options: BridgeModelActionOptions): Pick<BridgeModelControls, 'selectActiveModel' | 'selectModel'> {
  const { config, loadProviders, modelSelectionEnabled, saveConfig } = options;
  const selectModel = useCallback((model: string) => {
    selectBridgeModel({ config, model, modelSelectionEnabled, saveConfig });
  }, [config, modelSelectionEnabled, saveConfig]);

  const selectActiveModel = useCallback(async (option: ProviderModelOption): Promise<boolean> => {
    return selectBridgeActiveModel({
      config,
      loadProviders,
      modelSelectionEnabled,
      option,
      saveConfig,
    });
  }, [config, loadProviders, modelSelectionEnabled, saveConfig]);

  return {
    selectModel,
    selectActiveModel,
  };
}

function selectBridgeModel(options: {
  config: BridgeConfig | null;
  model: string;
  modelSelectionEnabled: boolean;
  saveConfig: SaveConfigAction;
}): void {
  const { config, model, modelSelectionEnabled, saveConfig } = options;

  if (!modelSelectionEnabled || !config || model === config.model) {
    return;
  }
  void saveConfig({ model });
}

async function selectBridgeActiveModel(options: {
  config: BridgeConfig | null;
  loadProviders: BridgeModelControlOptions['loadProviders'];
  modelSelectionEnabled: boolean;
  option: ProviderModelOption;
  saveConfig: SaveConfigAction;
}): Promise<boolean> {
  const { config, loadProviders, modelSelectionEnabled, option, saveConfig } = options;

  if (!modelSelectionEnabled || !option.model.trim()) {
    return false;
  }
  if (isCurrentProviderModel(config, option)) {
    return true;
  }

  const saved = await saveConfig({ provider: option.providerName, model: option.model });
  if (!saved) {
    return false;
  }

  await loadProviders(true);
  return true;
}

function isCurrentProviderModel(config: BridgeConfig | null, option: ProviderModelOption): boolean {
  if (!config) {
    return false;
  }
  return stringsEqualIgnoreCase(option.providerName, config.provider)
    && stringsEqualIgnoreCase(option.model, config.model);
}
