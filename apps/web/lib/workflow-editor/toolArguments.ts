export type ToolArgumentValueType = 'string' | 'number' | 'boolean' | 'object' | 'array' | 'null';

export interface WorkflowToolArgumentRow {
  key: string;
  valueType: ToolArgumentValueType;
  value: string;
}

const EMPTY_OBJECT_TEXT = '{}';

export function toolArgumentsToJSONText(args?: Record<string, unknown>): string {
  if (!args || Object.keys(args).length === 0) {
    return EMPTY_OBJECT_TEXT;
  }

  return JSON.stringify(args, null, 2);
}

export function jsonTextToToolArguments(raw: string): Record<string, unknown> {
  const text = raw.trim();
  if (text.length === 0) {
    return {};
  }
  const parsed = JSON.parse(text) as unknown;
  if (!isPlainObject(parsed)) {
    throw new Error('tool arguments JSON must be an object');
  }

  return parsed;
}

export function toolArgumentsToRows(args?: Record<string, unknown>): WorkflowToolArgumentRow[] {
  if (!args || Object.keys(args).length === 0) {
    return [];
  }

  return Object.entries(args).map(([key, value]) => {
    const valueType = inferValueType(value);
    return {
      key,
      valueType,
      value: serializeValue(valueType, value),
    };
  });
}

export function rowsToToolArguments(rows: WorkflowToolArgumentRow[]): Record<string, unknown> {
  const output: Record<string, unknown> = {};

  for (let index = 0; index < rows.length; index += 1) {
    const row = rows[index];
    const key = row.key.trim();
    if (key.length === 0) {
      throw new Error(`tool argument row[${index}] key is required`);
    }
    if (Object.prototype.hasOwnProperty.call(output, key)) {
      throw new Error(`duplicate tool argument key "${key}"`);
    }

    output[key] = parseRowValue(row, index);
  }

  return output;
}

function parseRowValue(row: WorkflowToolArgumentRow, index: number): unknown {
  switch (row.valueType) {
    case 'string':
      return row.value;
    case 'number':
      return parseNumber(row.value, index);
    case 'boolean':
      return parseBoolean(row.value, index);
    case 'object':
      return parseJSONObject(row.value, index);
    case 'array':
      return parseJSONArray(row.value, index);
    case 'null':
      return null;
    default:
      throw new Error(`unsupported tool argument type at row[${index}]`);
  }
}

function parseNumber(raw: string, index: number): unknown {
  const reference = parseFindIconReference(raw);
  if (reference) {
    return reference;
  }
  const value = Number(raw.trim());
  if (!Number.isFinite(value)) {
    throw new Error(`tool argument row[${index}] number value is invalid`);
  }

  return value;
}

function parseBoolean(raw: string, index: number): unknown {
  const reference = parseFindIconReference(raw);
  if (reference) {
    return reference;
  }
  const normalized = raw.trim().toLowerCase();
  if (normalized === 'true') {
    return true;
  }
  if (normalized === 'false') {
    return false;
  }
  throw new Error(`tool argument row[${index}] boolean value must be true or false`);
}

function parseJSONObject(raw: string, index: number): unknown {
  const reference = parseFindIconReference(raw);
  if (reference) {
    return reference;
  }
  const parsed = JSON.parse(raw) as unknown;
  if (!isPlainObject(parsed)) {
    throw new Error(`tool argument row[${index}] object value must be valid JSON object`);
  }

  return parsed;
}

function parseJSONArray(raw: string, index: number): unknown {
  const reference = parseFindIconReference(raw);
  if (reference) {
    return reference;
  }
  const parsed = JSON.parse(raw) as unknown;
  if (!Array.isArray(parsed)) {
    throw new Error(`tool argument row[${index}] array value must be valid JSON array`);
  }

  return parsed;
}

function inferValueType(value: unknown): ToolArgumentValueType {
  if (value === null) {
    return 'null';
  }
  if (Array.isArray(value)) {
    return 'array';
  }

  switch (typeof value) {
    case 'string':
      return 'string';
    case 'number':
      return 'number';
    case 'boolean':
      return 'boolean';
    default:
      return 'object';
  }
}

function serializeValue(valueType: ToolArgumentValueType, value: unknown): string {
  if (valueType === 'null') {
    return '';
  }
  if (typeof value === 'string' && isFindIconReference(value)) {
    return value;
  }
  if (valueType === 'object' || valueType === 'array') {
    return JSON.stringify(value, null, 2);
  }

  return String(value);
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function parseFindIconReference(raw: string): string | undefined {
  const trimmed = raw.trim();
  return isFindIconReference(trimmed) ? trimmed : undefined;
}

function isFindIconReference(value: string): boolean {
  return /^\$\{find_icon(?:\.[A-Za-z0-9_-]+)*\}$/.test(value.trim());
}
