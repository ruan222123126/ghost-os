import type { ReactNode } from "react";
import type { AgentPayload, StatusMessage } from "../../mobileTypes";
import { EMPTY_STATE_SUGGESTIONS } from "./data";
import { AssistantMarkdownContent } from "./AssistantMarkdownContent";
import { UiIcon } from "./icons";
import { MessageCopyButton } from "./MessageCopyButton";
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

  const replyMessage = props.reply?.message.trim() ?? "";
  const displaySessionId = props.reply?.session_id || props.sessionId;

  return (
    <AssistantPanel ariaLive="polite">
      <div className="assistant-copy assistant-reply">
        {props.status.tone === "error" ? (
          <p className="error-text">{replyMessage || props.status.text}</p>
        ) : (
          <>
            {replyMessage ? (
              <div className="assistant-reply-actions">
                <MessageCopyButton text={replyMessage} />
              </div>
            ) : null}
            {replyMessage ? <AssistantMarkdownContent content={replyMessage} /> : <p>{props.status.text}</p>}
          </>
        )}
        {displaySessionId ? <p className="assistant-meta">Session：{displaySessionId}</p> : null}
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
