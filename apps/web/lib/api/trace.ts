import { createTraceId } from "../../../shared/trace";

export function createClientTraceId(prefix = "web"): string {
  return createTraceId(prefix);
}
