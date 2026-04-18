// RSSBriefingPanel renders the latest persisted briefing and allows manual regeneration.

'use client';

import type { FC } from 'react';
import { useWebLocale } from '@/lib/i18n/provider';
import type { RSSBriefing } from '@/lib/types';

interface RSSBriefingPanelProps {
  briefing: RSSBriefing | null;
  loading: boolean;
  generating: boolean;
  error: string;
  onRefresh: () => void;
}

function formatBriefingTimestamp(
  value: string | undefined,
  locale: ReturnType<typeof useWebLocale>['locale'],
  fallback: string,
): string {
  if (!value) {
    return fallback;
  }
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }
  return new Intl.DateTimeFormat(locale, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(parsed);
}

export const RSSBriefingPanel: FC<RSSBriefingPanelProps> = ({
  briefing,
  loading,
  generating,
  error,
  onRefresh,
}) => {
  const { copy, locale } = useWebLocale();
  const updatedLabel = formatBriefingTimestamp(briefing?.saved_at ?? briefing?.generated_at, locale, copy.system.briefingNoData);
  const canRefresh = !loading && !generating;

  return (
    <aside className="briefing-rail panel">
      <div className="panel-head briefing-head">
        <div className="panel-head-main">
          <p className="kicker">{copy.system.briefingSignals}</p>
          <h2>{copy.system.briefingTitle}</h2>
        </div>

        <div className="panel-head-actions">
          <span className="session-badge">{copy.system.briefingUpdated(updatedLabel)}</span>
          <button type="button" onClick={onRefresh} disabled={!canRefresh} className="button-secondary">
            {generating ? copy.system.briefingGenerating : briefing ? copy.system.briefingRefresh : copy.system.briefingGenerate}
          </button>
        </div>
      </div>

      {loading ? <div className="status-line info briefing-status">{copy.system.briefingLoading}</div> : null}
      {error ? <div className="status-line error briefing-status">{error}</div> : null}

      <div className="briefing-scroll ui-scroll">
        {!briefing && !loading ? (
          <div className="empty-state briefing-empty">
            <p className="briefing-empty-title">{copy.system.briefingNoPersistedTitle}</p>
            <p className="briefing-empty-copy">{copy.system.briefingNoPersistedCopy}</p>
          </div>
        ) : null}

        {briefing ? (
          <div className="briefing-stack">
            <section className="briefing-hero panel-muted">
              <div className="briefing-hero-meta">
                <span className="status-chip status-chip-active">{copy.system.briefingHighlights(briefing.highlight_count)}</span>
                <span className="status-chip">{copy.system.briefingGroupsScanned(briefing.scanned_groups)}</span>
                <span className="status-chip">{copy.system.briefingWindowHours(briefing.window_hours)}</span>
              </div>
              <h3>{briefing.title}</h3>
              {briefing.summary ? <p className="briefing-hero-copy">{briefing.summary}</p> : null}
            </section>

            <div className="briefing-highlight-list">
              {briefing.highlights.map((highlight) => (
                <article key={highlight.group_id} className="briefing-highlight panel-muted">
                  <div className="briefing-highlight-head">
                    <div className="briefing-rank">{highlight.rank}</div>
                    <div className="briefing-highlight-copy">
                      {highlight.topic_label ? <p className="briefing-topic">{highlight.topic_label}</p> : null}
                      <h4>{highlight.headline}</h4>
                    </div>
                  </div>

                  {highlight.summary ? <p className="briefing-highlight-summary">{highlight.summary}</p> : null}
                  {highlight.why_it_matters ? (
                    <p className="briefing-highlight-why">
                      <span>{copy.system.briefingWhyItMatters}</span>
                      {highlight.why_it_matters}
                    </p>
                  ) : null}

                  <div className="briefing-highlight-meta">
                    <span className={`status-chip ${highlight.importance === 'high' ? 'status-chip-active' : ''}`}>
                      {formatImportanceLabel(highlight.importance, copy)}
                    </span>
                    <span className="status-chip">{copy.system.briefingItems(highlight.source_item_count)}</span>
                    <span className="status-chip">{copy.system.briefingFeeds(highlight.source_feed_count)}</span>
                  </div>

                  {highlight.tags?.length ? (
                    <div className="briefing-tag-row">
                      {highlight.tags.map((tag) => (
                        <span key={`${highlight.group_id}-${tag}`} className="briefing-tag">
                          {tag}
                        </span>
                      ))}
                    </div>
                  ) : null}
                </article>
              ))}
            </div>
          </div>
        ) : null}
      </div>
    </aside>
  );
};

function formatImportanceLabel(
  importance: RSSBriefing['highlights'][number]['importance'],
  copy: ReturnType<typeof useWebLocale>['copy'],
): string {
  if (importance === 'high') {
    return copy.system.briefingImportanceHigh;
  }
  if (importance === 'low') {
    return copy.system.briefingImportanceLow;
  }
  return copy.system.briefingImportanceNormal;
}
