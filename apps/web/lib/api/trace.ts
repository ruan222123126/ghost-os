function buildTraceIdSuffix(): string {
  const randomUUID = globalThis.crypto?.randomUUID?.();
  if (randomUUID) {
    return randomUUID;
  }

  return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;
}

export function createClientTraceId(prefix = 'web'): string {
  const normalizedPrefix = prefix.trim() || 'web';
  return `${normalizedPrefix}-${buildTraceIdSuffix()}`;
}
