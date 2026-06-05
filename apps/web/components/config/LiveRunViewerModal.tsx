'use client';

import { useEffect, useMemo, useRef, useState, type RefObject } from 'react';
import { CloseButton } from '@/components/CloseButton';
import { MessageRow } from '@/components/message/MessageRow';
import { useLiveRunViewer } from '@/hooks/config/useLiveRunViewer';
import { useWebLocale } from '@/lib/i18n/provider';
import { buildLiveRunViewerOutputSignature } from '@/lib/liveRunViewerOutputSignature';
import { formatTaskRunStatus, formatTaskRunTimestamp } from '@/lib/taskRunDisplay';
import type { TaskRunLog } from '@/lib/types';
import type { LiveTaskRunCard } from '@/lib/taskRunViewerCards';

const OUTPUT_THEME_CLASS_NAME = `
  space-y-4 [&_.message-row]:m-0 [&_.message-row]:w-full [&_.message-row]:[animation:none]
  [&_.message-assistant-body]:max-w-none [&_.tool-spinner]:[animation:none] [&_.thinking-indicator-status]:[animation:none]
`.replace(/\s+/g, ' ').trim();

export interface LiveRunViewerTarget { run: TaskRunLog; }
interface LiveRunViewerModalProps extends LiveRunViewerTarget { onClose: () => void; }

export function LiveRunViewerModal(props: LiveRunViewerModalProps) {
  const { copy } = useWebLocale();
  const { run, onClose } = props;
  const { cards, followLatest, output, selectedCard, selectCard, sourceSessionError, streamError } = useLiveRunViewer({ run });
  const outputRef = useRef<HTMLDivElement>(null);
  const [autoScroll, setAutoScroll] = useState(true);
  const outputSignature = useMemo(
    () => buildLiveRunViewerOutputSignature(selectedCard?.card_id ?? '', output),
    [output, selectedCard?.card_id],
  );

  useEffect(() => {
    if (!autoScroll) {
      return;
    }
    const node = outputRef.current;
    if (!node) {
      return;
    }
    node.scrollTop = node.scrollHeight;
  }, [autoScroll, outputSignature]);

  return (
    <div className="fixed inset-0 z-[90] flex items-center justify-center p-4 sm:p-6" role="dialog" aria-modal="true">
      <button type="button" className="absolute inset-0 bg-black/20 backdrop-blur-[2px]" onClick={onClose} aria-label={copy.settings.tasksLogsClose} />
      <section className="relative z-[91] flex h-[85vh] w-full max-w-[1400px] flex-col overflow-hidden rounded-[20px] border border-[#E5E5E5] bg-white text-[#262626] shadow-2xl shadow-black/10">
        <ViewerHeader run={run} title={copy.settings.tasksLogsLiveViewTitle(run.run_id, run.session_id_output ?? '')} onClose={onClose} />
        <div className="grid min-h-0 flex-1 grid-cols-1 overflow-hidden lg:grid-cols-[320px_minmax(0,1fr)]">
          <RunCardTimeline
            cards={cards}
            followLatest={followLatest}
            selectedCard={selectedCard}
            streamError={streamError}
            onSelectCard={selectCard}
          />
          <RunCardOutput
            autoScroll={autoScroll}
            output={output}
            outputRef={outputRef}
            selectedCard={selectedCard}
            sourceSessionError={sourceSessionError}
            onToggleAutoScroll={() => setAutoScroll((value) => !value)}
          />
        </div>
      </section>
    </div>
  );
}

function ViewerHeader(props: { onClose: () => void; run: TaskRunLog; title: string }) {
  const { copy } = useWebLocale();
  const { onClose, run, title } = props;
  return (
    <header className="flex h-14 flex-none items-center justify-between border-b border-[#F0F0F0] bg-[#FAFAFA] px-4">
      <div className="min-w-0">
        <div className="flex items-center gap-4">
          <div className="flex min-w-0 items-center gap-2 text-sm font-medium text-[#111111]">
            <span className="grid h-4 w-4 place-items-center rounded-sm border border-sky-300 font-mono text-[10px] text-sky-600">{'>'}</span>
            <span className="truncate tracking-wide">{title}</span>
          </div>
          <div className="hidden h-4 w-px bg-[#E5E5E5] sm:block" />
          <div className="hidden items-center gap-2 font-mono text-[11px] text-[#737373] sm:flex">
            <ViewerBadge label="session_id" value={run.session_id_output ?? '-'} />
            <ViewerBadge label="trace_id" value={run.trace_id || '-'} />
          </div>
        </div>
      </div>
      <CloseButton onClick={onClose} className="shrink-0 bg-transparent p-1.5 text-[#737373] hover:bg-[#F5F5F5] hover:text-[#111111]" aria-label={copy.settings.tasksLogsClose} />
    </header>
  );
}

function ViewerBadge(props: { label: string; value: string }) {
  const { label, value } = props;
  return (
    <span className="flex items-center gap-1.5 rounded-md border border-[#EAEAEA] bg-white px-2 py-0.5"><span className="text-[#A3A3A3]">{label}:</span><span className="max-w-[220px] truncate text-[#525252]"> {value}</span></span>
  );
}

function RunCardTimeline(props: {
  cards: LiveTaskRunCard[];
  followLatest: boolean;
  selectedCard: LiveTaskRunCard | null;
  streamError: string;
  onSelectCard: (cardId: string) => void;
}) {
  const { copy } = useWebLocale();
  const { cards, followLatest, selectedCard, streamError, onSelectCard } = props;
  return (
    <aside className="flex min-h-0 flex-col border-b border-[#F0F0F0] bg-[#FCFCFC] lg:border-b-0 lg:border-r lg:border-[#F0F0F0]">
      <div className="border-b border-[#F0F0F0] bg-[#FAFAFA] px-4 py-3">
        <div className="flex items-center justify-between gap-3">
          <h3 className="text-[11px] font-semibold uppercase tracking-[0.16em] text-[#737373]">{copy.settings.tasksLogsTimeline}</h3>
          <span className="rounded-md border border-[#EAEAEA] bg-white px-2 py-1 text-[10px] font-medium text-[#525252]">
            {followLatest ? copy.settings.tasksLogsLiveFollowing : copy.settings.tasksLogsLivePinned}
          </span>
        </div>
      </div>
      {streamError ? <ViewerErrorBanner message={copy.settings.tasksLogsLiveStreamError(streamError)} /> : null}
      {cards.length === 0 ? (
        <p className="p-4 text-[12px] text-[#737373]">{copy.settings.tasksLogsRunCardsEmpty}</p>
      ) : (
        <ol className="min-h-0 flex-1 overflow-y-auto px-0 py-2">
          {cards.map((card) => (
            <TimelineCard key={card.card_id} card={card} selected={selectedCard?.card_id === card.card_id} onSelect={() => onSelectCard(card.card_id)} />
          ))}
        </ol>
      )}
    </aside>
  );
}

function TimelineCard(props: { card: LiveTaskRunCard; onSelect: () => void; selected: boolean }) {
  const { copy, locale } = useWebLocale();
  const { card, onSelect, selected } = props;
  return (
    <li>
      <button
        type="button"
        onClick={onSelect}
        className={[
          'group flex w-full items-start gap-3 border-l-2 px-4 py-3 text-left',
          selected
            ? 'border-sky-500 bg-gradient-to-r from-sky-50 to-white'
            : 'border-transparent hover:bg-[#FAFAFA]',
        ].join(' ')}
      >
        <StatusIndicator status={card.status} />
        <div className="min-w-0 flex-1">
          <div className="mb-1 flex items-baseline justify-between gap-3">
            <span className={selected ? 'truncate text-xs font-medium text-[#111111]' : 'truncate text-xs font-medium text-[#525252] group-hover:text-[#111111]'}>{cardTitle(card)}</span>
            <span className={selected ? 'font-mono text-[10px] text-sky-600' : 'font-mono text-[10px] text-[#A3A3A3]'}>{formatCardClock(card.started_at)}</span>
          </div>
          <div className={selected ? 'text-[11px] font-mono text-[#525252]' : 'text-[11px] font-mono text-[#A3A3A3]'}>{formatCardSubline(card, locale)}</div>
          <p className={selected ? 'mt-2 line-clamp-2 text-[11px] text-sky-700' : 'mt-2 line-clamp-2 text-[11px] text-[#737373]'}>{card.preview?.trim() || card.error?.trim() || copy.settings.tasksLogsRunCardWaiting}</p>
        </div>
      </button>
    </li>
  );
}

function RunCardOutput(props: {
  autoScroll: boolean;
  output: ReturnType<typeof useLiveRunViewer>['output'];
  outputRef: RefObject<HTMLDivElement>;
  selectedCard: LiveTaskRunCard | null;
  sourceSessionError: string;
  onToggleAutoScroll: () => void;
}) {
  const { copy, locale } = useWebLocale();
  const { autoScroll, output, outputRef, selectedCard, sourceSessionError, onToggleAutoScroll } = props;
  const rows = [...output.committedMessages.map((message) => ({ key: message.id, message })), ...output.streamingRows];
  return (
    <section className="flex min-h-0 flex-col overflow-hidden bg-white">
      <div className="flex items-center justify-between gap-3 border-b border-[#F0F0F0] px-6 py-3">
        <div className="min-w-0 font-mono text-xs text-[#737373]">
          <span>{copy.settings.tasksLogsAIOutput}</span>
          <span className="mx-2 text-[#D4D4D4]">{'>'}</span>
          <span className="truncate text-[#262626]">{selectedCard ? cardTitle(selectedCard) : copy.settings.tasksLogsAIOutput}</span>
        </div>
        <button
          type="button"
          onClick={onToggleAutoScroll}
          className={autoScroll ? 'rounded-md bg-sky-50 px-2.5 py-1 text-[11px] font-medium text-sky-700' : 'rounded-md px-2.5 py-1 text-[11px] font-medium text-[#737373] hover:bg-[#F5F5F5] hover:text-[#111111]'}
        >
          Auto-scroll {autoScroll ? 'ON' : 'OFF'}
        </button>
      </div>
      {sourceSessionError ? <ViewerErrorBanner message={copy.settings.tasksLogsLiveSourceError(sourceSessionError)} /> : null}
      <div ref={outputRef} className="min-h-0 flex-1 overflow-y-auto px-6 pt-5 pb-10">
        {rows.length === 0 ? (
          <WaitingOutput message={copy.settings.tasksLogsRunCardWaiting} meta={selectedCard ? formatCardMeta(selectedCard, locale) : ''} />
        ) : (
          <div className={OUTPUT_THEME_CLASS_NAME}>
            {rows.map((row, index) => (
              <MessageRow
                key={row.key}
                message={row.message}
                assistantMarkdownEnabled
                toolCallCompactOutputEnabled={false}
                hasTrailingTool={rows[index + 1]?.message.kind === 'tool'}
                isThinkingPanelOpen
                isToolCardOpen
                loading={false}
                onAnswerQuestion={async () => undefined}
                onCancelQuestion={async () => undefined}
              />
            ))}
          </div>
        )}
      </div>
    </section>
  );
}

function ViewerErrorBanner(props: { message: string }) {
  const { message } = props;
  return (
    <div className="border-b border-[#FECACA] bg-[#FEF2F2] px-6 py-4">
      <div className="rounded-r-md border-l-[3px] border-[#F87171] pl-4"><p className="text-xs font-medium text-[#B91C1C]">Session Load Error</p><p className="mt-1 font-mono text-[12px] leading-relaxed text-[#991B1B]">{message}</p></div>
    </div>
  );
}

function WaitingOutput(props: { message: string; meta: string }) {
  const { message, meta } = props;
  return (
    <div className="space-y-2 font-mono text-[12px]">
      {meta ? <p className="text-[#737373]">{meta}</p> : null}
      <div className="flex items-center gap-3 text-sky-700"><span>{message}</span><span className="inline-block h-2.5 w-2.5 rounded-full bg-sky-600" /></div>
    </div>
  );
}

function StatusIndicator(props: { status?: string }) {
  const status = props.status?.trim() || 'running';
  if (status === 'success') {
    return <span className="mt-0.5 h-4 w-4 rounded-full border border-emerald-200 bg-emerald-50 text-center text-[10px] leading-[14px] text-emerald-600">✓</span>;
  }
  if (status === 'error' || status === 'cancelled') {
    return <span className="mt-0.5 h-4 w-4 rounded-full border border-rose-200 bg-rose-50 text-center text-[10px] leading-[14px] text-rose-600">!</span>;
  }
  if (status === 'awaiting_human') {
    return <span className="mt-0.5 h-4 w-4 rounded-full border border-amber-200 bg-amber-50 text-center text-[10px] leading-[14px] text-amber-600">?</span>;
  }
  return <span className="mt-1 h-2.5 w-2.5 rounded-full bg-sky-500" />;
}

function cardTitle(card: LiveTaskRunCard): string {
  return card.title?.trim() || card.node_id?.trim() || card.card_id;
}

function formatCardSubline(card: LiveTaskRunCard, locale: ReturnType<typeof useWebLocale>['locale']): string {
  const parts = [];
  if (card.round) {
    parts.push(`round ${card.round}`);
  }
  if (card.iteration) {
    parts.push(`iteration ${card.iteration}`);
  }
  if (card.node_id?.trim()) {
    parts.push(card.node_id.trim());
  }
  if (card.branch_id?.trim()) {
    parts.push(card.branch_id.trim());
  }
  return parts.join(' · ') || formatTaskRunStatus(card.status, locale);
}

function formatCardMeta(card: LiveTaskRunCard, locale: ReturnType<typeof useWebLocale>['locale']): string {
  const parts = [formatCardClock(card.started_at)];
  const subline = formatCardSubline(card, locale);
  if (subline) {
    parts.push(subline);
  }
  return parts.join(' · ');
}

function formatCardClock(value?: string): string {
  if (!value) {
    return '--:--:--';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return formatTaskRunTimestamp(value).slice(11);
}
