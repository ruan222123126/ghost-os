import type { ApiEnvelope } from "./envelope.generated";
import { createTraceId as createSharedTraceId } from "../../../shared/trace";

export interface BridgeBusCommand {
  baseUrl: string;
  apiToken?: string;
  action: string;
  params: Record<string, unknown>;
  traceId: string;
}

export type BridgeEnvelope<TPayload> = ApiEnvelope<TPayload>;

export function createTraceId(prefix: string): string {
  return createSharedTraceId(prefix);
}

export function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

export function hasTauriRuntime(): boolean {
  return "__TAURI_INTERNALS__" in window;
}
