import { useCallback, useEffect, useMemo, useState } from 'react';
import { getConfig, updateConfig } from '@/lib/api';
import { ignorePromise, toErrorMessage } from '@/lib/errors';
import type { BridgeConfig, ConfigUpdate } from '@/lib/types';

const defaultModel = 'gpt-4o';

interface UseBridgeConfigResult {
  config: BridgeConfig | null;
  configLoading: boolean;
  savingConfig: boolean;
  configError: string;
  modelValue: string;
  saveConfig: (update: ConfigUpdate) => Promise<boolean>;
  selectModel: (model: string) => void;
}

export function useBridgeConfig(): UseBridgeConfigResult {
  const [config, setConfig] = useState<BridgeConfig | null>(null);
  const [configLoading, setConfigLoading] = useState(true);
  const [savingConfig, setSavingConfig] = useState(false);
  const [configError, setConfigError] = useState('');

  useEffect(() => {
    let active = true;

    async function loadConfig() {
      try {
        const loaded = await getConfig();
        if (!active) {
          return;
        }
        setConfig(loaded);
      } catch (error) {
        if (!active) {
          return;
        }
        setConfigError(toErrorMessage(error, 'failed to load config'));
      } finally {
        if (active) {
          setConfigLoading(false);
        }
      }
    }

    ignorePromise(loadConfig());
    return () => {
      active = false;
    };
  }, []);

  const saveConfig = useCallback(async (update: ConfigUpdate): Promise<boolean> => {
    setSavingConfig(true);
    setConfigError('');
    try {
      const updated = await updateConfig(update);
      setConfig(updated);
      return true;
    } catch (error) {
      setConfigError(toErrorMessage(error, 'failed to save config'));
      return false;
    } finally {
      setSavingConfig(false);
    }
  }, []);

  const selectModel = useCallback(
    (model: string) => {
      if (!config || model === config.model) {
        return;
      }

      ignorePromise(saveConfig({ model }));
    },
    [config, saveConfig],
  );

  const modelValue = useMemo(() => config?.model ?? defaultModel, [config]);

  return {
    config,
    configLoading,
    savingConfig,
    configError,
    modelValue,
    saveConfig,
    selectModel,
  };
}
