'use client';

import { useMemo } from 'react';
import { CloseButton } from '@/components/CloseButton';
import { useLiveRunViewer } from '@/hooks/config/useLiveRunViewer';
import { useWebLocale } from '@/lib/i18n/provider';
import type { SessionDetail, SessionMessage, SessionPushEvent } from '@/lib/types';

const RECENT_MESSAGE_LIMIT = 5;

export interface LiveRunViewerTarget {
  sessionId: string;
  traceId?: string;
  runId?: string;
}

interface LiveRunViewerModalProps extends LiveRunViewerTarget {
  onClose: () => void;
}

export function LiveRunViewerModal(props: LiveRunViewerModalProps) {
  const { copy } = useWebLocale();
  const { sessionId, traceId, runId, onClose } = props;
  const { events, session, streamError, snapshotError } = useLiveRunViewer({ sessionId });
  const recentMessages = useMemo(() => recentSessionMessages(session), [session]);

  return (
    <div className="fixed inset-0 z-[90] flex items-center justify-center p-4" role="dialog" aria-modal="true">
      <button type="button" className="absolute inset-0 bg-black/35 backdrop-blur-[2px]" onClick={onClose} aria-label={copy.settings.tasksLogsClose} />
      <section className="relative z-[91] flex h-[82vh] w-full max-w-[1040px] flex-col overflow-hidden rounded-[18px] border border-[#E5E5E5] bg-white shadow-2xl">
        <header className="flex items-start justify-between gap-4 border-b border-[#E5E5E5] px-5 py-4">
          <div className="min-w-0">
            <h2 className="text-[15px] font-semibold text-[#111111]">{copy.settings.tasksLogsLiveViewTitle(runId ?? '', sessionId)}</h2>
            <p className="mt-1 break-all font-mono text-[11px] text-[#737373]">session_id: {sessionId}</p>
            {traceId ? <p className="mt-1 break-all font-mono text-[11px] text-[#737373]">trace_id: {traceId}</p> : null}
          </div>
          <CloseButton
            onClick={onClose}
            className="shrink-0"
            aria-label={copy.settings.tasksLogsClose}
          />
        </header>
        <div className="grid min-h-0 flex-1 grid-cols-1 gap-3 overflow-hidden p-4 lg:grid-cols-2">
          <LiveEventPanel events={events} error={streamError} />
          <LiveSnapshotPanel session={session} messages={recentMessages} error={snapshotError} />
        </div>
      </section>
    </div>
  );
}

function LiveEventPanel(props: { events: SessionPushEvent[]; error: string }) {
  const { copy } = useWebLocale();
  const { events, error } = props;
  return (
    <section className="min-h-0 overflow-hidden rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA]">
      <h3 className="border-b border-[#E5E5E5] px-3 py-2 text-[12px] font-semibold text-[#111111]">{copy.settings.tasksLogsLiveEvents}</h3>
      {error ? <p className="border-b border-[#FECACA] bg-[#FEF2F2] px-3 py-2 text-[11px] text-[#B91C1C]">{copy.settings.tasksLogsLiveStreamError(error)}</p> : null}
      {events.length === 0 ? (
        <p className="p-3 text-[12px] text-[#737373]">{copy.settings.tasksLogsLiveNoEvents}</p>
      ) : (
        <ol className="h-full max-h-full space-y-2 overflow-y-auto p-3">
          {events.map((event) => <LiveEventItem key={event.id} event={event} />)}
        </ol>
      )}
    </section>
  );
}

function LiveEventItem(props: { event: SessionPushEvent }) {
  const { event } = props;
  return (
    <li className="rounded-[8px] border border-[#E5E5E5] bg-white p-2 text-[11px] text-[#111111]">
      <p className="font-mono text-[#525252]">{event.type} · {event.at ?? '-'}</p>
      <pre className="mt-2 max-h-32 overflow-auto whitespace-pre-wrap break-words rounded-[6px] bg-[#F5F5F5] p-2 text-[#374151]">{formatJSON(event.payload)}</pre>
    </li>
  );
}

function LiveSnapshotPanel(props: { session: SessionDetail | null; messages: SessionMessage[]; error: string }) {
  const { copy } = useWebLocale();
  const { session, messages, error } = props;
  return (
    <section className="min-h-0 overflow-hidden rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA]">
      <h3 className="border-b border-[#E5E5E5] px-3 py-2 text-[12px] font-semibold text-[#111111]">{copy.settings.tasksLogsLiveSnapshot}</h3>
      {error ? <p className="border-b border-[#FECACA] bg-[#FEF2F2] px-3 py-2 text-[11px] text-[#B91C1C]">{copy.settings.tasksLogsLiveSnapshotError(error)}</p> : null}
      <div className="h-full max-h-full overflow-y-auto p-3 text-[12px] text-[#111111]">
        <p className="text-[11px] text-[#737373]">{copy.settings.tasksLogsLiveUpdatedAt}: {session?.updated_at ?? '-'}</p>
        <p className="mt-1 text-[11px] text-[#737373]">{copy.settings.tasksLogsLiveMessageCount}: {session?.message_count ?? 0}</p>
        <TaskDraftBlock draft={session?.turn_draft ?? null} />
        <RecentMessages messages={messages} />
      </div>
    </section>
  );
}

function TaskDraftBlock(props: { draft: unknown }) {
  const { copy } = useWebLocale();
  const { draft } = props;
  return (
    <div className="mt-3">
      <p className="mb-1 text-[11px] font-semibold uppercase tracking-wide text-[#737373]">turn_draft</p>
      <pre className="max-h-64 overflow-auto whitespace-pre-wrap break-words rounded-[8px] bg-white p-2 text-[11px] text-[#374151]">
        {draft ? formatJSON(draft) : copy.settings.tasksLogsLiveNoDraft}
      </pre>
    </div>
  );
}

function RecentMessages(props: { messages: SessionMessage[] }) {
  const { copy } = useWebLocale();
  const { messages } = props;
  return (
    <div className="mt-3">
      <p className="mb-1 text-[11px] font-semibold uppercase tracking-wide text-[#737373]">{copy.settings.tasksLogsLiveRecentMessages}</p>
      <ol className="space-y-1">
        {messages.map((message) => (
          <li key={message.index} className="rounded-[8px] bg-white px-2 py-1 text-[11px] text-[#374151]">
            <span className="font-mono text-[#737373]">#{message.index} {message.role}: </span>
            {messagePreview(message)}
          </li>
        ))}
      </ol>
    </div>
  );
}

function recentSessionMessages(session: SessionDetail | null): SessionMessage[] {
  if (!session || session.messages.length <= RECENT_MESSAGE_LIMIT) {
    return session?.messages ?? [];
  }
  return session.messages.slice(session.messages.length - RECENT_MESSAGE_LIMIT);
}

function messagePreview(message: SessionMessage): string {
  const text = message.text?.trim();
  if (text) {
    return text;
  }
  const content = message.content?.map((part) => part.text?.trim()).filter(Boolean).join(' ');
  return content || '(empty)';
}

function formatJSON(value: unknown): string {
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}
