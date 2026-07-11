export type RecordValue = Record<string, unknown>;

export function parseRecord(rawText?: string): RecordValue | undefined {
  if (!rawText || !rawText.trim()) return undefined;
  try {
    const parsed: unknown = JSON.parse(rawText);
    return isRecord(parsed) ? parsed : undefined;
  } catch {
    return undefined;
  }
}

export function readRecord(value: RecordValue | undefined, key: string): RecordValue | undefined {
  const candidate = value?.[key];
  return isRecord(candidate) ? candidate : undefined;
}

export function readString(value: RecordValue | undefined, key: string): string {
  const candidate = value?.[key];
  return typeof candidate === 'string' ? candidate.trim() : '';
}

export function readPositiveInt(value: RecordValue | undefined, key: string): number | undefined {
  const candidate = value?.[key];
  if (typeof candidate === 'number' && Number.isFinite(candidate) && candidate > 0) {
    return Math.floor(candidate);
  }
  if (typeof candidate !== 'string') return undefined;
  const parsed = Number.parseInt(candidate, 10);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : undefined;
}

export function readRecordArray(value: RecordValue, key: string): RecordValue[] | undefined {
  const list = value[key];
  if (!Array.isArray(list)) return undefined;
  return list.filter((item): item is RecordValue => isRecord(item));
}

function isRecord(value: unknown): value is RecordValue {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

