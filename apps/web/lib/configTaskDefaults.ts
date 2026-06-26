import type { TaskRelayConfig } from '@/lib/types';

export const MIN_INTERVAL_SECONDS = 1;
export const DECIMAL_RADIX = 10;
export const TOOL_ALLOWLIST_SEPARATOR_PATTERN = /[\n,]/;
export const DEFAULT_INTERVAL_SECONDS = '300';
export const DEFAULT_RELAY_MAX_ROUNDS = 20;
export const DEFAULT_RELAY_EXECUTION_TIMEOUT_MS = 0;
export const DEFAULT_RELAY_STOP_POLICY: TaskRelayConfig['stop_policy'] = 'ai_decides';
