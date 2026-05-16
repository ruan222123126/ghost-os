import type { SessionSourceKind } from '@/lib/sessionSidebarSessionSources';
import type { TaskRunLog } from '@/lib/types';

export function collectHiddenSessionIDsFromRun(
  run: TaskRunLog,
  source: SessionSourceKind,
): string[] {
  if (source === 'workflow') {
    return workflowExecutionSessionIDs(run);
  }
  if (source === 'orchestration') {
    return orchestrationExecutionSessionIDs(run);
  }
  return [];
}

function workflowExecutionSessionIDs(run: TaskRunLog): string[] {
  const ids = new Set<string>();
  for (const node of run.node_results ?? []) {
    addSessionID(ids, recordFrom(node.output)?.session_id_output);
  }
  return [...ids];
}

function orchestrationExecutionSessionIDs(run: TaskRunLog): string[] {
  const ids = new Set<string>();
  for (const node of run.node_results ?? []) {
    addOrchestrationOutputSessionIDs(ids, recordFrom(node.output));
  }
  return [...ids];
}

function addOrchestrationOutputSessionIDs(
  ids: Set<string>,
  output: Record<string, unknown> | undefined,
): void {
  if (!output) {
    return;
  }
  addSessionID(ids, output.owner_session_id);
  addRecordValues(ids, recordFrom(output.member_session_ids));
  addMemberResultSessionIDs(ids, arrayFrom(output.member_results));
  for (const dispatch of arrayFrom(output.dispatch_results)) {
    addMemberResultSessionIDs(ids, arrayFrom(recordFrom(dispatch)?.member_results));
  }
}

function addMemberResultSessionIDs(ids: Set<string>, items: unknown[]): void {
  for (const item of items) {
    addSessionID(ids, recordFrom(item)?.session_id);
  }
}

function addRecordValues(
  ids: Set<string>,
  record: Record<string, unknown> | undefined,
): void {
  if (!record) {
    return;
  }
  for (const value of Object.values(record)) {
    addSessionID(ids, value);
  }
}

function addSessionID(ids: Set<string>, value: unknown): void {
  if (typeof value !== 'string') {
    return;
  }
  const id = value.trim();
  if (id) {
    ids.add(id);
  }
}

function recordFrom(value: unknown): Record<string, unknown> | undefined {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) {
    return undefined;
  }
  return value as Record<string, unknown>;
}

function arrayFrom(value: unknown): unknown[] {
  return Array.isArray(value) ? value : [];
}
