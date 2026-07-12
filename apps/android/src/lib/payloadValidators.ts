export function requireRecord(value: unknown, label: string): Record<string, unknown> {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    throw new Error(`${label} must be an object`);
  }
  return value as Record<string, unknown>;
}

export function requireArray(value: unknown, label: string): unknown[] {
  if (!Array.isArray(value)) {
    throw new Error(`${label} must be an array`);
  }
  return value;
}

export function parseArray<T>(
  value: unknown,
  label: string,
  parser: (item: unknown, label: string) => T,
): T[] {
  return requireArray(value, label).map((item, index) => parser(item, `${label}[${index}]`));
}

export function requireString(value: unknown, label: string): string {
  if (typeof value !== "string") {
    throw new Error(`${label} must be a string`);
  }
  return value;
}

export function parseOptionalString(value: unknown, label: string): string | undefined {
  if (value === undefined) {
    return undefined;
  }
  return requireString(value, label);
}

export function requireNumber(value: unknown, label: string): number {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    throw new Error(`${label} must be a number`);
  }
  return value;
}

export function parseOptionalNumber(value: unknown, label: string): number | undefined {
  if (value === undefined) {
    return undefined;
  }
  return requireNumber(value, label);
}

export function requireInteger(value: unknown, label: string): number {
  if (typeof value !== "number" || !Number.isInteger(value)) {
    throw new Error(`${label} must be an integer`);
  }
  return value;
}

export function requireNullableInteger(value: unknown, label: string): number | null {
  if (value === null) {
    return null;
  }
  return requireInteger(value, label);
}

export function requireBoolean(value: unknown, label: string): boolean {
  if (typeof value !== "boolean") {
    throw new Error(`${label} must be a boolean`);
  }
  return value;
}

export function parseOptionalBoolean(value: unknown, label: string): boolean | undefined {
  if (value === undefined) {
    return undefined;
  }
  return requireBoolean(value, label);
}

export function requireStringEnum<T extends string>(
  value: unknown,
  options: Record<T, true>,
  label: string,
): T {
  const text = requireString(value, label);
  if (!Object.prototype.hasOwnProperty.call(options, text)) {
    throw new Error(`${label} has unsupported value: ${text}`);
  }
  return text as T;
}
