import type {
  BridgeConfig,
  ConfigUpdate,
  ProviderConfigInput,
  ProviderListResponse,
  SetActiveProviderRequest,
} from '@/lib/types';
import { requestJSON } from '@/lib/api/client';
import { parseBridgeConfig, parseProviderListResponse } from '@/lib/api/config/parser';

export async function getConfig(): Promise<BridgeConfig> {
  return requestJSON('/api/config', {}, parseBridgeConfig);
}

export async function updateConfig(update: ConfigUpdate): Promise<BridgeConfig> {
  return requestJSON('/api/config', {
    method: 'POST',
    body: JSON.stringify(update),
  }, parseBridgeConfig);
}

export async function getProviders(): Promise<ProviderListResponse> {
  return requestJSON('/api/config/providers', {}, parseProviderListResponse);
}

export async function createProvider(input: ProviderConfigInput): Promise<ProviderListResponse> {
  return requestJSON('/api/config/providers', {
    method: 'POST',
    body: JSON.stringify(input),
  }, parseProviderListResponse);
}

export async function updateProvider(
  name: string,
  input: ProviderConfigInput,
): Promise<ProviderListResponse> {
  return requestJSON(`/api/config/providers/${encodeURIComponent(name)}`, {
    method: 'PUT',
    body: JSON.stringify(input),
  }, parseProviderListResponse);
}

export async function deleteProvider(name: string): Promise<ProviderListResponse> {
  return requestJSON(`/api/config/providers/${encodeURIComponent(name)}`, {
    method: 'DELETE',
  }, parseProviderListResponse);
}

export async function setActiveProvider(name: string): Promise<ProviderListResponse> {
  const body: SetActiveProviderRequest = { name };

  return requestJSON('/api/config/active-provider', {
    method: 'PUT',
    body: JSON.stringify(body),
  }, parseProviderListResponse);
}
