'use client';

import { useEffect, useMemo, useRef, type RefObject } from 'react';
import { CloseButton } from '@/components/CloseButton';
import { MessageRow } from '@/components/message/MessageRow';
import { useLiveRunViewer } from '@/hooks/config/useLiveRunViewer';
import { useWebLocale } from '@/lib/i18n/provider';
import type { TaskRunLog } from '@/lib/types';
import type { LiveTaskRunCard } from '@/lib/taskRunViewerCards';

export interface LiveRunViewerTarget {
  run: TaskRunLog;
}

interface LiveRunViewerModalProps extends LiveRunViewerTarget {
  onClose: () => void;
}

export function LiveRunViewerModal(props: LiveRunViewerModalProps) {
  const { copy } = useWebLocale();
  const { run, onClose } = props;
  const { cards, followLatest, output, selectedCard, selectCard, sourceSessionError, streamError } = useLiveRunViewer({ run });
  const outputRef = useRef<HTMLDivElement>(null);
  const outputSignature = useMemo(() => [
    selectedCard?.card_id ?? '',
    output.committedMessages.map((message) => message.id).join(','),
    output.streamingRows.map((row) => row.key).join(','),
  ].join('|'), [output.committedMessages, output.streamingRows, selectedCard?.card_id]);

  useEffect(() => {
    const node = outputRef.current;
    if (!node) {
      return;
    }
    node.scrollTop = node.scrollHeight;
  }, [outputSignature]);

  return (
    <div className="fixed inset-0 z-[90] flex items-center justify-center p-4" role="dialog" aria-modal="true">
      <button type="button" className="absolute inset-0 bg-black/35 backdrop-blur-[2px]" onClick={onClose} aria-label={copy.settings.tasksLogsClose} />
      <section className="relative z-[91] flex h-[82vh] w-full max-w-[1180px] flex-col overflow-hidden rounded-[18px] border border-[#E5E5E5] bg-white shadow-2xl">
        <header className="flex items-start justify-between gap-4 border-b border-[#E5E5E5] px-5 py-4">
          <div className="min-w-0">
            <h2 className="text-[15px] font-semibold text-[#111111]">{copy.settings.tasksLogsLiveViewTitle(run.run_id, run.session_id_output ?? '')}</h2>
            <p className="mt-1 break-all font-mono text-[11px] text-[#737373]">session_id: {run.session_id_output ?? '-'}</p>
            <p className="mt-1 break-all font-mono text-[11px] text-[#737373]">trace_id: {run.trace_id}</p>
          </div>
          <CloseButton onClick={onClose} className="shrink-0" aria-label={copy.settings.tasksLogsClose} />
        </header>
        <div className="grid min-h-0 flex-1 grid-cols-1 overflow-hidden lg:grid-cols-[300px_minmax(0,1fr)]">
          <RunCardTimeline
            cards={cards}
            followLatest={followLatest}
            selectedCard={selectedCard}
            streamError={streamError}
            onSelectCard={selectCard}
          />
          <RunCardOutput
            outputRef={outputRef}
            followLatest={followLatest}
            output={output}
            selectedCard={selectedCard}
            sourceSessionError={sourceSessionError}
          />
        </div>
      </section>
    </div>
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
    <section className="min-h-0 border-b border-[#E5E5E5] bg-[#FAFAFA] lg:border-b-0 lg:border-r">
      <div className="border-b border-[#E5E5E5] px-4 py-3">
        <div className="flex items-center justify-between gap-3">
          <h3 className="text-[12px] font-semibold text-[#111111]">{copy.settings.tasksLogsTimeline}</h3>
          <span className="rounded-full border border-[#E5E5E5] px-2 py-1 text-[10px] font-medium text-[#525252]">
            {followLatest ? copy.settings.tasksLogsLiveFollowing : copy.settings.tasksLogsLivePinned}
          </span>
        </div>
      </div>
      {streamError ? <p className="border-b border-[#FECACA] bg-[#FEF2F2] px-4 py-2 text-[11px] text-[#B91C1C]">{copy.settings.tasksLogsLiveStreamError(streamError)}</p> : null}
      {cards.length === 0 ? (
        <p className="p-4 text-[12px] text-[#737373]">{copy.settings.tasksLogsRunCardsEmpty}</p>
      ) : (
        <ol className="h-full max-h-full space-y-2 overflow-y-auto px-3 py-3">
          {cards.map((card) => {
            const selected = selectedCard?.card_id === card.card_id;
            return (
              <li key={card.card_id}>
                <button
                  type="button"
                  onClick={() => onSelectCard(card.card_id)}
                  className={[
                    'w-full rounded-[14px] border px-3 py-3 text-left transition-colors',
                    selected
                      ? 'border-[#111111] bg-white shadow-[0_8px_24px_rgba(17,17,17,0.08)]'
                      : 'border-[#E5E5E5] bg-white/80 hover:border-[#D4D4D4] hover:bg-white',
                  ].join(' ')}
                >
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <p className="truncate text-[12px] font-semibold text-[#111111]">{cardTitle(card)}</p>
                      <p className="mt-1 font-mono text-[10px] text-[#737373]">{formatCardMeta(card)}</p>
                    </div>
                    <span className={statusClassName(card.status)}>{card.status ?? 'running'}</span>
                  </div>
                  <p className="mt-2 line-clamp-3 text-[11px] text-[#525252]">{card.preview?.trim() || card.error?.trim() || copy.settings.tasksLogsRunCardWaiting}</p>
                </button>
              </li>
            );
          })}
        </ol>
      )}
    </section>
  );
}

function RunCardOutput(props: {
  followLatest: boolean;
  output: ReturnType<typeof useLiveRunViewer>['output'];
  outputRef: RefObject<HTMLDivElement>;
  selectedCard: LiveTaskRunCard | null;
  sourceSessionError: string;
}) {
  const { copy } = useWebLocale();
  const { followLatest, output, outputRef, selectedCard, sourceSessionError } = props;
  const rows = [...output.committedMessages.map((message) => ({ key: message.id, message })), ...output.streamingRows];

  return (
    <section className="min-h-0 bg-white">
      <div className="border-b border-[#E5E5E5] px-5 py-3">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <h3 className="text-[12px] font-semibold text-[#111111]">{copy.settings.tasksLogsAIOutput}</h3>
            <p className="mt-1 font-mono text-[10px] text-[#737373]">{selectedCard ? formatCardMeta(selectedCard) : '-'}</p>
          </div>
          {selectedCard ? <span className={statusClassName(selectedCard.status)}>{selectedCard.status ?? 'running'}</span> : null}
        </div>
      </div>
      {sourceSessionError ? <p className="border-b border-[#FECACA] bg-[#FEF2F2] px-5 py-2 text-[11px] text-[#B91C1C]">{copy.settings.tasksLogsLiveSourceError(sourceSessionError)}</p> : null}
      <div ref={outputRef} className="h-full max-h-full overflow-y-auto px-5 py-4">
        {rows.length === 0 ? (
          <p className="text-[12px] text-[#737373]">{copy.settings.tasksLogsRunCardWaiting}</p>
        ) : (
          <div className="space-y-3">
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

function cardTitle(card: LiveTaskRunCard): string {
  return card.title?.trim() || card.node_id?.trim() || card.card_id;
}

function formatCardMeta(card: LiveTaskRunCard): string {
  const parts = [card.started_at ? formatLogTime(card.started_at) : '-'];
  if (card.node_id?.trim()) {
    parts.push(card.node_id.trim());
  }
  if (card.round) {
    parts.push(`round ${card.round}`);
  }
  if (card.iteration) {
    parts.push(`iteration ${card.iteration}`);
  }
  if (card.branch_id?.trim()) {
    parts.push(card.branch_id.trim());
  }
  return parts.join(' · ');
}

function formatLogTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toISOString();
}

function statusClassName(status?: string): string {
  const value = status?.trim() || 'running';
  switch (value) {
    case 'success':
      return 'rounded-full border border-[#BBF7D0] bg-[#F0FDF4] px-2 py-1 text-[10px] font-semibold text-[#166534]';
    case 'awaiting_human':
      return 'rounded-full border border-[#FED7AA] bg-[#FFF7ED] px-2 py-1 text-[10px] font-semibold text-[#9A3412]';
    case 'error':
    case 'cancelled':
      return 'rounded-full border border-[#FECACA] bg-[#FEF2F2] px-2 py-1 text-[10px] font-semibold text-[#B91C1C]';
    default:
      return 'rounded-full border border-[#BFDBFE] bg-[#EFF6FF] px-2 py-1 text-[10px] font-semibold text-[#1D4ED8]';
  }
}
