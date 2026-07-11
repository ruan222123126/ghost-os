'use client';

import { useMemo } from 'react';
import {
  createProvider as createProviderRequest,
  deleteProvider as deleteProviderRequest,
  getProviders as getProvidersRequest,
  updateProvider as updateProviderRequest,
} from '@/lib/api/config/api';
import type { ConfigUpdate } from '@/lib/types';

interface UseProvidersApiOptions {
  onReloadConfig: () => Promise<void>;
  onActivateRuntimeConfig: (update: ConfigUpdate) => Promise<boolean>;
  modelSelectionEnabled: boolean;
}

export function useProvidersApi(options: UseProvidersApiOptions) {
  const { onReloadConfig, onActivateRuntimeConfig, modelSelectionEnabled } = options;

  return useMemo(() => ({
    getProviders: getProvidersRequest,
    createProvider: createProviderRequest,
    updateProvider: updateProviderRequest,
    deleteProvider: deleteProviderRequest,
    reloadConfig: onReloadConfig,
    activateRuntimeConfig: onActivateRuntimeConfig,
    modelSelectionEnabled,
  }), [modelSelectionEnabled, onActivateRuntimeConfig, onReloadConfig]);
}
