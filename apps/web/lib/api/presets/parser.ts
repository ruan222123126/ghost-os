import type { PresetPayload, PresetPromptRefs } from '@/lib/types';
import {
  expectRecord,
  expectString,
  expectStringArray,
  parseOptionalStringArray,
  parseOptionalString,
  pickKnownKeys,
} from '@/lib/api/shared';

const PRESET_KEYS = ['id', 'name', 'tool_allowlist', 'prompt_refs'] as const;
const PRESET_PROMPT_REF_KEYS = ['rule', 'core_job', 'memory', 'context'] as const;

function parsePresetPromptRefs(value: unknown, label: string): PresetPromptRefs {
  const record = pickKnownKeys(expectRecord(value, label), PRESET_PROMPT_REF_KEYS);

  return {
    rule: parseOptionalString(record.rule, `${label}.rule`),
    core_job: parseOptionalString(record.core_job, `${label}.core_job`),
    memory: parseOptionalString(record.memory, `${label}.memory`),
    context: parseOptionalStringArray(record.context, `${label}.context`),
  };
}

function parsePresetPayloadWithLabel(value: unknown, label: string): PresetPayload {
  const record = pickKnownKeys(expectRecord(value, label), PRESET_KEYS);

  return {
    id: expectString(record.id, `${label}.id`),
    name: expectString(record.name, `${label}.name`),
    tool_allowlist: expectStringArray(record.tool_allowlist, `${label}.tool_allowlist`),
    prompt_refs: parsePresetPromptRefs(record.prompt_refs, `${label}.prompt_refs`),
  };
}

export function parsePresetPayload(payload: unknown): PresetPayload {
  return parsePresetPayloadWithLabel(payload, 'preset payload');
}

export function parsePresetPayloadList(payload: unknown): PresetPayload[] {
  if (!Array.isArray(payload)) {
    throw new Error('Invalid presets list: expected array');
  }

  return payload.map((item, index) => parsePresetPayloadWithLabel(item, `presets list[${index}]`));
}
