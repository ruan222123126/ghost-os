import type { AgentStreamEvent, SessionPushEvent } from '@/lib/types';
import {
  parseAgentDonePayload,
  parseAgentErrorPayload,
  parseAgentRunStartedPayload,
} from '@/lib/api/agent/parser';
import {
  defineStringEnumValues,
  expectRecord,
  expectString,
  expectStringEnum,
  parseOptionalString,
} from '@/lib/api/shared';

const SESSION_EVENT_TYPES = defineStringEnumValues<SessionPushEvent['type']>({
  assistant_message: true,
  awaiting_human: true,
  run_started: true,
  completion_delta: true,
  tool_call_started: true,
  tool_call_finished: true,
  error: true,
  done: true,
});

const SSE_CONTENT_TYPE = 'text/event-stream';
const SSE_BLOCK_SEPARATOR = '\n\n';

interface StreamSessionEventsOptions {
  onEvent: (event: SessionPushEvent) => void | Promise<void>;
  sessionId: string;
  signal?: AbortSignal;
}

export async function streamSessionEvents(options: StreamSessionEventsOptions): Promise<void> {
  const response = await fetch(`/api/sessions/${encodeURIComponent(options.sessionId)}/events`, {
    method: 'GET',
    cache: 'no-store',
    signal: options.signal,
  });
  await ensureStreamResponse(response);
  await consumeSessionEventStream(response, options.onEvent);
}

export function toAgentStreamEvent(event: SessionPushEvent): AgentStreamEvent {
  return {
    id: event.id,
    step_id: '',
    trace_id: event.trace_id?.trim() || '',
    session_id: event.session_id,
    turn: 0,
    type: event.type as AgentStreamEvent['type'],
    payload: expectRecord(event.payload, `session push ${event.type} payload`),
    at: event.at,
  };
}

export function resolveSessionPushSessionId(event: SessionPushEvent): string {
  if (event.session_id.trim()) {
    return event.session_id.trim();
  }

  switch (event.type) {
    case 'run_started':
      return parseAgentRunStartedPayload(event.payload).session_id?.trim() || '';
    case 'done':
      return parseAgentDonePayload(event.payload).session_id?.trim() || '';
    case 'error':
      return parseAgentErrorPayload(event.payload).session_id?.trim() || '';
    case 'assistant_message':
      return parseOptionalString(
        expectRecord(event.payload, 'session push assistant_message payload').session_id,
        'session push assistant_message payload.session_id',
      ) ?? '';
    default:
      return '';
  }
}

async function ensureStreamResponse(response: Response): Promise<void> {
  const contentType = response.headers.get('content-type') ?? '';
  if (response.ok && contentType.includes(SSE_CONTENT_TYPE) && response.body) {
    return;
  }
  throw new Error(await parseUnexpectedResponse(response));
}

async function consumeSessionEventStream(
  response: Response,
  onEvent: (event: SessionPushEvent) => void | Promise<void>,
): Promise<void> {
  const reader = response.body!.getReader();
  const decoder = new TextDecoder();
  let buffer = '';

  while (true) {
    const { done, value } = await reader.read();
    buffer += decoder.decode(value ?? new Uint8Array(), { stream: !done });
    const { blocks, rest } = parseSSEBlocks(buffer, done);
    buffer = rest;

    for (const block of blocks) {
      const event = parseSessionPushBlock(block);
      if (event) {
        await onEvent(event);
      }
    }

    if (done) {
      break;
    }
  }
}

function parseSSEBlocks(buffer: string, flush: boolean): { blocks: string[]; rest: string } {
  const normalized = buffer.replace(/\r\n/g, '\n');
  const blocks = normalized.split(SSE_BLOCK_SEPARATOR);
  if (flush) {
    return { blocks: blocks.filter(Boolean), rest: '' };
  }

  return {
    blocks: blocks.slice(0, -1).filter(Boolean),
    rest: blocks.at(-1) ?? '',
  };
}

function parseSessionPushBlock(block: string): SessionPushEvent | null {
  const dataLines = block
    .split('\n')
    .filter((line) => line.startsWith('data: '))
    .map((line) => line.slice(6));
  if (dataLines.length === 0) {
    return null;
  }
  return parseSessionPushEvent(JSON.parse(dataLines.join('\n')));
}

function parseSessionPushEvent(payload: unknown): SessionPushEvent {
  const record = expectRecord(payload, 'session push event');
  return {
    id: expectString(record.id, 'session push event.id'),
    type: expectStringEnum(record.type, SESSION_EVENT_TYPES, 'session push event.type'),
    trace_id: parseOptionalString(record.trace_id, 'session push event.trace_id'),
    session_id: expectString(record.session_id, 'session push event.session_id'),
    payload: expectRecord(record.payload, 'session push event.payload'),
    at: parseOptionalString(record.at, 'session push event.at'),
  };
}

async function parseUnexpectedResponse(response: Response): Promise<string> {
  const fallback = response.status
    ? `Request failed with status ${response.status}`
    : 'Request failed';
  const bodyText = (await response.text()).trim();
  if (!bodyText) {
    return fallback;
  }

  try {
    const payload = JSON.parse(bodyText) as { error?: unknown };
    if (typeof payload.error === 'string' && payload.error.trim()) {
      return payload.error.trim();
    }
  } catch {
    return bodyText;
  }

  return bodyText;
}
