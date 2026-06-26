import type { AgentStreamEvent } from '@/lib/types';
import type { LiveTaskRunCard } from '@/lib/taskRunViewerCards';

const RELAY_TOOL_NAMES = new Set(['relay_update_record', 'relay_complete']);

export interface RelayCardSummary {
  did: string;
  nextStep: string;
  log: string;
}

type RelaySummaryField = keyof RelayCardSummary;

const RELAY_SUMMARY_FIELDS_BY_LABEL = new Map<string, RelaySummaryField>([
  ['did', 'did'],
  ['已完成', 'did'],
  ['nextstep', 'nextStep'],
  ['next', 'nextStep'],
  ['下阶段', 'nextStep'],
  ['下一阶段', 'nextStep'],
  ['finalchangelog', 'log'],
  ['changelog', 'log'],
  ['log', 'log'],
  ['交付', 'log'],
]);

const IGNORED_RELAY_LABELS = new Set([
  'remaining',
  'failedattempts',
  'finalmessage',
  'status',
]);

export function buildRelayCardSummary(card: LiveTaskRunCard): RelayCardSummary | null {
  if (card.kind !== 'relay_round') {
    return null;
  }
  const fromEvents = relaySummaryFromEvents(card.source_events);
  const fromFinalText = relaySummaryFromText(card.final_text);
  const fromPreview = relaySummaryFromText(card.preview);
  const summary = mergeRelaySummaries(fromEvents, fromFinalText, fromPreview);
  return hasRelaySummary(summary) ? summary : null;
}

function relaySummaryFromEvents(events: AgentStreamEvent[]): RelayCardSummary {
  let summary = emptyRelaySummary();
  for (const event of events) {
    const payload = relayToolPayloadFromEvent(event);
    if (!payload) {
      continue;
    }
    summary = mergeRelaySummaries(summary, relaySummaryFromRecord(payload));
  }
  return summary;
}

function relayToolPayloadFromEvent(event: AgentStreamEvent): Record<string, unknown> | null {
  if (event.type === 'tool_call_finished') {
    return relayToolPayloadFromFinishedEvent(event.payload);
  }
  if (event.type === 'tool_call_started') {
    return relayToolPayloadFromStartedEvent(event.payload);
  }
  return null;
}

function relayToolPayloadFromFinishedEvent(payload: Record<string, unknown>): Record<string, unknown> | null {
  if (!isRelayToolName(payload.tool)) {
    return null;
  }
  return parseRecord(payload.output);
}

function relayToolPayloadFromStartedEvent(payload: Record<string, unknown>): Record<string, unknown> | null {
  if (!isRelayToolName(payload.tool)) {
    return null;
  }
  return parseRecord(payload.arguments_json);
}

function isRelayToolName(value: unknown): boolean {
  return typeof value === 'string' && RELAY_TOOL_NAMES.has(value.trim());
}

function relaySummaryFromText(value?: string): RelayCardSummary {
  const text = value?.trim();
  if (!text) {
    return emptyRelaySummary();
  }
  const record = parseRecord(text);
  if (record) {
    return relaySummaryFromRecord(record);
  }
  return relaySummaryFromLines(text);
}

function relaySummaryFromRecord(record: Record<string, unknown>): RelayCardSummary {
  return {
    did: readString(record, ['did', '已完成']),
    nextStep: readString(record, ['next_step', 'nextStep', 'nextstep', '下阶段', '下一阶段']),
    log: readString(record, ['final_change_log', 'finalChangeLog', 'change_log', 'log', '交付']),
  };
}

function relaySummaryFromLines(text: string): RelayCardSummary {
  const buckets: Record<keyof RelayCardSummary, string[]> = {
    did: [],
    nextStep: [],
    log: [],
  };
  let current: keyof RelayCardSummary | null = null;
  for (const rawLine of text.split(/\r?\n/)) {
    const parsed = parseSummaryLine(rawLine);
    if (parsed) {
      if (!parsed.field) {
        current = null;
        continue;
      }
      current = parsed.field;
      if (parsed.content) {
        buckets[current].push(parsed.content);
      }
      continue;
    }
    const content = rawLine.trim();
    if (current && content) {
      buckets[current].push(content);
    }
  }
  return {
    did: buckets.did.join('\n').trim(),
    nextStep: buckets.nextStep.join('\n').trim(),
    log: buckets.log.join('\n').trim(),
  };
}

function parseSummaryLine(line: string): { content: string; field: keyof RelayCardSummary | null } | null {
  const match = /^\s*([^:：]+)\s*[:：]\s*(.*)$/.exec(line);
  if (!match) {
    return null;
  }
  const field = relayFieldFromLabel(match[1]);
  if (!field && !isIgnoredRelayLabel(match[1])) {
    return null;
  }
  return {
    field,
    content: match[2].trim(),
  };
}

function relayFieldFromLabel(label: string): RelaySummaryField | null {
  return RELAY_SUMMARY_FIELDS_BY_LABEL.get(normalizeRelayLabel(label)) ?? null;
}

function isIgnoredRelayLabel(label: string): boolean {
  return IGNORED_RELAY_LABELS.has(normalizeRelayLabel(label));
}

function normalizeRelayLabel(label: string): string {
  return label.trim().toLowerCase().replace(/[\s_-]+/g, '');
}

function parseRecord(value: unknown): Record<string, unknown> | null {
  if (isRecord(value)) {
    return value;
  }
  if (typeof value !== 'string') {
    return null;
  }
  const text = value.trim();
  if (!text) {
    return null;
  }
  try {
    const parsed = JSON.parse(text) as unknown;
    return isRecord(parsed) ? parsed : null;
  } catch {
    return null;
  }
}

function readString(record: Record<string, unknown>, keys: string[]): string {
  for (const key of keys) {
    const value = record[key];
    if (typeof value !== 'string') {
      continue;
    }
    const text = value.trim();
    if (text) {
      return text;
    }
  }
  return '';
}

function mergeRelaySummaries(...summaries: RelayCardSummary[]): RelayCardSummary {
  return summaries.reduce(
    (merged, summary) => ({
      did: merged.did || summary.did,
      nextStep: merged.nextStep || summary.nextStep,
      log: merged.log || summary.log,
    }),
    emptyRelaySummary(),
  );
}

function hasRelaySummary(summary: RelayCardSummary): boolean {
  return Boolean(summary.did || summary.nextStep || summary.log);
}

function emptyRelaySummary(): RelayCardSummary {
  return {
    did: '',
    nextStep: '',
    log: '',
  };
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}
