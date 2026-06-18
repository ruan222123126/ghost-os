import type { ReactNode } from "react";
import type { AgentPayload, StatusMessage } from "../../mobileTypes";
import { CONVERSATION_PLACEHOLDER_SECTIONS, EMPTY_STATE_SUGGESTIONS } from "./data";
import { UiIcon } from "./icons";
import type { UiIconName } from "./types";

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
  sessionId: string;
}

export function AssistantReply(props: AssistantReplyProps) {
  if (!props.reply && props.status.tone !== "error") {
    return null;
  }

  return (
    <AssistantPanel ariaLive="polite">
      <div className="assistant-copy">
        <p className={props.status.tone === "error" ? "error-text" : undefined}>
          {props.reply?.message || props.status.text}
        </p>
        {props.reply?.session_id || props.sessionId ? (
          <p className="assistant-meta">Session：{props.reply?.session_id || props.sessionId}</p>
        ) : null}
      </div>
    </AssistantPanel>
  );
}

export function ConversationPlaceholder() {
  return (
    <AssistantPanel>
      <div className="assistant-copy conversation-placeholder">
        <p>
          下面是会话中页面的占位内容，用来检查顶部栏、三点菜单、左侧抽屉和首页输入框在长内容下的滚动表现。
        </p>
        {CONVERSATION_PLACEHOLDER_SECTIONS.map((section) => (
          <section key={section.title} className="conversation-placeholder-section">
            <h3>{section.title}</h3>
            <p>{section.body}</p>
            <ul>
              {section.bullets.map((bullet) => (
                <li key={bullet}>{bullet}</li>
              ))}
            </ul>
          </section>
        ))}
      </div>
    </AssistantPanel>
  );
}

export function ChatBubble(props: { children: ReactNode }) {
  return (
    <div className="message-row user-row">
      <div className="user-bubble">{props.children}</div>
    </div>
  );
}

function AssistantPanel(props: { children: ReactNode; ariaLive?: "polite" }) {
  return (
    <div className="message-row assistant-row">
      <section className="assistant-panel" aria-live={props.ariaLive}>
        {props.children}
      </section>
    </div>
  );
}
