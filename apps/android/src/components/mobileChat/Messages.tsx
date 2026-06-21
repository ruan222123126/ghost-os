import { forwardRef, useEffect, useRef, useState, type ReactNode } from "react";
import type { AgentPayload, MobileToolCard, StatusMessage } from "../../mobileTypes";
import { buildMobileToolCardViewModel, type MobileToolTone } from "../../lib/mobileToolCardViewModel";
import { EMPTY_STATE_SUGGESTIONS } from "./data";
import { AssistantMarkdownContent } from "./AssistantMarkdownContent";
import { UiIcon } from "./icons";
import { MessageCopyButton } from "./MessageCopyButton";
import type { UiIconName } from "./types";

const THINKING_ELAPSED_UPDATE_MS = 1000;
const THINKING_ELAPSED_NEXT_TICK_BUFFER_MS = 16;

interface AssistantIntroProps {
  onSelectSuggestion: (value: string) => void;
}

export function AssistantIntro(props: AssistantIntroProps) {
  return (
    <div className="empty-state">
      <h2>有什么可以帮忙的？</h2>
      <div className="suggestion-grid">
        {EMPTY_STATE_SUGGESTIONS.map((suggestion) => (
          <SuggestionButton
            key={suggestion.text}
            icon={suggestion.icon}
            text={suggestion.text}
            tone={suggestion.tone}
            onClick={() => props.onSelectSuggestion(suggestion.text)}
          />
        ))}
      </div>
    </div>
  );
}

function SuggestionButton(props: {
  icon: UiIconName;
  text: string;
  tone: "blue" | "orange" | "purple" | "yellow";
  onClick: () => void;
}) {
  return (
    <button className="suggestion-button" type="button" onClick={props.onClick}>
      <span className={`suggestion-icon suggestion-icon-${props.tone}`}>
        <UiIcon name={props.icon} />
      </span>
      <span>{props.text}</span>
    </button>
  );
}

interface AssistantReplyProps {
  reply: AgentPayload | undefined;
  status: StatusMessage;
}

export function AssistantReply(props: AssistantReplyProps) {
  const replyMessage = props.reply?.message.trim() ?? "";
  const thinkingText = props.reply?.thinking ?? "";
  const hasThinkingText = thinkingText.trim().length > 0;
  const thinkingActive = props.status.tone === "loading" && hasThinkingText;
  const thinkingStartedAtMs = useThinkingStartedAtMs(thinkingActive);
  const [thinkingPanelOpen, toggleThinkingPanel] = useThinkingPanelOpen(thinkingText, replyMessage);

  if (!props.reply && props.status.tone !== "error") {
    return null;
  }

  return (
    <AssistantPanel ariaLive="polite">
      <div className="assistant-copy assistant-reply">
        {hasThinkingText ? (
          <ThinkingPanel
            active={thinkingActive}
            expanded={thinkingPanelOpen}
            startedAtMs={thinkingStartedAtMs}
            text={thinkingText}
            onToggleExpanded={toggleThinkingPanel}
          />
        ) : null}
        {props.reply?.tools?.length ? <ToolCardList tools={props.reply.tools} /> : null}
        {props.status.tone === "error" ? (
          <>
            {replyMessage ? <AssistantMarkdownContent content={replyMessage} /> : null}
            <p className="error-text">{props.status.text}</p>
            {replyMessage ? (
              <div className="assistant-reply-actions">
                <MessageCopyButton text={replyMessage} />
              </div>
            ) : null}
          </>
        ) : (
          <>
            {replyMessage ? (
              <AssistantMarkdownContent content={replyMessage} />
            ) : hasThinkingText ? null : (
              <p>{props.status.text}</p>
            )}
            {replyMessage ? (
              <div className="assistant-reply-actions">
                <MessageCopyButton text={replyMessage} />
              </div>
            ) : null}
          </>
        )}
      </div>
    </AssistantPanel>
  );
}

function ToolCardList(props: { tools: MobileToolCard[] }) {
  return (
    <div className="tool-card-list">
      {props.tools.map((tool) => (
        <ToolCard key={tool.id} tool={tool} />
      ))}
    </div>
  );
}

function ToolCard(props: { tool: MobileToolCard }) {
  const [expanded, setExpanded] = useState(false);
  const viewModel = buildMobileToolCardViewModel(props.tool);
  const displayTitle = viewModel.titleMode === "plain"
    ? viewModel.title
    : buildToolStatusTitle(viewModel.tone, viewModel.title);
  const displayStatus = buildToolStatusLabel(viewModel.tone, viewModel.statusLabel);

  return (
    <section className={`tool-card is-${viewModel.tone}`}>
      <button
        type="button"
        className={`tool-card-button${expanded ? " is-open" : ""} is-${viewModel.tone}`}
        aria-expanded={expanded}
        aria-busy={viewModel.tone === "running"}
        onClick={() => setExpanded((current) => !current)}
      >
        <span className="tool-card-heading">
          {viewModel.showTerminalIcon ? <UiIcon name="terminal" /> : null}
          <span className="tool-card-title" title={displayTitle}>{displayTitle}</span>
          <span className="tool-chevron">
            <UiIcon name="chevron-down" />
          </span>
        </span>
      </button>
      {expanded ? (
        <div className={`tool-details is-${viewModel.tone}`}>
          <span className="tool-details-kind">工具</span>
          <div className="tool-details-command">{viewModel.title}</div>
          {viewModel.details ? (
            <div className="tool-details-output">
              <pre>{viewModel.details}</pre>
            </div>
          ) : null}
          <div className={`tool-card-status is-${viewModel.tone}`}>
            {viewModel.tone === "running"
              ? <span className="tool-spinner" />
              : viewModel.tone === "error"
                ? <UiIcon name="x" />
                : <UiIcon name="check" />}
            <span className="tool-card-status-label">{displayStatus}</span>
          </div>
        </div>
      ) : null}
    </section>
  );
}

interface ChatBubbleProps {
  children: ReactNode;
}

export const ChatBubble = forwardRef<HTMLDivElement, ChatBubbleProps>(function ChatBubble(props, ref) {
  return (
    <div ref={ref} className="message-row user-row">
      <div className="user-bubble">{props.children}</div>
    </div>
  );
});

function AssistantPanel(props: { children: ReactNode; ariaLive?: "polite" }) {
  return (
    <div className="message-row assistant-row">
      <section className="assistant-panel" aria-live={props.ariaLive}>
        {props.children}
      </section>
    </div>
  );
}

function ThinkingPanel(props: {
  active: boolean;
  expanded: boolean;
  startedAtMs: number | null;
  text: string;
  onToggleExpanded: () => void;
}) {
  const title = props.active ? "正在思考" : "已思考";
  const titleClassName = [
    "thinking-panel-title",
    props.active ? "thinking-sweep-text" : "",
  ]
    .filter(Boolean)
    .join(" ");
  const panelClassName = [
    "thinking-panel",
    props.expanded ? "is-expanded" : "is-collapsed",
    props.active ? "is-active" : "is-complete",
  ].join(" ");

  return (
    <div className={panelClassName}>
      <button
        type="button"
        className="thinking-panel-toggle"
        aria-expanded={props.expanded}
        aria-busy={props.active}
        onClick={props.onToggleExpanded}
      >
        <span className="thinking-panel-heading">
          <span className={titleClassName}>{title}</span>
          {props.active ? <ThinkingElapsed className="thinking-panel-elapsed" startedAtMs={props.startedAtMs} /> : null}
          <span className="thinking-panel-chevron">
            <UiIcon name="chevron-down" />
          </span>
        </span>
      </button>
      {props.expanded ? <pre className="thinking-panel-content">{props.text}</pre> : null}
    </div>
  );
}

function ThinkingElapsed(props: { className: string; startedAtMs: number | null }) {
  const elapsedSeconds = useThinkingElapsedSeconds(props.startedAtMs);

  if (props.startedAtMs === null) {
    return null;
  }

  return <span className={props.className} aria-hidden="true">（{elapsedSeconds}s）</span>;
}

function buildToolStatusTitle(tone: MobileToolTone, title: string): string {
  switch (tone) {
    case "running":
      return `正在运行 ${title}`;
    case "error":
      return `运行失败 ${title}`;
    case "success":
      return `已运行 ${title}`;
    default:
      return title;
  }
}

function buildToolStatusLabel(tone: MobileToolTone, statusLabel: string): string {
  switch (tone) {
    case "running":
      return "运行中";
    case "error":
      return "失败";
    case "success":
      return statusLabel === "SUCCESS" ? "成功" : statusLabel;
    default:
      return statusLabel;
  }
}

function useThinkingPanelOpen(thinkingText: string, replyMessage: string): [boolean, () => void] {
  const [open, setOpen] = useState(false);
  const hadThinkingTextRef = useRef(false);
  const hadReplyMessageRef = useRef(false);

  useEffect(() => {
    const hasThinkingText = thinkingText.trim().length > 0;
    const hasReplyMessage = replyMessage.trim().length > 0;

    if (!hasThinkingText) {
      setOpen(false);
    } else if (hasReplyMessage && !hadReplyMessageRef.current) {
      setOpen(false);
    }

    hadThinkingTextRef.current = hasThinkingText;
    hadReplyMessageRef.current = hasReplyMessage;
  }, [replyMessage, thinkingText]);

  return [open, () => setOpen((current) => !current)];
}

function useThinkingStartedAtMs(active: boolean): number | null {
  const [startedAtMs, setStartedAtMs] = useState<number | null>(() => (active ? Date.now() : null));

  useEffect(() => {
    if (!active) {
      setStartedAtMs(null);
      return;
    }

    setStartedAtMs((current) => current ?? Date.now());
  }, [active]);

  return startedAtMs;
}

function useThinkingElapsedSeconds(startedAtMs: number | null): number {
  const [nowMs, setNowMs] = useState(() => Date.now());

  useEffect(() => {
    if (startedAtMs === null) {
      return;
    }

    let timeoutId: number | null = null;
    const update = () => {
      const now = Date.now();
      setNowMs(now);
      const elapsedMs = Math.max(0, now - startedAtMs);
      const delayMs = THINKING_ELAPSED_UPDATE_MS
        - (elapsedMs % THINKING_ELAPSED_UPDATE_MS)
        + THINKING_ELAPSED_NEXT_TICK_BUFFER_MS;
      timeoutId = window.setTimeout(update, delayMs);
    };

    update();

    return () => {
      if (timeoutId !== null) {
        window.clearTimeout(timeoutId);
      }
    };
  }, [startedAtMs]);

  if (startedAtMs === null) {
    return 0;
  }

  return Math.max(0, Math.floor((nowMs - startedAtMs) / THINKING_ELAPSED_UPDATE_MS));
}
