'use client';

import { useEffect, useMemo, useRef, useState, type RefObject } from 'react';
import { MessageRow } from '@/components/message/MessageRow';
import { IconPanelLeftClose, IconPanelLeftOpen, IconX } from '@/components/sessionSidebarIcons';
import { useLiveRunViewer } from '@/hooks/config/useLiveRunViewer';
import { buildMessageListRows } from '@/lib/chat-view/messageRows';
import type { MessageListRow } from '@/lib/chat-view/types';
import type { WebLocale } from '@/lib/i18n/locale';
import { useWebLocale } from '@/lib/i18n/provider';
import { buildLiveRunViewerOutputSignature } from '@/lib/liveRunViewerOutputSignature';
import { formatTaskRunStatus, formatTaskRunTimestamp } from '@/lib/taskRunDisplay';
import type { TaskRunLog } from '@/lib/types';
import type { LiveTaskRunCard } from '@/lib/taskRunViewerCards';
import { buildRelayCardSummary, type RelayCardSummary } from '@/lib/taskRunViewerSummary';

const OUTPUT_THEME_CLASS_NAME = `
  space-y-4 [&_.message-row]:m-0 [&_.message-row]:w-full [&_.message-row]:max-w-none [&_.message-row]:[animation:none]
  [&_.message-assistant-body]:max-w-none [&_.message-stack]:w-full [&_.tool-spinner]:[animation:none]
  [&_.thinking-indicator-status]:[animation:none]
`.replace(/\s+/g, ' ').trim();

export interface LiveRunViewerTarget { run: TaskRunLog; }
interface LiveRunViewerModalProps extends LiveRunViewerTarget { onClose: () => void; }

export function LiveRunViewerModal(props: LiveRunViewerModalProps) {
  const { copy } = useWebLocale();
  const { run, onClose } = props;
  const { cards, followLatest, output, selectedCard, selectCard, sourceSessionError, streamError } = useLiveRunViewer({ run });
  const outputRef = useRef<HTMLDivElement>(null);
  const [autoScroll, setAutoScroll] = useState(true);
  const [sidebarOpen, setSidebarOpen] = useState(true);
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
    <div className="fixed inset-0 z-[90] flex items-center justify-center p-3 sm:p-6" role="dialog" aria-modal="true">
      <button type="button" className="absolute inset-0 bg-black/20 backdrop-blur-[2px]" onClick={onClose} aria-label={copy.settings.tasksLogsClose} />
      <section className="relative z-[91] flex h-[90vh] w-full max-w-[1480px] flex-col overflow-hidden rounded-[16px] border border-[#E5E5E5] bg-[#FAFAFA] font-sans text-[#333333] shadow-2xl shadow-black/10">
        <ViewerTitleBar
          onClose={onClose}
          title={copy.settings.tasksLogsLiveViewTitle(run.run_id, run.session_id_output ?? '')}
        />
        <div className="flex min-h-0 flex-1 overflow-hidden">
          {sidebarOpen ? (
            <RunCardTimeline
              cards={cards}
              followLatest={followLatest}
              selectedCard={selectedCard}
              streamError={streamError}
              onSelectCard={selectCard}
            />
          ) : null}
          <div className="relative flex min-w-0 flex-1 flex-col rounded-tl-xl border-l border-t border-gray-200/50 bg-white shadow-sm">
            <WorkspaceHeader
              autoScroll={autoScroll}
              onToggleAutoScroll={() => setAutoScroll((value) => !value)}
              onToggleSidebar={() => setSidebarOpen((value) => !value)}
              run={run}
              selectedCard={selectedCard}
              sidebarOpen={sidebarOpen}
            />
            <div className="grid min-h-0 flex-1 grid-cols-1 grid-rows-[minmax(0,1fr)_minmax(180px,0.45fr)] overflow-hidden lg:grid-cols-[minmax(0,1fr)_minmax(300px,380px)] lg:grid-rows-1">
              <div className="min-h-0 min-w-0 overflow-hidden">
                <RunCardOutput
                  output={output}
                  outputRef={outputRef}
                  selectedCard={selectedCard}
                  sourceSessionError={sourceSessionError}
                />
              </div>
              <aside className="min-h-0 overflow-hidden border-t border-gray-100 bg-[#FCFCFC] lg:border-l lg:border-t-0">
                <RunCardSummary selectedCard={selectedCard} />
              </aside>
            </div>
          </div>
        </div>
        <ViewerStatusBar output={output} run={run} selectedCard={selectedCard} />
        <style>{`
          .log-detail-scrollbar::-webkit-scrollbar {
            width: 6px;
            height: 6px;
          }
          .log-detail-scrollbar::-webkit-scrollbar-track {
            background: transparent;
          }
          .log-detail-scrollbar::-webkit-scrollbar-thumb {
            background-color: rgba(0, 0, 0, 0.1);
            border-radius: 10px;
          }
          .log-detail-scrollbar::-webkit-scrollbar-thumb:hover {
            background-color: rgba(0, 0, 0, 0.2);
          }
        `}</style>
      </section>
    </div>
  );
}

function ViewerTitleBar(props: {
  onClose: () => void;
  title: string;
}) {
  const { copy } = useWebLocale();
  const { onClose, title } = props;
  return (
    <header className="flex h-12 flex-none select-none items-center justify-between border-b border-transparent px-4">
      <div className="flex min-w-0 items-center gap-3">
        <span className="font-semibold tracking-wider text-gray-700">Ghost-OS</span>
        <span className="text-sm text-gray-300">|</span>
        <span className="truncate text-sm text-gray-500">{title}</span>
      </div>
      <div className="flex items-center gap-4 sm:gap-6">
        <button
          type="button"
          className="grid h-8 w-8 place-items-center text-gray-400 transition-colors hover:text-red-500"
          onClick={onClose}
          aria-label={copy.settings.tasksLogsClose}
        >
          <IconX size={18} />
        </button>
      </div>
    </header>
  );
}

function RunCardTimeline(props: {
  cards: LiveTaskRunCard[];
  followLatest: boolean;
  selectedCard: LiveTaskRunCard | null;
  streamError: string;
  onSelectCard: (cardId: string) => void;
}) {
  const { copy, locale } = useWebLocale();
  const { cards, followLatest, selectedCard, streamError, onSelectCard } = props;
  return (
    <aside className="flex min-h-0 w-[280px] min-w-[280px] flex-col border-r border-gray-200/60 bg-[#FAFAFA]">
      <div className="flex items-center justify-between gap-3 px-4 py-3 text-xs text-gray-400">
        <span>{copy.settings.tasksLogsCardsCount(cards.length)}</span>
        <span>{followLatest ? copy.settings.tasksLogsLiveFollowing : copy.settings.tasksLogsLivePinned}</span>
      </div>
      {streamError ? <ViewerErrorBanner message={copy.settings.tasksLogsLiveStreamError(streamError)} /> : null}
      {cards.length === 0 ? (
        <p className="p-4 text-[12px] text-[#737373]">{copy.settings.tasksLogsRunCardsEmpty}</p>
      ) : (
        <ol className="log-detail-scrollbar min-h-0 flex-1 space-y-1 overflow-y-auto px-2 pb-4">
          {cards.map((card) => (
            <TimelineCard
              key={card.card_id}
              card={card}
              locale={locale}
              selected={selectedCard?.card_id === card.card_id}
              onSelect={() => onSelectCard(card.card_id)}
            />
          ))}
        </ol>
      )}
    </aside>
  );
}

function TimelineCard(props: { card: LiveTaskRunCard; locale: WebLocale; onSelect: () => void; selected: boolean }) {
  const { copy } = useWebLocale();
  const { card, locale, onSelect, selected } = props;
  const preview = card.preview?.trim() || card.error?.trim() || copy.settings.tasksLogsRunCardWaiting;
  return (
    <li>
      <button
        type="button"
        onClick={onSelect}
        className={[
          'w-full rounded-lg p-3 text-left transition-all duration-200',
          selected ? 'bg-[#F0F4F2] shadow-sm' : 'hover:bg-gray-100',
        ].join(' ')}
      >
        <div className="mb-1 flex items-start justify-between gap-3">
          <span className={selected ? 'truncate text-sm font-medium text-gray-900' : 'truncate text-sm font-medium text-gray-700'}>
            {cardTitle(card)}
          </span>
          <span className="whitespace-nowrap text-[11px] text-gray-400">{formatCardDate(card.started_at)}</span>
        </div>
        <p className="mb-1 line-clamp-2 text-[12px] leading-tight text-gray-400">{preview}</p>
        <div className="flex items-center gap-2 text-[11px] text-gray-300">
          <span>{formatCardClock(card.started_at)}</span>
          <span>·</span>
          <span className={statusTextClassName(card.status)}>
            {formatTaskRunStatus(card.status, locale)}
          </span>
          <span>·</span>
          <span className="truncate">{formatCardSubline(card, locale)}</span>
        </div>
      </button>
    </li>
  );
}

function WorkspaceHeader(props: {
  autoScroll: boolean;
  onToggleAutoScroll: () => void;
  onToggleSidebar: () => void;
  run: TaskRunLog;
  selectedCard: LiveTaskRunCard | null;
  sidebarOpen: boolean;
}) {
  const { copy, locale } = useWebLocale();
  const { autoScroll, onToggleAutoScroll, onToggleSidebar, run, selectedCard, sidebarOpen } = props;
  const title = selectedCard ? cardTitle(selectedCard) : copy.settings.tasksLogsAIOutput;
  return (
    <div className="flex flex-col justify-between border-b border-transparent px-8 pb-6 pt-6">
      <div className="flex items-start justify-between gap-4">
        <div className="flex min-w-0 flex-col">
          <div className="mb-2 flex min-w-0 items-center">
            <button
              type="button"
              className="mr-3 grid h-7 w-7 shrink-0 place-items-center text-gray-400 transition-colors hover:text-gray-600"
              onClick={onToggleSidebar}
              aria-label={sidebarOpen ? copy.settings.tasksLogsHideCards : copy.settings.tasksLogsShowCards}
            >
              {sidebarOpen ? <IconPanelLeftClose size={18} /> : <IconPanelLeftOpen size={18} />}
            </button>
            <span className="sr-only">{copy.settings.tasksLogsAIOutput}</span>
            <h2 className="truncate text-2xl font-semibold text-gray-800">{title}</h2>
          </div>
          <div className="ml-10 flex min-w-0 items-center gap-3 text-xs text-gray-400">
            <span className="whitespace-nowrap">{selectedCard ? formatCardMeta(selectedCard, locale) : formatTaskRunTimestamp(run.started_at ?? run.scheduled_at)}</span>
            <span>·</span>
            <span className={statusTextClassName(selectedCard?.status ?? run.status)}>
              {formatTaskRunStatus(selectedCard?.status ?? run.status, locale)}
            </span>
          </div>
        </div>
        <button
          type="button"
          onClick={onToggleAutoScroll}
          className={autoScroll ? 'rounded-md bg-[#F0F4F2] px-3 py-1.5 text-[11px] font-medium text-gray-800 shadow-sm' : 'rounded-md px-3 py-1.5 text-[11px] font-medium text-gray-500 hover:bg-gray-100 hover:text-gray-700'}
        >
          {copy.settings.tasksLogsAutoScroll(autoScroll)}
        </button>
      </div>
    </div>
  );
}

function RunCardOutput(props: {
  output: ReturnType<typeof useLiveRunViewer>['output'];
  outputRef: RefObject<HTMLDivElement>;
  selectedCard: LiveTaskRunCard | null;
  sourceSessionError: string;
}) {
  const { copy, locale } = useWebLocale();
  const { output, outputRef, selectedCard, sourceSessionError } = props;
  const [openMessageRows, setOpenMessageRows] = useState<Record<string, boolean>>({});
  const rows = buildMessageListRows({
    committedMessages: output.committedMessages,
    loadingOlderHistory: false,
    showThinkingIndicator: false,
    streamingRows: output.streamingRows,
    toolCard: {
      fallbackTitle: copy.chat.toolFallbackName,
      preparingDetails: copy.chat.toolPreparingOutput,
    },
  });

  useEffect(() => {
    setOpenMessageRows({});
  }, [selectedCard?.card_id]);

  const toggleCollapsibleMessageRow = (messageId: string) => {
    setOpenMessageRows((previous) => ({
      ...previous,
      [messageId]: !previous[messageId],
    }));
  };

  return (
    <section className="flex h-full min-h-0 flex-col bg-white">
      {sourceSessionError ? <ViewerErrorBanner message={copy.settings.tasksLogsLiveSourceError(sourceSessionError)} /> : null}
      <div ref={outputRef} className="log-detail-scrollbar min-h-0 flex-1 overflow-y-auto px-8 pb-10 pt-4">
        {rows.length === 0 ? (
          <WaitingOutput message={copy.settings.tasksLogsRunCardWaiting} meta={selectedCard ? formatCardMeta(selectedCard, locale) : ''} />
        ) : (
          <div className={OUTPUT_THEME_CLASS_NAME}>
            {rows.map((row, index) => {
              if (row.kind !== 'message') {
                return null;
              }
              const nextRow = rows[index + 1];
              const open = Boolean(openMessageRows[row.message.id]);
              return (
                <MessageRow
                  key={row.key}
                  message={row.message}
                  toolCard={row.toolCard}
                  assistantMarkdownEnabled
                  hasTrailingTool={isMessageListMessageRow(nextRow) && nextRow.message.kind === 'tool'}
                  isThinkingPanelOpen={open}
                  isToolCardOpen={open}
                  loading={false}
                  onAnswerQuestion={async () => undefined}
                  onCancelQuestion={async () => undefined}
                  onToggleThinkingPanel={toggleCollapsibleMessageRow}
                  onToggleToolCard={toggleCollapsibleMessageRow}
                />
              );
            })}
          </div>
        )}
      </div>
    </section>
  );
}

function RunCardSummary(props: { selectedCard: LiveTaskRunCard | null }) {
  const { copy, locale } = useWebLocale();
  const { selectedCard } = props;
  if (!selectedCard) {
    return (
      <div className="px-8 py-4 text-[13px] text-gray-400">
        {copy.settings.tasksLogsRunCardWaiting}
      </div>
    );
  }
  const relaySummary = buildRelayCardSummary(selectedCard);
  const summary = buildSummaryText(selectedCard, locale);
  return (
    <div className="log-detail-scrollbar h-full overflow-y-auto px-8 py-4 text-[14px] leading-relaxed text-gray-700">
      {relaySummary ? (
        <RelaySummaryDetails summary={relaySummary} />
      ) : summary ? (
        <pre className="whitespace-pre-wrap break-words font-sans leading-[1.8] text-[#333333]">{summary}</pre>
      ) : (
        <p className="text-gray-400">{copy.settings.tasksLogsSummaryEmpty}</p>
      )}
    </div>
  );
}

function RelaySummaryDetails(props: { summary: RelayCardSummary }) {
  const { summary } = props;
  const rows = [
    { key: 'did', label: '已完成：', content: summary.did },
    { key: 'nextStep', label: '下阶段：', content: summary.nextStep },
    { key: 'log', label: '交付：', content: summary.log },
  ].filter((row) => row.content);
  return (
    <div className="space-y-4 text-[#333333]">
      {rows.map((row) => (
        <p key={row.key} className="whitespace-pre-wrap break-words leading-[1.8]">
          <strong className="font-semibold text-gray-900">{row.label}</strong>
          {row.content}
        </p>
      ))}
    </div>
  );
}

function ViewerStatusBar(props: {
  output: ReturnType<typeof useLiveRunViewer>['output'];
  run: TaskRunLog;
  selectedCard: LiveTaskRunCard | null;
}) {
  const { locale } = useWebLocale();
  const { output, run, selectedCard } = props;
  return (
    <div className="flex h-7 flex-none select-none items-center justify-end border-t border-gray-100 bg-white px-4 text-[11px] text-gray-400">
      <div className="flex items-center gap-4">
        <span>{formatTaskRunStatus(selectedCard?.status ?? run.status, locale)}</span>
        <span>{formatOutputSize(output)}</span>
      </div>
    </div>
  );
}

function isMessageListMessageRow(row: MessageListRow | undefined): row is Extract<MessageListRow, { kind: 'message' }> {
  return row?.kind === 'message';
}

function ViewerErrorBanner(props: { message: string }) {
  const { message } = props;
  return (
    <div className="border-b border-[#FECACA] bg-[#FEF2F2] px-6 py-4">
      <div className="rounded-r-md border-l-[3px] border-[#F87171] pl-4">
        <p className="text-xs font-medium text-[#B91C1C]">Session Load Error</p>
        <p className="mt-1 font-mono text-[12px] leading-relaxed text-[#991B1B]">{message}</p>
      </div>
    </div>
  );
}

function WaitingOutput(props: { message: string; meta: string }) {
  const { message, meta } = props;
  return (
    <div className="space-y-2 text-[12px]">
      {meta ? <p className="text-gray-400">{meta}</p> : null}
      <div className="flex items-center gap-3 text-gray-500">
        <span>{message}</span>
        <span className="inline-block h-2.5 w-2.5 rounded-full bg-gray-300" />
      </div>
    </div>
  );
}

function cardTitle(card: LiveTaskRunCard): string {
  return card.title?.trim() || '运行卡片';
}

function formatCardSubline(card: LiveTaskRunCard, locale: WebLocale): string {
  const parts = [];
  if (card.round) {
    parts.push(`round ${card.round}`);
  }
  if (card.iteration) {
    parts.push(`iteration ${card.iteration}`);
  }
  return parts.join(' · ') || formatTaskRunStatus(card.status, locale);
}

function formatCardMeta(card: LiveTaskRunCard, locale: WebLocale): string {
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

function formatCardDate(value?: string): string {
  if (!value) {
    return '--';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return formatTaskRunTimestamp(value).slice(5, 10);
}

function statusTextClassName(status?: string): string {
  const normalized = status?.trim();
  if (normalized === 'error' || normalized === 'cancelled') {
    return 'text-red-400';
  }
  if (normalized === 'success') {
    return 'text-green-500';
  }
  if (normalized === 'awaiting_human') {
    return 'text-amber-500';
  }
  return 'text-gray-400';
}

function buildSummaryText(card: LiveTaskRunCard, locale: WebLocale): string {
  const lines = [
    `${formatTaskRunStatus(card.status, locale)} · ${formatCardMeta(card, locale)}`,
  ];
  if (card.preview?.trim()) {
    lines.push('', 'preview:', card.preview.trim());
  }
  if (card.final_text?.trim()) {
    lines.push('', 'final:', card.final_text.trim());
  }
  if (card.error?.trim()) {
    lines.push('', 'error:', card.error.trim());
  }
  return lines.join('\n').trim();
}

function formatOutputSize(output: ReturnType<typeof useLiveRunViewer>['output']): string {
  const textLength = output.committedMessages.reduce((total, message) => {
    if ('content' in message && typeof message.content === 'string') {
      return total + message.content.length;
    }
    return total;
  }, 0) + output.streamingRows.reduce((total, row) => total + row.message.content.length, 0);
  const kilobytes = Math.max(0.1, textLength / 1024);
  return `${kilobytes.toFixed(1)} KB`;
}
