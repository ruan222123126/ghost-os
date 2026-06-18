import { Check, ChevronDown, ChevronRight, CircleX, LoaderCircle, Terminal } from "lucide-react";
import { useState } from "react";
import type { ToolCardTitleMode, ToolCardViewModel, ToolTone } from "./types";

interface ToolCardProps {
  card: ToolCardViewModel;
  defaultOpen?: boolean;
}

export function ToolCard({ card, defaultOpen = false }: ToolCardProps) {
  const [isOpen, setIsOpen] = useState(defaultOpen);
  const displayTitle = card.titleMode === "plain" ? card.title : buildToolStatusTitle(card.tone, card.title);
  const displayStatus = buildToolStatusLabel(card.tone, card.statusLabel);
  const ChevronIcon = isOpen ? ChevronDown : ChevronRight;

  return (
    <div className={`tool-card is-${card.tone}`}>
      <button
        type="button"
        className={`tool-card-button${isOpen ? " is-open" : ""} is-${card.tone}`}
        onClick={() => setIsOpen((current) => !current)}
      >
        <span className="tool-card-heading">
          {card.showTerminalIcon !== false ? (
            <Terminal className="tool-terminal-icon" aria-hidden="true" strokeWidth={1.7} />
          ) : null}
          <span className="tool-card-title" title={displayTitle}>
            {displayTitle}
          </span>
          <ChevronIcon className="tool-chevron" aria-hidden="true" strokeWidth={1.8} />
        </span>
      </button>

      {isOpen ? (
        <div className={`tool-details is-${card.tone}`}>
          <span className="tool-details-kind">命令</span>
          <div className="tool-details-command">{card.title}</div>
          {card.details ? (
            <div className="tool-details-output">
              <pre>{card.details}</pre>
            </div>
          ) : null}
          <div className={`tool-card-status is-${card.tone}`}>
            <ToolStatusIcon tone={card.tone} />
            <span className="tool-card-status-label">{displayStatus}</span>
          </div>
        </div>
      ) : null}
    </div>
  );
}

function ToolStatusIcon({ tone }: { tone: ToolTone }) {
  if (tone === "running") {
    return <LoaderCircle className="tool-status-icon tool-status-spinner" aria-hidden="true" strokeWidth={1.8} />;
  }
  if (tone === "error") {
    return <CircleX className="tool-status-icon" aria-hidden="true" strokeWidth={1.8} />;
  }
  return <Check className="tool-status-icon" aria-hidden="true" strokeWidth={1.8} />;
}

function buildToolStatusTitle(tone: ToolTone, title: string): string {
  switch (tone) {
    case "running":
      return `正在运行 ${title}`;
    case "error":
      return `运行出错 ${title}`;
    case "success":
      return `已运行 ${title}`;
  }
}

function buildToolStatusLabel(tone: ToolTone, statusLabel: string): string {
  if (tone === "running") {
    return "运行中";
  }
  return statusLabel;
}

export type { ToolCardTitleMode, ToolCardViewModel, ToolTone };
