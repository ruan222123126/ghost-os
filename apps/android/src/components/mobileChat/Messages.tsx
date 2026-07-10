import { useVirtualizer } from "@tanstack/react-virtual";
import {
  forwardRef,
  memo,
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
  type MutableRefObject,
  type ReactNode,
  type RefObject,
} from "react";
import type {
  AgentPayload,
  ChatSelectedSkill,
  ExternalAgentApprovalDecision,
  MobileAssistantPart,
  MobileConversationMessage,
  MobileToolCard,
  StatusMessage,
} from "../../mobileTypes";
import { areMobileConversationMessagesEqual } from "../../lib/mobileConversationMessageEquality";
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
  const replyMessage = props.reply?.message ?? "";
  const replyMessageTrimmed = replyMessage.trim();
  const replyParts = resolveAssistantParts(props.reply);
  const thinkingText = props.reply?.thinking ?? "";
  const hasThinkingText = thinkingText.trim().length > 0;
  const thinkingActive = props.status.tone === "loading" && hasThinkingText;
  const replyFinal = props.status.tone !== "loading";
  const thinkingStartedAtMs = useThinkingStartedAtMs(thinkingActive);
  const [thinkingPanelOpen, toggleThinkingPanel] = useThinkingPanelOpen(thinkingText, replyMessageTrimmed, thinkingActive);

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
        {props.status.tone === "error" ? (
          <>
            <AssistantReplyParts
              final={replyFinal}
              parts={replyParts}
              sessionId={props.reply?.session_id}
              onApproveExternalAgent={props.onApproveExternalAgent}
            />
            <p className="error-text">{props.status.text}</p>
            {replyMessageTrimmed ? (
              <div className="assistant-reply-actions">
                <MessageCopyButton text={replyMessage} />
              </div>
            ) : null}
          </>
        ) : (
          <>
            {replyParts.length ? (
              <AssistantReplyParts
                final={replyFinal}
                parts={replyParts}
                sessionId={props.reply?.session_id}
                onApproveExternalAgent={props.onApproveExternalAgent}
              />
            ) : hasThinkingText ? null : (
              <p>{props.status.text}</p>
            )}
            {replyMessageTrimmed ? (
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

export const ConversationMessageList = memo(function ConversationMessageList(props: {
  messages: MobileConversationMessage[];
  onApproveExternalAgent?: (input: ExternalApprovalActionInput) => Promise<boolean>;
  postSendFocusRequest?: { messageId: string; token: number } | null;
  registerUserMessageRow: (messageId: string) => (node: HTMLDivElement | null) => void;
  reply: AgentPayload | undefined;
  scrollElementRef?: RefObject<HTMLElement | null>;
  scrollToBottomRef?: MutableRefObject<(() => void) | null>;
  status: StatusMessage;
}) {
  const hasActiveReply = Boolean(props.reply) || props.status.tone === "error";
  const itemCount = props.messages.length + (hasActiveReply ? 1 : 0);
  const getItemKey = useCallback((index: number) => {
    const message = props.messages[index];
    return message?.id ?? `active-reply:${props.reply?.session_id ?? "status"}`;
  }, [props.messages, props.reply?.session_id]);
  const virtualizer = useVirtualizer({
    anchorTo: "start",
    count: itemCount,
    estimateSize: estimateConversationRowSize,
    followOnAppend: false,
    getItemKey,
    getScrollElement: () => props.scrollElementRef?.current ?? null,
    overscan: 8,
    useAnimationFrameWithResizeObserver: true,
  });
  const virtualItems = virtualizer.getVirtualItems();
  const postSendMessageIndex = useMemo(() => {
    const messageId = props.postSendFocusRequest?.messageId;
    return messageId ? props.messages.findIndex((message) => message.id === messageId) : -1;
  }, [props.messages, props.postSendFocusRequest?.messageId]);
  const scrollToVirtualBottom = useCallback(() => {
    virtualizer.scrollToEnd({ behavior: "smooth" });
  }, [virtualizer]);

  useLayoutEffect(() => {
    const scrollToBottomRef = props.scrollToBottomRef;
    if (!scrollToBottomRef) {
      return;
    }
    scrollToBottomRef.current = scrollToVirtualBottom;
    return () => {
      if (scrollToBottomRef.current === scrollToVirtualBottom) {
        scrollToBottomRef.current = null;
      }
    };
  }, [props.scrollToBottomRef, scrollToVirtualBottom]);

  useLayoutEffect(() => {
    if (!props.scrollElementRef || postSendMessageIndex < 0 || !props.postSendFocusRequest) {
      return;
    }
    virtualizer.scrollToIndex(postSendMessageIndex, { align: "start", behavior: "auto" });
  }, [postSendMessageIndex, props.postSendFocusRequest?.token, props.scrollElementRef, virtualizer]);

  if (itemCount === 0) {
    return null;
  }

  if (!props.scrollElementRef) {
    return (
      <div className="conversation-list" data-chat-feed-content="">
        {props.messages.map((message) => (
          <ConversationListItemRow
            key={message.id}
            item={{ kind: "message", message }}
            onApproveExternalAgent={props.onApproveExternalAgent}
            registerUserMessageRow={props.registerUserMessageRow}
          />
        ))}
        {hasActiveReply ? (
          <ConversationListItemRow
            key={getItemKey(props.messages.length)}
            item={{ kind: "reply", reply: props.reply, status: props.status }}
            onApproveExternalAgent={props.onApproveExternalAgent}
            registerUserMessageRow={props.registerUserMessageRow}
          />
        ) : null}
      </div>
    );
  }

  return (
    <div
      className="conversation-list"
      data-chat-feed-content=""
      style={{ height: virtualizer.getTotalSize() }}
    >
      {virtualItems.map((virtualItem) => {
        const message = props.messages[virtualItem.index];
        const item: ConversationListItem = message
          ? { kind: "message", message }
          : { kind: "reply", reply: props.reply, status: props.status };
        return (
          <div
            key={virtualItem.key}
            ref={virtualizer.measureElement}
            className="conversation-virtual-row"
            data-index={virtualItem.index}
            style={{ transform: `translateY(${virtualItem.start}px)` }}
          >
            <ConversationListItemRow
              item={item}
              onApproveExternalAgent={props.onApproveExternalAgent}
              registerUserMessageRow={props.registerUserMessageRow}
            />
          </div>
        );
      })}
    </div>
  );
});

type ConversationListItem =
  | { kind: "message"; message: MobileConversationMessage }
  | { kind: "reply"; reply: AgentPayload | undefined; status: StatusMessage };

function estimateConversationRowSize(index: number): number {
  return index === 0 ? 96 : 132;
}

function ConversationListItemRow(props: {
  item: ConversationListItem;
  onApproveExternalAgent?: (input: ExternalApprovalActionInput) => Promise<boolean>;
  registerUserMessageRow: (messageId: string) => (node: HTMLDivElement | null) => void;
}) {
  if (props.item.kind === "reply") {
    return (
      <AssistantReply
        reply={props.item.reply}
        status={props.item.status}
        onApproveExternalAgent={props.onApproveExternalAgent}
      />
    );
  }

  return (
    <ConversationMessageRow
      message={props.item.message}
      onApproveExternalAgent={props.onApproveExternalAgent}
      registerUserMessageRow={props.registerUserMessageRow}
    />
  );
}

const ConversationMessageRow = memo(function ConversationMessageRow(props: {
  message: MobileConversationMessage;
  onApproveExternalAgent?: (input: ExternalApprovalActionInput) => Promise<boolean>;
  registerUserMessageRow: (messageId: string) => (node: HTMLDivElement | null) => void;
}) {
  if (props.message.role === "user") {
    return (
      <ChatBubble ref={props.registerUserMessageRow(props.message.id)} selectedSkill={props.message.selectedSkill}>
        {props.message.text}
      </ChatBubble>
    );
  }

  return (
    <AssistantReply
      reply={conversationMessageToAgentPayload(props.message)}
      status={assistantMessageStatus()}
      onApproveExternalAgent={props.onApproveExternalAgent}
    />
  );
}, areConversationMessageRowPropsEqual);

function areConversationMessageRowPropsEqual(
  previous: {
    message: MobileConversationMessage;
    onApproveExternalAgent?: (input: ExternalApprovalActionInput) => Promise<boolean>;
    registerUserMessageRow: (messageId: string) => (node: HTMLDivElement | null) => void;
  },
  next: {
    message: MobileConversationMessage;
    onApproveExternalAgent?: (input: ExternalApprovalActionInput) => Promise<boolean>;
    registerUserMessageRow: (messageId: string) => (node: HTMLDivElement | null) => void;
  },
): boolean {
  return previous.onApproveExternalAgent === next.onApproveExternalAgent
    && previous.registerUserMessageRow === next.registerUserMessageRow
    && areMobileConversationMessagesEqual(previous.message, next.message);
}

function AssistantReplyParts(props: {
  final: boolean;
  onApproveExternalAgent?: (input: ExternalApprovalActionInput) => Promise<boolean>;
  parts: MobileAssistantPart[];
  sessionId?: string;
}) {
  if (props.parts.length === 0) {
    return null;
  }

  return (
    <div className="assistant-reply-parts">
      {props.parts.map((part) => part.kind === "text" ? (
        <AssistantMarkdownContent
          key={part.id}
          content={part.text}
          final={props.final}
          showCopyButton={false}
        />
      ) : (
        <ToolCard
          key={part.id}
          sessionId={props.sessionId}
          tool={part.tool}
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
            <ToolStatusMark tone={viewModel.tone} />
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

function ToolStatusMark(props: { tone: MobileToolTone }) {
  if (props.tone === "running") {
    return (
      <svg
        className="tool-spinner"
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2.5"
        strokeLinecap="round"
        strokeLinejoin="round"
        aria-hidden={true}
      >
        <path d="M21 12a9 9 0 1 1-6.219-8.56" />
      </svg>
    );
  }

  if (props.tone === "success") {
    return <span className="tool-status-dot" aria-hidden={true} />;
  }

  if (props.tone === "error") {
    return <UiIcon name="x" />;
  }

  return null;
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
    <div ref={ref} className="message-row user-row" data-chat-feed-item="">
      <div className="user-bubble">
        {props.selectedSkill ? <div className="user-bubble-selected-skill">{props.selectedSkill.name}</div> : null}
        {hasText ? props.children : null}
      </div>
    </div>
  );
});

function AssistantPanel(props: { children: ReactNode; ariaLive?: "polite" }) {
  return (
    <div className="message-row assistant-row" data-chat-feed-item="">
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

function resolveAssistantParts(reply: AgentPayload | undefined): MobileAssistantPart[] {
  if (!reply) {
    return [];
  }
  if (reply.parts?.length) {
    return reply.parts;
  }

  const parts: MobileAssistantPart[] = [];
  if (reply.message) {
    parts.push({
      id: `${reply.session_id}:reply:text`,
      kind: "text",
      text: reply.message,
    });
  }
  for (const tool of reply.tools ?? []) {
    parts.push({
      id: tool.id,
      kind: "tool",
      tool,
    });
  }
  return parts;
}

function assistantMessageStatus(): StatusMessage {
  return { tone: "success", text: "回复已返回" };
}

function conversationMessageToAgentPayload(message: MobileConversationMessage): AgentPayload {
  return {
    message: message.text,
    parts: message.parts,
    session_ended: false,
    session_id: message.sessionId ?? "",
    thinking: message.thinking,
    tools: message.tools,
  };
}
