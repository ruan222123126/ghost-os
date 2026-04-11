import type { ToolPayload } from '@/lib/types';
import {
  expectBoolean,
  expectRecord,
  expectString,
  parseOptionalString,
  pickKnownKeys,
} from '@/lib/api/shared';

const TOOL_PAYLOAD_KEYS = [
  'name',
  'enabled',
  'prompt_override',
] as const;

function parseToolPayloadWithLabel(value: unknown, label: string): ToolPayload {
  const record = pickKnownKeys(expectRecord(value, label), TOOL_PAYLOAD_KEYS);
  return {
    name: expectString(record.name, `${label}.name`),
    enabled: expectBoolean(record.enabled, `${label}.enabled`),
    prompt_override: parseOptionalString(record.prompt_override, `${label}.prompt_override`),
  };
}

export function parseToolPayload(payload: unknown): ToolPayload {
  return parseToolPayloadWithLabel(payload, 'tool payload');
}

export function parseToolPayloadList(payload: unknown): ToolPayload[] {
  if (!Array.isArray(payload)) {
    throw new Error('Invalid tools list: expected array');
  }
  return payload.map((item, index) => parseToolPayloadWithLabel(item, `tools list[${index}]`));
}
