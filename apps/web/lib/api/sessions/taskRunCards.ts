import { parseAgentStreamEvent } from '@/lib/api/agent/parser';
import {
  expectRecord,
  expectString,
  parseOptionalNumber,
  parseOptionalString,
} from '@/lib/api/shared';
import type {
  TaskRunCardEventPayload,
  TaskRunCardFinishedPayload,
  TaskRunCardStartedPayload,
} from '@/lib/types';

export function parseTaskRunCardStartedPayload(
  value: unknown,
): TaskRunCardStartedPayload {
  const record = expectRecord(value, 'task run card started payload');
  return {
    card_id: expectString(record.card_id, 'task run card started payload.card_id'),
    run_id: expectString(record.run_id, 'task run card started payload.run_id'),
    kind: expectString(record.kind, 'task run card started payload.kind'),
    title: parseOptionalString(record.title, 'task run card started payload.title'),
    node_id: parseOptionalString(record.node_id, 'task run card started payload.node_id'),
    node_type: parseOptionalString(record.node_type, 'task run card started payload.node_type'),
    round: parseOptionalNumber(record.round, 'task run card started payload.round'),
    iteration: parseOptionalNumber(record.iteration, 'task run card started payload.iteration'),
    branch_id: parseOptionalString(record.branch_id, 'task run card started payload.branch_id'),
    source_session_id: parseOptionalString(record.source_session_id, 'task run card started payload.source_session_id'),
    started_at: expectString(record.started_at, 'task run card started payload.started_at'),
  };
}

export function parseTaskRunCardEventPayload(
  value: unknown,
): TaskRunCardEventPayload {
  const record = expectRecord(value, 'task run card event payload');
  return {
    card_id: expectString(record.card_id, 'task run card event payload.card_id'),
    source_session_id: parseOptionalString(record.source_session_id, 'task run card event payload.source_session_id'),
    source_event: parseAgentStreamEvent(record.source_event),
  };
}

export function parseTaskRunCardFinishedPayload(
  value: unknown,
): TaskRunCardFinishedPayload {
  const record = expectRecord(value, 'task run card finished payload');
  return {
    card_id: expectString(record.card_id, 'task run card finished payload.card_id'),
    status: expectString(record.status, 'task run card finished payload.status'),
    finished_at: expectString(record.finished_at, 'task run card finished payload.finished_at'),
    preview: parseOptionalString(record.preview, 'task run card finished payload.preview'),
    error: parseOptionalString(record.error, 'task run card finished payload.error'),
    source_session_id: parseOptionalString(record.source_session_id, 'task run card finished payload.source_session_id'),
  };
}
