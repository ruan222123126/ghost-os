import type { AskHumanOption } from '@/lib/types';

const ASK_HUMAN_OPTION_KEYS = ['label', 'allow_custom'] as const;
const ASK_HUMAN_SELECTION_MODES = ['single', 'multiple'] as const;

export function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

export function ensureKnownKeys(
  value: Record<string, unknown>,
  allowedKeys: readonly string[],
  label: string,
) {
  for (const key of Object.keys(value)) {
    if (!allowedKeys.includes(key)) {
      throw new Error(`Invalid ${label}: unexpected field "${key}"`);
    }
  }
}

export function expectRecord(value: unknown, label: string): Record<string, unknown> {
  if (!isRecord(value)) {
    throw new Error(`Invalid ${label}: expected object`);
  }

  return value;
}

export function expectString(value: unknown, label: string): string {
  if (typeof value !== 'string') {
    throw new Error(`Invalid ${label}: expected string`);
  }

  return value;
}

export function expectNumber(value: unknown, label: string): number {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    throw new Error(`Invalid ${label}: expected number`);
  }

  return value;
}

export function expectBoolean(value: unknown, label: string): boolean {
  if (typeof value !== 'boolean') {
    throw new Error(`Invalid ${label}: expected boolean`);
  }

  return value;
}

export function expectStringEnum<TValue extends string>(
  value: unknown,
  allowedValues: readonly TValue[],
  label: string,
): TValue {
  const parsed = expectString(value, label);
  if (!allowedValues.includes(parsed as TValue)) {
    throw new Error(`Invalid ${label}: unexpected value "${parsed}"`);
  }

  return parsed as TValue;
}

export function expectStringArray(value: unknown, label: string): string[] {
  if (!Array.isArray(value)) {
    throw new Error(`Invalid ${label}: expected array`);
  }

  return value.map((entry, index) => expectString(entry, `${label}[${index}]`));
}

export function parseOptionalString(value: unknown, label: string): string | undefined {
  if (value === undefined) {
    return undefined;
  }

  return expectString(value, label);
}

export function parseOptionalNumber(value: unknown, label: string): number | undefined {
  if (value === undefined) {
    return undefined;
  }

  return expectNumber(value, label);
}

export function parseOptionalBoolean(value: unknown, label: string): boolean | undefined {
  if (value === undefined) {
    return undefined;
  }

  return expectBoolean(value, label);
}

export function parseOptionalRecord(
  value: unknown,
  label: string,
): Record<string, unknown> | undefined {
  if (value === undefined) {
    return undefined;
  }

  return expectRecord(value, label);
}

export function parseOptionalStringArray(value: unknown, label: string): string[] | undefined {
  if (value === undefined) {
    return undefined;
  }

  return expectStringArray(value, label);
}

export function parseOptionalSelectionMode(
  value: unknown,
  label: string,
): 'single' | 'multiple' | undefined {
  if (value === undefined) {
    return undefined;
  }

  return expectStringEnum(value, ASK_HUMAN_SELECTION_MODES, label);
}

export function parseAskHumanOption(value: unknown, label: string): AskHumanOption {
  const record = expectRecord(value, label);
  ensureKnownKeys(record, ASK_HUMAN_OPTION_KEYS, label);

  return {
    label: expectString(record.label, `${label}.label`),
    allow_custom: parseOptionalBoolean(record.allow_custom, `${label}.allow_custom`),
  };
}

export function parseOptionalAskHumanOptions(
  value: unknown,
  label: string,
): AskHumanOption[] | undefined {
  if (value === undefined) {
    return undefined;
  }
  if (!Array.isArray(value)) {
    throw new Error(`Invalid ${label}: expected array`);
  }

  return value.map((entry, index) => parseAskHumanOption(entry, `${label}[${index}]`));
}
