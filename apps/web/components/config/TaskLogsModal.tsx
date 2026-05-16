'use client';

import { useWebLocale } from '@/lib/i18n/provider';
import type { TaskRunLog, TaskRunNodeResult } from '@/lib/types';
import { OrchestrationRoundsBlock, parseOrchestrationGroupOutput } from './TaskLogsOrchestration';

interface TaskLogsModalProps {
  taskID: string;
  logs: TaskRunLog[];
  loading: boolean;
  error: string;
  onClose: () => void;
}

export function TaskLogsModal(props: TaskLogsModalProps) {
  const { copy } = useWebLocale();
  const { taskID, logs, loading, error, onClose } = props;

  return (
    <div className="fixed inset-0 z-[80] flex items-center justify-center p-4" role="dialog" aria-modal="true">
      <button type="button" className="absolute inset-0 bg-black/35 backdrop-blur-[2px]" onClick={onClose} aria-label={copy.settings.tasksLogsClose} />
      <section className="relative z-[81] flex h-[78vh] w-full max-w-[900px] flex-col overflow-hidden rounded-[18px] border border-[#E5E5E5] bg-white shadow-2xl">
        <TaskLogsModalHeader taskID={taskID} onClose={onClose} />
        <TaskLogsModalBody logs={logs} loading={loading} error={error} />
      </section>
    </div>
  );
}

function TaskLogsModalHeader(props: { taskID: string; onClose: () => void }) {
  const { copy } = useWebLocale();
  const { taskID, onClose } = props;
  return (
    <header className="flex items-center justify-between border-b border-[#E5E5E5] px-5 py-4">
      <h2 className="text-[15px] font-semibold text-[#111111]">{copy.settings.tasksLogsTitle(taskID)}</h2>
      <button
        type="button"
        onClick={onClose}
        className="rounded-full border border-[#E5E5E5] px-3 py-1 text-[12px] font-medium text-[#111111] transition-colors hover:bg-[#F5F5F5]"
      >
        {copy.settings.tasksLogsClose}
      </button>
    </header>
  );
}

function TaskLogsModalBody(props: { logs: TaskRunLog[]; loading: boolean; error: string }) {
  const { copy } = useWebLocale();
  const { logs, loading, error } = props;

  if (loading) {
    return <div className="p-5 text-[13px] text-[#737373]">{copy.settings.tasksLogsLoading}</div>;
  }
  if (error) {
    return <div className="m-5 rounded-[10px] border border-[#FECACA] bg-[#FEF2F2] px-3 py-2 text-[12px] text-[#B91C1C]">{error}</div>;
  }
  if (logs.length === 0) {
    return <div className="p-5 text-[13px] text-[#737373]">{copy.settings.tasksLogsEmpty}</div>;
  }
  return (
    <div className="min-h-0 flex-1 overflow-y-auto px-5 py-4">
      {logs.map((log) => <TaskRunLogCard key={log.run_id} log={log} />)}
    </div>
  );
}

function TaskRunLogCard(props: { log: TaskRunLog }) {
  const { copy } = useWebLocale();
  const { log } = props;

  return (
    <details className="mb-3 rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] p-3 last:mb-0" open>
      <summary className="cursor-pointer list-none text-[12px] text-[#111111]">
        <span className="font-semibold">{log.run_id}</span>
        <span className="mx-2 text-[#737373]">{log.status}</span>
        <span className="text-[#737373]">{formatLogTime(log.started_at ?? log.scheduled_at)}</span>
      </summary>
      <div className="mt-3 space-y-2 text-[12px] text-[#111111]">
        <p className="font-mono text-[#525252]">trace_id: {log.trace_id}</p>
        {log.response_preview ? <p>preview: {log.response_preview}</p> : null}
        {log.error ? <p className="text-[#B91C1C]">error: {log.error}</p> : null}
        <p className="mt-2 text-[11px] font-semibold uppercase tracking-wide text-[#737373]">{copy.settings.tasksLogsNodeResults}</p>
        <TaskRunNodeResults nodeResults={log.node_results} />
      </div>
    </details>
  );
}

function TaskRunNodeResults(props: { nodeResults: TaskRunNodeResult[] | undefined }) {
  const { copy } = useWebLocale();
  const { nodeResults } = props;
  if (!nodeResults || nodeResults.length === 0) {
    return <p className="text-[#737373]">{copy.settings.tasksLogsNoNodeResults}</p>;
  }
  return (
    <div className="space-y-2">
      {nodeResults.map((node, index) => <TaskRunNodeCard key={`${node.node_id}-${index}`} node={node} />)}
    </div>
  );
}

function TaskRunNodeCard(props: { node: TaskRunNodeResult }) {
  const { node } = props;
  const orchestrationGroupOutput = parseOrchestrationGroupOutput(node.output);
  return (
    <details className="rounded-[10px] border border-[#E5E5E5] bg-white p-2">
      <summary className="cursor-pointer list-none text-[12px]">
        <span className="font-semibold">{node.completed_seq}</span>
        <span className="mx-2">{node.node_id}</span>
        <span className="text-[#737373]">{node.node_type}</span>
        <span className="mx-2 text-[#737373]">{node.status}</span>
        {node.branch_id ? <span className="font-mono text-[#737373]">{node.branch_id}</span> : null}
      </summary>
      <div className="mt-2 space-y-2 text-[12px] text-[#111111]">
        {node.preview ? <p>preview: {node.preview}</p> : null}
        {node.error ? <p className="text-[#B91C1C]">error: {node.error}</p> : null}
        <p className="text-[#525252]">started: {formatLogTime(node.started_at)}</p>
        <p className="text-[#525252]">finished: {formatLogTime(node.finished_at)}</p>
        {orchestrationGroupOutput ? <OrchestrationRoundsBlock output={orchestrationGroupOutput} /> : null}
        <TaskJSONBlock label="input" value={node.input} />
        <TaskJSONBlock label="output" value={node.output} />
      </div>
    </details>
  );
}

function TaskJSONBlock(props: { label: string; value: unknown }) {
  const { label, value } = props;
  return (
    <pre className="overflow-x-auto rounded-[8px] bg-[#F5F5F5] p-2 text-[11px] text-[#374151]">
      {label}: {formatJSON(value)}
    </pre>
  );
}

function formatJSON(value: unknown): string {
  if (value === undefined) {
    return 'undefined';
  }
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}

function formatLogTime(value: string | undefined): string {
  if (!value) {
    return '-';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toISOString();
}
