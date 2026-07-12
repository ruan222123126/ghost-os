import type { ApiRequest, CodexModelCatalog } from '@/lib/types';
import { requestJSON } from '@/lib/api/client';
import { createClientTraceId } from '@/lib/api/trace';

export async function getCodexModelCatalog(): Promise<CodexModelCatalog> {
  const body: ApiRequest<Record<string, never>> = {
    action: 'EXTERNAL_AGENT_MODELS_GET',
    params: {},
    trace_id: createClientTraceId('codex-models'),
  };

  return requestJSON('/api/bus', {
    method: 'POST',
    body: JSON.stringify(body),
  }, parseCodexModelCatalog);
}

function parseCodexModelCatalog(payload: unknown): CodexModelCatalog {
  if (!payload || typeof payload !== 'object' || Array.isArray(payload)) {
    throw new Error('Invalid Codex model catalog: expected object');
  }
  const record = payload as Record<string, unknown>;
  if (!Array.isArray(record.models) || !record.models.every(isNonEmptyString)) {
    throw new Error('Invalid Codex model catalog.models');
  }
  if (!isNonEmptyString(record.default_model)) {
    throw new Error('Invalid Codex model catalog.default_model');
  }
  if (!record.models.includes(record.default_model)) {
    throw new Error('Invalid Codex model catalog: default_model is not in models');
  }
  return {
    models: record.models.map((model) => model.trim()),
    default_model: record.default_model.trim(),
  };
}

function isNonEmptyString(value: unknown): value is string {
  return typeof value === 'string' && value.trim().length > 0;
}
