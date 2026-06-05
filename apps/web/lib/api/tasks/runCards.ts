import { parseAgentStreamEvent } from '@/lib/api/agent/parser';
import type { TaskRunCard } from '@/lib/taskRunCards';
import {
  expectRecord,
  expectString,
  parseOptionalNumber,
  parseOptionalString,
} from '@/lib/api/shared';

export function parseTaskRunCard(value: unknown, label: string): TaskRunCard {
  const record = expectRecord(value, label);
  return {
    card_id: expectString(record.card_id, `${label}.card_id`),
    run_id: parseOptionalString(record.run_id, `${label}.run_id`),
    kind: expectString(record.kind, `${label}.kind`),
    title: parseOptionalString(record.title, `${label}.title`),
    node_id: parseOptionalString(record.node_id, `${label}.node_id`),
    node_type: parseOptionalString(record.node_type, `${label}.node_type`),
    round: parseOptionalNumber(record.round, `${label}.round`),
    iteration: parseOptionalNumber(record.iteration, `${label}.iteration`),
    branch_id: parseOptionalString(record.branch_id, `${label}.branch_id`),
    source_session_id: parseOptionalString(record.source_session_id, `${label}.source_session_id`),
    started_at: parseOptionalString(record.started_at, `${label}.started_at`),
    status: parseOptionalString(record.status, `${label}.status`),
    finished_at: parseOptionalString(record.finished_at, `${label}.finished_at`),
    preview: parseOptionalString(record.preview, `${label}.preview`),
    error: parseOptionalString(record.error, `${label}.error`),
    final_text: parseOptionalString(record.final_text, `${label}.final_text`),
    source_events: parseTaskRunCardSourceEvents(record.source_events, `${label}.source_events`),
  };
}

export function parseTaskRunCardList(
  value: unknown,
  label: string,
): TaskRunCard[] | undefined {
  if (value === undefined || value === null) {
    return undefined;
  }
  if (!Array.isArray(value)) {
    throw new Error(`Invalid ${label}: expected array`);
  }
  return value.map((item, index) => parseTaskRunCard(item, `${label}[${index}]`));
}

function parseTaskRunCardSourceEvents(
  value: unknown,
  label: string,
) {
  if (value === undefined || value === null) {
    return undefined;
  }
  if (!Array.isArray(value)) {
    throw new Error(`Invalid ${label}: expected array`);
  }
  return value.map((item) => parseAgentStreamEvent(item));
}
