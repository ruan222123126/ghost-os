// RSSBriefingPanel renders the latest persisted briefing and allows manual regeneration.

'use client';

import type { FC } from 'react';
import type { RSSBriefing } from '@/lib/types';

interface RSSBriefingPanelProps {
  briefing: RSSBriefing | null;
  loading: boolean;
  generating: boolean;
  error: string;
  onRefresh: () => void;
}

function formatBriefingTimestamp(value?: string): string {
  if (!value) {
    return 'No briefing yet';
  }
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }
  return new Intl.DateTimeFormat('en', {
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
  const updatedLabel = formatBriefingTimestamp(briefing?.saved_at ?? briefing?.generated_at);
  const canRefresh = !loading && !generating;

  return (
    <aside className="briefing-rail panel">
      <div className="panel-head briefing-head">
        <div className="panel-head-main">
          <p className="kicker">Signals</p>
          <h2>RSS Briefing</h2>
        </div>

        <div className="panel-head-actions">
          <span className="session-badge">Updated · {updatedLabel}</span>
          <button type="button" onClick={onRefresh} disabled={!canRefresh} className="button-secondary">
            {generating ? 'Generating…' : briefing ? 'Refresh' : 'Generate'}
          </button>
        </div>
      </div>

      {loading ? <div className="status-line info briefing-status">Loading latest briefing…</div> : null}
      {error ? <div className="status-line error briefing-status">{error}</div> : null}

      <div className="briefing-scroll ui-scroll">
        {!briefing && !loading ? (
          <div className="empty-state briefing-empty">
            <p className="briefing-empty-title">No persisted briefing yet.</p>
            <p className="briefing-empty-copy">Generate one to turn the current RSS inbox into a readable news brief.</p>
          </div>
        ) : null}

        {briefing ? (
          <div className="briefing-stack">
            <section className="briefing-hero panel-muted">
              <div className="briefing-hero-meta">
                <span className="status-chip status-chip-active">{briefing.highlight_count} highlights</span>
                <span className="status-chip">{briefing.scanned_groups} groups scanned</span>
                <span className="status-chip">{briefing.window_hours}h window</span>
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
                      <span>Why it matters</span>
                      {highlight.why_it_matters}
                    </p>
                  ) : null}

                  <div className="briefing-highlight-meta">
                    <span className={`status-chip ${highlight.importance === 'high' ? 'status-chip-active' : ''}`}>
                      {highlight.importance}
                    </span>
                    <span className="status-chip">{highlight.source_item_count} items</span>
                    <span className="status-chip">{highlight.source_feed_count} feeds</span>
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
