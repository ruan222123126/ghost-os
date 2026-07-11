export interface BridgeBusCommand {
  baseUrl: string;
  apiToken?: string;
  action: string;
  params: Record<string, unknown>;
  traceId: string;
}

export interface BridgeEnvelope<TPayload> {
  status: "success" | "error";
  payload: TPayload;
  error: string;
}

export function createTraceId(prefix: string): string {
  const normalizedPrefix = prefix.trim() || "mobile";
  const random = crypto.randomUUID?.() ?? `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`;
  return `${normalizedPrefix}-${random}`;
}

export function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

export function hasTauriRuntime(): boolean {
  return "__TAURI_INTERNALS__" in window;
}
