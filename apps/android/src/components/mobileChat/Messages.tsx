import { forwardRef, useEffect, useRef, useState, type ReactNode } from "react";
import type {
  AgentPayload,
  ChatSelectedSkill,
  ExternalAgentApprovalDecision,
  MobileToolCard,
  StatusMessage,
} from "../../mobileTypes";
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
  onApproveExternalAgent?: (input: ExternalApprovalActionInput) => Promise<boolean>;
  reply: AgentPayload | undefined;
  status: StatusMessage;
}

interface ExternalApprovalActionInput {
  approvalId: string;
  decision: ExternalAgentApprovalDecision;
  sessionId: string;
}

export function AssistantReply(props: AssistantReplyProps) {
  const replyMessage = props.reply?.message.trim() ?? "";
  const thinkingText = props.reply?.thinking ?? "";
  const hasThinkingText = thinkingText.trim().length > 0;
  const thinkingActive = props.status.tone === "loading" && hasThinkingText;
  const replyFinal = props.status.tone !== "loading";
  const thinkingStartedAtMs = useThinkingStartedAtMs(thinkingActive);
  const [thinkingPanelOpen, toggleThinkingPanel] = useThinkingPanelOpen(thinkingText, replyMessage, thinkingActive);

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
        {props.reply?.tools?.length ? (
          <ToolCardList
            sessionId={props.reply.session_id}
            tools={props.reply.tools}
            onApproveExternalAgent={props.onApproveExternalAgent}
          />
        ) : null}
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
              <AssistantMarkdownContent
                content={replyMessage}
                final={replyFinal}
                showCopyButton={replyFinal}
              />
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

function ToolCardList(props: {
  onApproveExternalAgent?: (input: ExternalApprovalActionInput) => Promise<boolean>;
  sessionId?: string;
  tools: MobileToolCard[];
}) {
  return (
    <div className="tool-card-list">
      {props.tools.map((tool) => (
        <ToolCard
          key={tool.id}
          sessionId={props.sessionId}
          tool={tool}
          onApproveExternalAgent={props.onApproveExternalAgent}
        />
      ))}
    </div>
  );
}

function ToolCard(props: {
  onApproveExternalAgent?: (input: ExternalApprovalActionInput) => Promise<boolean>;
  sessionId?: string;
  tool: MobileToolCard;
}) {
  const [expanded, setExpanded] = useState(Boolean(props.tool.approvalId));
  const [pendingDecision, setPendingDecision] = useState<ExternalAgentApprovalDecision | "">("");
  const viewModel = buildMobileToolCardViewModel(props.tool);
  const displayTitle = viewModel.titleMode === "plain"
    ? viewModel.title
    : buildToolStatusTitle(viewModel.tone, viewModel.title);
  const displayStatus = buildToolStatusLabel(viewModel.tone, viewModel.statusLabel);
  const approvalDisabled = Boolean(
    pendingDecision || props.tool.approvalDecision || !props.sessionId?.trim() || !props.onApproveExternalAgent,
  );

  async function approve(decision: ExternalAgentApprovalDecision): Promise<void> {
    const approvalId = props.tool.approvalId?.trim();
    const sessionId = props.sessionId?.trim();
    if (!approvalId || !sessionId || !props.onApproveExternalAgent || approvalDisabled) {
      return;
    }
    setPendingDecision(decision);
    const ok = await props.onApproveExternalAgent({ approvalId, decision, sessionId });
    if (!ok) {
      setPendingDecision("");
    }
  }

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
          {props.tool.approvalId ? (
            <ApprovalActions
              disabled={approvalDisabled}
              pendingDecision={pendingDecision}
              onApprove={approve}
            />
          ) : null}
        </div>
      ) : null}
    </section>
  );
}

function ApprovalActions(props: {
  disabled: boolean;
  pendingDecision: ExternalAgentApprovalDecision | "";
  onApprove: (decision: ExternalAgentApprovalDecision) => Promise<void>;
}) {
  const actions: Array<{ decision: ExternalAgentApprovalDecision; label: string }> = [
    { decision: "approved", label: "批准一次" },
    { decision: "approved_for_session", label: "本会话" },
    { decision: "denied", label: "拒绝" },
    { decision: "abort", label: "中止" },
  ];

  return (
    <div className="tool-approval-actions">
      {actions.map((action) => (
        <button
          key={action.decision}
          type="button"
          disabled={props.disabled}
          onClick={() => void props.onApprove(action.decision)}
        >
          {props.pendingDecision === action.decision ? "处理中" : action.label}
        </button>
      ))}
    </div>
  );
}

interface ChatBubbleProps {
  children?: ReactNode;
  selectedSkill?: ChatSelectedSkill;
}

export const ChatBubble = forwardRef<HTMLDivElement, ChatBubbleProps>(function ChatBubble(props, ref) {
  const hasText = typeof props.children === "string"
    ? props.children.trim().length > 0
    : props.children !== null && props.children !== undefined;

  return (
    <div ref={ref} className="message-row user-row">
      <div className="user-bubble">
        {props.selectedSkill ? <div className="user-bubble-selected-skill">{props.selectedSkill.name}</div> : null}
        {hasText ? props.children : null}
      </div>
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

function useThinkingPanelOpen(
  thinkingText: string,
  replyMessage: string,
  thinkingActive: boolean,
): [boolean, () => void] {
  const initialHasThinkingText = thinkingText.trim().length > 0;
  const initialHasReplyMessage = replyMessage.trim().length > 0;
  const [open, setOpen] = useState(thinkingActive && initialHasThinkingText && !initialHasReplyMessage);
  const hadThinkingTextRef = useRef(initialHasThinkingText);
  const hadReplyMessageRef = useRef(initialHasReplyMessage);
  const wasThinkingActiveRef = useRef(thinkingActive);

  useEffect(() => {
    const hasThinkingText = thinkingText.trim().length > 0;
    const hasReplyMessage = replyMessage.trim().length > 0;

    if (!hasThinkingText) {
      setOpen(false);
    } else if (thinkingActive && !hadThinkingTextRef.current) {
      setOpen(true);
    }
    if (hasThinkingText && hasReplyMessage && !hadReplyMessageRef.current) {
      setOpen(false);
    }
    if (hasThinkingText && !thinkingActive && wasThinkingActiveRef.current) {
      setOpen(false);
    }

    hadThinkingTextRef.current = hasThinkingText;
    hadReplyMessageRef.current = hasReplyMessage;
    wasThinkingActiveRef.current = thinkingActive;
  }, [replyMessage, thinkingActive, thinkingText]);

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
