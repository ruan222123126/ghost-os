import type {
  PresetCreateRequest,
  PresetPayload,
  PresetUpdateRequest,
} from '@/lib/types';
import { requestJSON } from '@/lib/api/client';
import { parsePresetPayload, parsePresetPayloadList } from '@/lib/api/presets/parser';

export async function listPresets(): Promise<PresetPayload[]> {
  return requestJSON('/api/presets', {}, parsePresetPayloadList);
}

export async function createPreset(input: PresetCreateRequest): Promise<PresetPayload> {
  return requestJSON('/api/presets', {
    method: 'POST',
    body: JSON.stringify(input),
  }, parsePresetPayload);
}

export async function updatePreset(
  id: string,
  input: PresetUpdateRequest,
): Promise<PresetPayload> {
  return requestJSON(`/api/presets/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    body: JSON.stringify(input),
  }, parsePresetPayload);
}

export async function deletePreset(id: string): Promise<PresetPayload> {
  return requestJSON(`/api/presets/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  }, parsePresetPayload);
}

export async function activatePreset(id: string): Promise<PresetPayload> {
  return requestJSON(`/api/presets/${encodeURIComponent(id)}/activate`, {
    method: 'PUT',
  }, parsePresetPayload);
}
