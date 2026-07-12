function buildTraceIdSuffix(): string {
  const randomUUID = globalThis.crypto?.randomUUID?.();
  if (randomUUID) {
    return randomUUID;
  }

  return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;
}

export function createTraceId(prefix: string): string {
  const normalizedPrefix = prefix.trim() || "shared";
  return `${normalizedPrefix}-${buildTraceIdSuffix()}`;
}
