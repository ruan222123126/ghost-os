import { invoke } from "@tauri-apps/api/core";
import { listen } from "@tauri-apps/api/event";
import {
  parseOptionalBoolean,
  parseOptionalNumber,
  parseOptionalString,
  requireNumber as expectNumber,
  requireRecord as expectRecord,
  requireString as expectString,
  requireStringEnum as expectStringEnum,
} from "./payloadValidators";

const BRIDGE_AGENT_STREAM_CHUNK_EVENT = "bridge-agent-stream-chunk";
const SSE_BLOCK_SEPARATOR = "\n\n";
const STREAM_RECONNECT_DELAYS_MS = [300, 700, 1500] as const;

type AgentStreamEventType =
  | "run_started"
  | "completion_delta"
  | "tool_call_started"
  | "tool_call_finished"
  | "awaiting_human"
  | "message"
  | "done"
  | "error";

type CompletionDeltaKind = "text" | "thinking" | "tool_call_start" | "tool_call_delta" | "tool_call_end";

export interface AgentStreamEvent {
  id: string;
  step_id: string;
  trace_id: string;
  session_id?: string;
  turn: number;
  type: AgentStreamEventType;
  payload: Record<string, unknown>;
  at?: string;
}

export interface AgentCompletionDeltaPayload {
  kind: CompletionDeltaKind;
  text?: string;
  thinking?: string;
  tool_call_index?: number;
  tool_call_id?: string;
  tool_name?: string;
  arguments_fragment?: string;
}

export interface AgentToolCallPayload {
  tool?: string;
  tool_call_id?: string;
  arguments_json?: string;
  status?: string;
  error?: string;
  output?: string;
}

export interface AgentAwaitingHumanStreamPayload {
  approval?: AgentStreamApprovalPayload;
  tool?: string;
  tool_call_id?: string;
  question_id: string;
  prompt: string;
}

export interface AgentStreamApprovalPayload {
  id?: string;
  kind?: string;
  payload?: Record<string, unknown>;
  tool?: string;
}

export interface AgentStreamMessagePayload {
  text: string;
  session_id?: string;
}

export interface AgentDonePayload {
  session_id?: string;
  session_ended?: boolean;
}

export interface AgentErrorPayload {
  message: string;
  session_id?: string;
  code?: number;
}

export interface AgentStreamResult {
  message?: string;
  sessionEnded: boolean;
  sessionId?: string;
  traceId?: string;
  awaitingHuman: boolean;
}

export interface AgentStreamSummary {
  sawTerminalEvent: boolean;
  result: AgentStreamResult;
}

interface StreamAgentMessageHTTPOptions {
  apiToken?: string;
  baseUrl: string;
  body?: Record<string, unknown>;
  message: string;
  onEvent: (event: AgentStreamEvent) => void;
  onReconnectAttempt?: (attempt: number, maxAttempts: number) => void;
  path?: string;
  requestId: string;
  runtimeOverrides?: Record<string, unknown>;
  sessionId?: string;
  traceId: string;
}

interface BridgeAgentStreamCommand {
  apiToken?: string;
  baseUrl: string;
  body?: Record<string, unknown>;
  message: string;
  path?: string;
  requestId: string;
  runtimeOverrides?: Record<string, unknown>;
  sessionId?: string;
  traceId: string;
}

interface BridgeAgentStreamReconnectCommand {
  apiToken?: string;
  baseUrl: string;
  lastEventId?: string;
  requestId: string;
  traceId: string;
}

interface BridgeAgentStreamChunk {
  requestId: string;
  chunk: number[];
}

interface SSEParseResult {
  blocks: string[];
  rest: string;
}

interface BridgeStreamInvocationOptions {
  consumeText: (text: string, flush: boolean) => void;
  flushDecoder: () => string;
  getStreamError: () => Error | null;
  invokeStream: () => Promise<void>;
  summary: AgentStreamSummary;
}

const AGENT_STREAM_EVENT_TYPES = {
  awaiting_human: true,
  completion_delta: true,
  done: true,
  error: true,
  message: true,
  run_started: true,
  tool_call_finished: true,
  tool_call_started: true,
} as const satisfies Record<AgentStreamEventType, true>;

const COMPLETION_DELTA_KINDS = {
  text: true,
  thinking: true,
  tool_call_delta: true,
  tool_call_end: true,
  tool_call_start: true,
} as const satisfies Record<CompletionDeltaKind, true>;

export async function streamAgentMessageHTTP(options: StreamAgentMessageHTTPOptions): Promise<AgentStreamResult> {
  const summary = createAgentStreamSummary();
  const seenEventIds = new Set<string>();
  let decoder = new TextDecoder();
  let buffer = "";
  let lastEventId = "";
  let streamError: Error | null = null;

  function consumeText(text: string, flush: boolean): void {
    if (streamError) {
      return;
    }

    try {
      buffer += text;
      const parsed = parseSSEBlocks(buffer, flush);
      buffer = parsed.rest;
      for (const block of parsed.blocks) {
        const event = parseSSEBlock(block);
        if (!event || seenEventIds.has(event.id)) {
          continue;
        }
        seenEventIds.add(event.id);
        lastEventId = event.id;
        options.onEvent(event);
        updateAgentStreamSummary(summary, event);
      }
    } catch (error) {
      streamError = toError(error);
    }
  }

  const unlisten = await listen<BridgeAgentStreamChunk>(BRIDGE_AGENT_STREAM_CHUNK_EVENT, (event) => {
    if (event.payload.requestId !== options.requestId) {
      return;
    }
    consumeText(decoder.decode(Uint8Array.from(event.payload.chunk), { stream: true }), false);
  });

  try {
    const initialError = await consumeBridgeStreamInvocation({
      consumeText,
      flushDecoder: () => decoder.decode(),
      getStreamError: () => streamError,
      invokeStream: () => invoke<void>("bridge_agent_stream", { request: buildBridgeAgentStreamCommand(options) }),
      summary,
    });
    if (!initialError) {
      return summary.result;
    }
    if (!isReconnectableBridgeStreamError(initialError)) {
      throw initialError;
    }

    let finalError = initialError;
    for (const [index, delayMs] of STREAM_RECONNECT_DELAYS_MS.entries()) {
      const attempt = index + 1;
      options.onReconnectAttempt?.(attempt, STREAM_RECONNECT_DELAYS_MS.length);
      await waitForReconnect(delayMs);
      decoder = new TextDecoder();
      buffer = "";
      const reconnectRequest = buildBridgeAgentStreamReconnectCommand(options, lastEventId);
      const reconnectError = await consumeBridgeStreamInvocation({
        consumeText,
        flushDecoder: () => decoder.decode(),
        getStreamError: () => streamError,
        invokeStream: () => invoke<void>("bridge_agent_stream_reconnect", { request: reconnectRequest }),
        summary,
      });
      if (!reconnectError) {
        return summary.result;
      }
      finalError = reconnectError;
    }
    throw finalError;
  } finally {
    unlisten();
  }
}

async function consumeBridgeStreamInvocation(options: BridgeStreamInvocationOptions): Promise<Error | null> {
  try {
    await options.invokeStream();
    await flushPendingEventCallbacks();
    options.consumeText(options.flushDecoder(), true);
  } catch (error) {
    await flushPendingEventCallbacks();
    const streamError = options.getStreamError();
    if (streamError) {
      throw streamError;
    }
    if (options.summary.sawTerminalEvent) {
      return null;
    }
    return toError(error);
  }

  const streamError = options.getStreamError();
  if (streamError) {
    throw streamError;
  }
  try {
    assertAgentStreamTerminal(options.summary);
    return null;
  } catch (error) {
    return toError(error);
  }
}

function isReconnectableBridgeStreamError(error: Error): boolean {
  return error.message.includes("read bridge stream failed:") ||
    error.message === "agent stream closed before terminal event";
}

async function waitForReconnect(delayMs: number): Promise<void> {
  await new Promise<void>((resolve) => {
    window.setTimeout(resolve, delayMs);
  });
}

export function createAgentStreamSummary(): AgentStreamSummary {
  return {
    result: {
      awaitingHuman: false,
      sessionEnded: false,
    },
    sawTerminalEvent: false,
  };
}

export function updateAgentStreamSummary(summary: AgentStreamSummary, event: AgentStreamEvent): void {
  summary.result.traceId = summary.result.traceId || event.trace_id;
  summary.result.sessionId = summary.result.sessionId || resolveSessionId(event);

  switch (event.type) {
    case "awaiting_human":
      summary.result.awaitingHuman = true;
      summary.sawTerminalEvent = true;
      break;
    case "message":
      summary.result.message = parseAgentStreamMessagePayload(event.payload).text;
      break;
    case "done":
      summary.sawTerminalEvent = true;
      summary.result.sessionEnded = Boolean(parseAgentDonePayload(event.payload).session_ended);
      break;
    case "error":
      summary.sawTerminalEvent = true;
      throw new Error(parseAgentErrorPayload(event.payload).message);
    default:
      break;
  }
}

export function assertAgentStreamTerminal(summary: AgentStreamSummary): void {
  if (!summary.sawTerminalEvent) {
    throw new Error("agent stream closed before terminal event");
  }
}

export function parseAgentStreamEvent(payload: unknown): AgentStreamEvent {
  const record = expectRecord(payload, "agent stream event");

  return {
    at: parseOptionalString(record.at, "agent stream event.at"),
    id: expectString(record.id, "agent stream event.id"),
    payload: expectRecord(record.payload, "agent stream event.payload"),
    session_id: parseOptionalString(record.session_id, "agent stream event.session_id"),
    step_id: expectString(record.step_id, "agent stream event.step_id"),
    trace_id: expectString(record.trace_id, "agent stream event.trace_id"),
    turn: expectNumber(record.turn, "agent stream event.turn"),
    type: expectStringEnum(record.type, AGENT_STREAM_EVENT_TYPES, "agent stream event.type"),
  };
}

export function parseAgentCompletionDeltaPayload(payload: unknown): AgentCompletionDeltaPayload {
  const record = expectRecord(payload, "agent completion_delta payload");

  return {
    arguments_fragment: parseOptionalString(record.arguments_fragment, "agent completion_delta payload.arguments_fragment"),
    kind: expectStringEnum(record.kind, COMPLETION_DELTA_KINDS, "agent completion_delta payload.kind"),
    text: parseOptionalString(record.text, "agent completion_delta payload.text"),
    thinking: parseOptionalString(record.thinking, "agent completion_delta payload.thinking"),
    tool_call_id: parseOptionalString(record.tool_call_id, "agent completion_delta payload.tool_call_id"),
    tool_call_index: parseOptionalNumber(record.tool_call_index, "agent completion_delta payload.tool_call_index"),
    tool_name: parseOptionalString(record.tool_name, "agent completion_delta payload.tool_name"),
  };
}

export function parseAgentToolCallPayload(payload: unknown): AgentToolCallPayload {
  const record = expectRecord(payload, "agent tool call payload");

  return {
    arguments_json: parseOptionalString(record.arguments_json, "agent tool call payload.arguments_json"),
    error: parseOptionalString(record.error, "agent tool call payload.error"),
    output: parseOptionalString(record.output, "agent tool call payload.output"),
    status: parseOptionalString(record.status, "agent tool call payload.status"),
    tool: parseOptionalString(record.tool, "agent tool call payload.tool"),
    tool_call_id: parseOptionalString(record.tool_call_id, "agent tool call payload.tool_call_id"),
  };
}

export function parseAgentAwaitingHumanStreamPayload(payload: unknown): AgentAwaitingHumanStreamPayload {
  const record = expectRecord(payload, "agent awaiting_human payload");
  const approval = record.approval === undefined
    ? undefined
    : parseAgentStreamApprovalPayload(record.approval);

  return {
    approval,
    prompt: expectString(record.prompt, "agent awaiting_human payload.prompt"),
    question_id: expectString(record.question_id, "agent awaiting_human payload.question_id"),
    tool: parseOptionalString(record.tool, "agent awaiting_human payload.tool"),
    tool_call_id: parseOptionalString(record.tool_call_id, "agent awaiting_human payload.tool_call_id"),
  };
}

function parseAgentStreamApprovalPayload(payload: unknown): AgentStreamApprovalPayload {
  const record = expectRecord(payload, "agent awaiting_human payload.approval");
  return {
    id: parseOptionalString(record.id, "agent awaiting_human payload.approval.id"),
    kind: parseOptionalString(record.kind, "agent awaiting_human payload.approval.kind"),
    payload: record.payload === undefined
      ? undefined
      : expectRecord(record.payload, "agent awaiting_human payload.approval.payload"),
    tool: parseOptionalString(record.tool, "agent awaiting_human payload.approval.tool"),
  };
}

export function parseAgentStreamMessagePayload(payload: unknown): AgentStreamMessagePayload {
  const record = expectRecord(payload, "agent message payload");

  return {
    session_id: parseOptionalString(record.session_id, "agent message payload.session_id"),
    text: expectString(record.text, "agent message payload.text"),
  };
}

export function parseAgentDonePayload(payload: unknown): AgentDonePayload {
  const record = expectRecord(payload, "agent done payload");

  return {
    session_ended: parseOptionalBoolean(record.session_ended, "agent done payload.session_ended"),
    session_id: parseOptionalString(record.session_id, "agent done payload.session_id"),
  };
}

export function parseAgentErrorPayload(payload: unknown): AgentErrorPayload {
  const record = expectRecord(payload, "agent error payload");

  return {
    code: parseOptionalNumber(record.code, "agent error payload.code"),
    message: expectString(record.message, "agent error payload.message"),
    session_id: parseOptionalString(record.session_id, "agent error payload.session_id"),
  };
}

export function resolveSessionId(event: AgentStreamEvent): string | undefined {
  if (event.session_id?.trim()) {
    return event.session_id.trim();
  }

  switch (event.type) {
    case "run_started":
      return parseOptionalString(event.payload.session_id, "agent run_started payload.session_id")?.trim() || undefined;
    case "message":
      return parseAgentStreamMessagePayload(event.payload).session_id?.trim() || undefined;
    case "done":
      return parseAgentDonePayload(event.payload).session_id?.trim() || undefined;
    case "error":
      return parseAgentErrorPayload(event.payload).session_id?.trim() || undefined;
    default:
      return undefined;
  }
}

function buildBridgeAgentStreamCommand(options: StreamAgentMessageHTTPOptions): BridgeAgentStreamCommand {
  return {
    apiToken: options.apiToken?.trim() || undefined,
    baseUrl: options.baseUrl,
    body: options.body,
    message: options.message,
    path: options.path,
    requestId: options.requestId,
    runtimeOverrides: options.runtimeOverrides,
    sessionId: options.sessionId?.trim() || undefined,
    traceId: options.traceId,
  };
}

function buildBridgeAgentStreamReconnectCommand(
  options: StreamAgentMessageHTTPOptions,
  lastEventId: string,
): BridgeAgentStreamReconnectCommand {
  return {
    apiToken: options.apiToken?.trim() || undefined,
    baseUrl: options.baseUrl,
    lastEventId: lastEventId || undefined,
    requestId: options.requestId,
    traceId: options.traceId,
  };
}

function parseSSEBlocks(buffer: string, flush: boolean): SSEParseResult {
  const normalized = buffer.replace(/\r\n/g, "\n");
  const blocks = normalized.split(SSE_BLOCK_SEPARATOR);
  if (flush) {
    return {
      blocks: blocks.filter((block) => block.trim() !== ""),
      rest: "",
    };
  }

  const rest = blocks.pop() ?? "";
  return {
    blocks: blocks.filter((block) => block.trim() !== ""),
    rest,
  };
}

function parseSSEBlock(block: string): AgentStreamEvent | null {
  const dataLines = block
    .split("\n")
    .filter((line) => line.startsWith("data: "))
    .map((line) => line.slice(6));
  if (dataLines.length === 0) {
    return null;
  }

  return parseAgentStreamEvent(JSON.parse(dataLines.join("\n")));
}

async function flushPendingEventCallbacks(): Promise<void> {
  await new Promise<void>((resolve) => {
    window.setTimeout(resolve, 0);
  });
}

function toError(error: unknown): Error {
  return error instanceof Error ? error : new Error(String(error));
}
