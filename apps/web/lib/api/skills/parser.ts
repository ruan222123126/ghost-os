import type { SkillPayload } from '@/lib/types';
import {
  expectBoolean,
  expectRecord,
  expectString,
  expectStringEnum,
  pickKnownKeys,
} from '@/lib/api/shared';

const SKILL_PAYLOAD_KEYS = [
  'id',
  'name',
  'description',
  'path',
  'source',
  'enabled',
] as const;
const SKILL_SOURCES = ['repo', 'user'] as const;

export function parseSkillPayload(value: unknown): SkillPayload {
  return parseSkillPayloadWithLabel(value, 'skill');
}

function parseSkillPayloadWithLabel(value: unknown, label: string): SkillPayload {
  const record = pickKnownKeys(expectRecord(value, label), SKILL_PAYLOAD_KEYS);
  return {
    id: expectString(record.id, `${label}.id`),
    name: expectString(record.name, `${label}.name`),
    description: expectString(record.description, `${label}.description`),
    path: expectString(record.path, `${label}.path`),
    source: expectStringEnum(record.source, SKILL_SOURCES, `${label}.source`),
    enabled: expectBoolean(record.enabled, `${label}.enabled`),
  };
}

export function parseSkillPayloadList(payload: unknown): SkillPayload[] {
  if (!Array.isArray(payload)) {
    throw new Error('Invalid skills list: expected array');
  }
  return payload.map((item, index) => parseSkillPayloadWithLabel(item, `skills list[${index}]`));
}
