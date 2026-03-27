import type {
  AgentRequest,
  AgentStreamEvent,
  HumanResponseRequest,
} from '@/lib/types';
import { createClientTraceId } from '@/lib/api/trace';
import {
  parseAgentDonePayload,
  parseAgentErrorPayload,
  parseAgentRunStartedPayload,
  parseAgentStreamEvent,
  parseAgentStreamMessagePayload,
} from './parser';

const SSE_CONTENT_TYPE = 'text/event-stream';
const SSE_BLOCK_SEPARATOR = '\n\n';

export interface AgentStreamResult {
  message?: string;
  sessionEnded: boolean;
  sessionId?: string;
  traceId?: string;
}

interface StreamAgentRequestOptions<TBody> {
  body: TBody;
  headers?: HeadersInit;
  onEvent: (event: AgentStreamEvent) => void | Promise<void>;
  path: string;
  signal?: AbortSignal;
}

interface StreamSSEParseResult {
  blocks: string[];
  rest: string;
}

interface StreamSummary {
  sawTerminalEvent: boolean;
  result: AgentStreamResult;
}

export interface StreamAgentMessageOptions {
  images?: AgentRequest['images'];
  message: string;
  onEvent: (event: AgentStreamEvent) => void | Promise<void>;
  sessionId?: string;
  signal?: AbortSignal;
  traceId?: string;
}

export interface StreamHumanResponseOptions {
  answer: string;
  cancelled?: boolean;
  onEvent: (event: AgentStreamEvent) => void | Promise<void>;
  questionId: string;
  sessionId: string;
  signal?: AbortSignal;
  traceId?: string;
}

export async function streamMessage(options: StreamAgentMessageOptions): Promise<AgentStreamResult> {
  const body: AgentRequest = { message: options.message };
  if (options.images?.length) {
    body.images = options.images.map((image) => ({ ...image }));
  }
  if (options.sessionId?.trim()) {
    body.session_id = options.sessionId.trim();
  }
  if (options.traceId?.trim()) {
    body.trace_id = options.traceId.trim();
  }

  return streamAgentRequest({
    body,
    onEvent: options.onEvent,
    path: '/api/agent/stream',
    signal: options.signal,
  });
}

export async function streamHumanResponse(options: StreamHumanResponseOptions): Promise<AgentStreamResult> {
  const body: HumanResponseRequest = {
    answer: options.answer.trim(),
    question_id: options.questionId.trim(),
    session_id: options.sessionId.trim(),
  };
  if (options.cancelled) {
    body.cancelled = true;
  }

  return streamAgentRequest({
    body,
    headers: {
      'X-Trace-ID': options.traceId?.trim() || createClientTraceId('human-response'),
    },
    onEvent: options.onEvent,
    path: '/api/questions/answer/stream',
    signal: options.signal,
  });
}

async function streamAgentRequest<TBody>(options: StreamAgentRequestOptions<TBody>): Promise<AgentStreamResult> {
  const response = await fetch(options.path, {
    method: 'POST',
    headers: buildHeaders(options.headers),
    body: JSON.stringify(options.body),
    cache: 'no-store',
    signal: options.signal,
  });
  await ensureStreamResponse(response);
  return consumeEventStream(response, options.onEvent);
}

function buildHeaders(headers: HeadersInit | undefined): Headers {
  const merged = new Headers(headers);
  if (!merged.has('Content-Type')) {
    merged.set('Content-Type', 'application/json');
  }
  return merged;
}

async function ensureStreamResponse(response: Response): Promise<void> {
  const contentType = response.headers.get('content-type') ?? '';
  if (response.ok && contentType.includes(SSE_CONTENT_TYPE) && response.body) {
    return;
  }

  throw new Error(await parseUnexpectedResponse(response, contentType));
}

async function parseUnexpectedResponse(response: Response, contentType: string): Promise<string> {
  const fallback = response.status
    ? `Request failed with status ${response.status}`
    : 'Request failed';
  const bodyText = (await response.text()).trim();
  if (!bodyText) {
    if (response.ok && !contentType.includes(SSE_CONTENT_TYPE)) {
      return `Expected ${SSE_CONTENT_TYPE} response but received ${contentType || 'empty content-type'}`;
    }
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

async function consumeEventStream(
  response: Response,
  onEvent: (event: AgentStreamEvent) => void | Promise<void>,
): Promise<AgentStreamResult> {
  const reader = response.body!.getReader();
  const decoder = new TextDecoder();
  const summary: StreamSummary = {
    sawTerminalEvent: false,
    result: {
      sessionEnded: false,
    },
  };
  let buffer = '';

  while (true) {
    const { done, value } = await reader.read();
    buffer += decoder.decode(value ?? new Uint8Array(), { stream: !done });
    const parsed = parseSSEBlocks(buffer, done);
    buffer = parsed.rest;
    for (const block of parsed.blocks) {
      const event = parseSSEBlock(block);
      if (!event) {
        continue;
      }
      await onEvent(event);
      updateStreamSummary(summary, event);
    }
    if (done) {
      break;
    }
  }

  if (!summary.sawTerminalEvent) {
    throw new Error('agent stream closed before terminal event');
  }
  return summary.result;
}

function parseSSEBlocks(buffer: string, flush: boolean): StreamSSEParseResult {
  const normalized = buffer.replace(/\r\n/g, '\n');
  const blocks = normalized.split(SSE_BLOCK_SEPARATOR);
  if (flush) {
    return {
      blocks: blocks.filter((block) => block.trim() !== ''),
      rest: '',
    };
  }

  const rest = blocks.pop() ?? '';
  return {
    blocks: blocks.filter((block) => block.trim() !== ''),
    rest,
  };
}

function parseSSEBlock(block: string): AgentStreamEvent | null {
  const dataLines = block
    .split('\n')
    .filter((line) => line.startsWith('data: '))
    .map((line) => line.slice(6));
  if (dataLines.length === 0) {
    return null;
  }

  return parseAgentStreamEvent(JSON.parse(dataLines.join('\n')));
}

function updateStreamSummary(summary: StreamSummary, event: AgentStreamEvent): void {
  summary.result.traceId = summary.result.traceId || event.trace_id;
  summary.result.sessionId = summary.result.sessionId || resolveSessionID(event);

  switch (event.type) {
    case 'awaiting_human':
      summary.sawTerminalEvent = true;
      break;
    case 'message':
      summary.result.message = parseAgentStreamMessagePayload(event.payload).text;
      break;
    case 'done':
      summary.sawTerminalEvent = true;
      summary.result.sessionEnded = Boolean(parseAgentDonePayload(event.payload).session_ended);
      break;
    case 'error':
      summary.sawTerminalEvent = true;
      throw new Error(parseAgentErrorPayload(event.payload).message);
    default:
      break;
  }
}

function resolveSessionID(event: AgentStreamEvent): string | undefined {
  if (event.session_id?.trim()) {
    return event.session_id.trim();
  }

  switch (event.type) {
    case 'run_started':
      return parseAgentRunStartedPayload(event.payload).session_id?.trim() || undefined;
    case 'message':
      return parseAgentStreamMessagePayload(event.payload).session_id?.trim() || undefined;
    case 'done':
      return parseAgentDonePayload(event.payload).session_id?.trim() || undefined;
    case 'error':
      return parseAgentErrorPayload(event.payload).session_id?.trim() || undefined;
    default:
      return undefined;
  }
}
