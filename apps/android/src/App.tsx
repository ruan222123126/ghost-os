import { useEffect, useMemo, useState } from "react";
import type { FormEvent } from "react";
import {
  AssistantIntro,
  AssistantReply,
  ChatBubble,
  ChatComposer,
  ChatHeader,
  MobileSidebar,
  MoreActionSheet,
  ScrollDownButton,
} from "./components/MobileChatHome";
import { MobileSettingsPanel } from "./components/MobileSettingsPanel";
import type { SidebarHistoryItem } from "./components/MobileChatHome";
import { useBodyScrollLock } from "./hooks/useBodyScrollLock";
import { useChatFeedScroll } from "./hooks/useChatFeedScroll";
import { useMobileBridge } from "./hooks/useMobileBridge";
import {
  appendStoredMobileMessages,
  loadStoredMobileConversations,
  saveStoredMobileConversations,
  upsertStoredMobileConversation,
} from "./lib/mobileSessionStorage";
import type {
  AgentPayload,
  MobileConversationMessage,
  SessionDetail,
  SessionMessage,
  SessionMetadata,
  StatusMessage,
  StoredMobileConversation,
} from "./mobileTypes";
import "./App.css";
import "./components/mobileChat/Messages.css";
import "./App.overlays.css";

function isNonEmptyMessage(value: string): boolean {
  return value.trim().length > 0;
}

function displayRuntime(config: ReturnType<typeof useMobileBridge>["config"]): string {
  if (config?.provider && config.model) {
    return `${config.provider} / ${config.model}`;
  }
  return config?.provider || config?.model || "Bridge Runtime";
}

function sessionFallbackTitle(sessionId: string): string {
  const shortId = sessionId.trim().slice(0, 8);
  return shortId ? `会话 ${shortId}` : "新会话";
}

function assistantMessageStatus(): StatusMessage {
  return { tone: "success", text: "回复已返回" };
}

function App() {
  const {
    bridgeUrl,
    config,
    connectBridge,
    connectionStatus,
    getSession,
    host,
    providerList,
    reply,
    sendAgentMessage,
    sessions,
    setReply,
    setSettings,
    setStatus,
    settings,
    switchModel,
    status,
  } = useMobileBridge();
  const [message, setMessage] = useState("");
  const [conversationMessages, setConversationMessages] = useState<MobileConversationMessage[]>([]);
  const [storedConversations, setStoredConversations] = useState<StoredMobileConversation[]>(() =>
    loadStoredMobileConversations(),
  );
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);
  const [isRuntimeMenuOpen, setIsRuntimeMenuOpen] = useState(false);
  const [isMoreMenuOpen, setIsMoreMenuOpen] = useState(false);
  const [pinnedHistoryIds, setPinnedHistoryIds] = useState<string[]>([]);
  const [activeHistoryId, setActiveHistoryId] = useState<string | undefined>();

  const canSend = isNonEmptyMessage(message) && status.tone !== "loading";
  const runtimeLabel = useMemo(() => displayRuntime(config), [config]);
  const isModalOpen = isSidebarOpen || isSettingsOpen || isMoreMenuOpen;
  const hasLocalConversation = conversationMessages.length > 0 || Boolean(reply);
  const conversationScrollKey = useMemo(
    () => conversationMessages.map((item) => `${item.id}:${item.text.length}`).join("|"),
    [conversationMessages],
  );
  const { handleScroll, resetScrollDown, scrollRef, scrollToBottom, showScrollDown } = useChatFeedScroll(
    conversationScrollKey,
    reply,
  );
  const historyItems = useMemo<SidebarHistoryItem[]>(() => {
    const pinned = new Set(pinnedHistoryIds);
    return mergeHistoryItems(storedConversations, sessions, pinned);
  }, [pinnedHistoryIds, sessions, storedConversations]);
  const activeHistoryItem = useMemo(
    () => historyItems.find((item) => item.id === activeHistoryId),
    [activeHistoryId, historyItems],
  );

  useBodyScrollLock(isModalOpen);

  useEffect(() => {
    const sessionId = settings.sessionId.trim();
    if (sessionId) {
      setActiveHistoryId(sessionId);
      const stored = storedConversations.find((conversation) => conversation.id === sessionId);
      if (stored && conversationMessages.length === 0 && !reply) {
        setConversationMessages(stored.messages);
      }
    }
  }, [conversationMessages.length, reply, settings.sessionId, storedConversations]);

  async function sendMessage(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    const trimmed = message.trim();
    if (!trimmed) {
      return;
    }

    const initialSessionId = settings.sessionId.trim();
    const activeStoredConversation = initialSessionId
      ? storedConversations.find((conversation) => conversation.id === initialSessionId)
      : undefined;
    const userMessage = createUserConversationMessage(trimmed, initialSessionId);
    const optimisticMessages = [...conversationMessages, userMessage];
    setConversationMessages(optimisticMessages);
    setReply(undefined);

    const result = await sendAgentMessage({ message: trimmed });
    if (result.ok) {
      setMessage("");
    }
    const resolvedSessionId = result.sessionId?.trim();
    if (!result.ok || !resolvedSessionId) {
      return;
    }

    const assistantMessage = result.reply
      ? createAssistantConversationMessage(result.reply, resolvedSessionId)
      : undefined;
    const nextMessages = normalizeConversationSessionIds(
      assistantMessage ? [...optimisticMessages, assistantMessage] : optimisticMessages,
      resolvedSessionId,
    );
    setConversationMessages(nextMessages);
    setReply(undefined);
    setActiveHistoryId(resolvedSessionId);
    persistConversation({
      id: resolvedSessionId,
      messages: appendStoredMobileMessages(activeStoredConversation, nextMessages),
      title: activeStoredConversation?.title || trimmed,
    });
  }

  async function selectHistory(sessionId: string): Promise<void> {
    const trimmedSessionId = sessionId.trim();
    if (!trimmedSessionId) {
      return;
    }

    const stored = storedConversations.find((conversation) => conversation.id === trimmedSessionId);
    setActiveHistoryId(trimmedSessionId);
    setSettings((current) => ({ ...current, sessionId: trimmedSessionId }));
    setReply(undefined);
    setMessage("");
    setConversationMessages(stored?.messages ?? []);
    setIsSidebarOpen(false);
    setStatus({
      tone: stored ? "success" : "loading",
      text: stored ? "历史会话已加载" : "正在加载历史会话",
    });

    if (!config) {
      if (!stored) {
        setStatus({ tone: "error", text: "需要先连接电脑端才能加载该历史会话" });
      }
      return;
    }

    try {
      const detail = await getSession(trimmedSessionId);
      const messages = sessionDetailToConversationMessages(detail);
      setConversationMessages(messages);
      persistConversation({
        createdAt: detail.created_at,
        id: detail.id,
        messages,
        title: detail.title.trim() || stored?.title || sessionFallbackTitle(detail.id),
        updatedAt: detail.updated_at,
      });
      setStatus({ tone: "success", text: "历史会话已加载" });
    } catch (error) {
      const text = error instanceof Error ? error.message : String(error);
      setStatus({ tone: "error", text });
    }
  }

  function openSidebar(): void {
    setIsRuntimeMenuOpen(false);
    setIsMoreMenuOpen(false);
    setIsSettingsOpen(false);
    setIsSidebarOpen(true);
  }

  function openSettings(): void {
    setIsRuntimeMenuOpen(false);
    setIsMoreMenuOpen(false);
    setIsSidebarOpen(false);
    setIsSettingsOpen(true);
  }

  function startNewSession(): void {
    setReply(undefined);
    setConversationMessages([]);
    setMessage("");
    setActiveHistoryId(undefined);
    setSettings((current) => ({ ...current, sessionId: "" }));
    setStatus({ tone: "idle", text: "新会话" });
    setIsRuntimeMenuOpen(false);
    setIsMoreMenuOpen(false);
    setIsSidebarOpen(false);
    resetScrollDown();
  }

  function clearLocalConversation(): void {
    setReply(undefined);
    setConversationMessages([]);
    setActiveHistoryId(undefined);
    setStatus({ tone: "idle", text: "本地消息已清空" });
    setIsMoreMenuOpen(false);
    resetScrollDown();
  }

  function toggleActiveHistoryPin(): void {
    if (!activeHistoryItem) {
      return;
    }

    setPinnedHistoryIds((current) =>
      current.includes(activeHistoryItem.id)
        ? current.filter((id) => id !== activeHistoryItem.id)
        : [...current, activeHistoryItem.id],
    );
    setIsMoreMenuOpen(false);
  }

  function persistConversation(input: {
    createdAt?: string;
    id: string;
    messages: MobileConversationMessage[];
    title: string;
    updatedAt?: string;
  }): void {
    setStoredConversations((current) => {
      const next = upsertStoredMobileConversation(current, {
        createdAt: input.createdAt,
        id: input.id,
        messages: input.messages,
        title: input.title,
        updatedAt: input.updatedAt,
      });
      saveStoredMobileConversations(next);
      return next;
    });
  }

  return (
    <div className="mobile-chat-shell">
      <MobileSidebar
        open={isSidebarOpen}
        host={host}
        config={config}
        settings={settings}
        historyItems={historyItems}
        activeHistoryId={activeHistoryId}
        onClose={() => setIsSidebarOpen(false)}
        onNewSession={startNewSession}
        onSelectHistory={(sessionId) => void selectHistory(sessionId)}
        onConnect={connectBridge}
        onOpenSettings={openSettings}
      />

      <div className="mobile-chat-content" aria-hidden={isModalOpen} inert={isModalOpen ? true : undefined}>
        <ChatHeader
          runtimeLabel={runtimeLabel}
          config={config}
          providerList={providerList}
          status={status}
          bridgeUrl={bridgeUrl}
          hasConversation={hasLocalConversation}
          runtimeMenuOpen={isRuntimeMenuOpen}
          onOpenSidebar={openSidebar}
          onToggleRuntimeMenu={() => setIsRuntimeMenuOpen((current) => !current)}
          onCloseRuntimeMenu={() => setIsRuntimeMenuOpen(false)}
          onSwitchModel={switchModel}
          onOpenSettings={openSettings}
          onOpenMoreMenu={() => {
            setIsRuntimeMenuOpen(false);
            setIsMoreMenuOpen(true);
          }}
          onNewSession={startNewSession}
        />

        <main
          ref={scrollRef}
          onScroll={handleScroll}
          className={`chat-feed ${hasLocalConversation ? "" : "is-empty"}`}
        >
          {!hasLocalConversation ? (
            <AssistantIntro onSelectSuggestion={setMessage} />
          ) : null}

          {conversationMessages.map((item) =>
            item.role === "user" ? (
              <ChatBubble key={item.id}>{item.text}</ChatBubble>
            ) : (
              <AssistantReply
                key={item.id}
                reply={conversationMessageToAgentPayload(item)}
                status={assistantMessageStatus()}
                sessionId={item.sessionId || settings.sessionId}
              />
            ),
          )}
          <AssistantReply reply={reply} status={status} sessionId={settings.sessionId} />
        </main>

        {showScrollDown ? <ScrollDownButton onClick={() => scrollToBottom()} /> : null}

        <ChatComposer
          value={message}
          disabled={!canSend}
          loading={status.tone === "loading"}
          onSubmit={sendMessage}
          onChange={setMessage}
          onOpenSettings={openSettings}
        />
      </div>

      <MobileSettingsPanel
        open={isSettingsOpen}
        settings={settings}
        connectionStatus={connectionStatus}
        onClose={() => setIsSettingsOpen(false)}
        onConnect={connectBridge}
        onSettingsChange={setSettings}
      />

      <MoreActionSheet
        open={isMoreMenuOpen}
        hasLocalConversation={hasLocalConversation}
        canTogglePin={Boolean(activeHistoryItem)}
        isPinned={activeHistoryItem?.pinned ?? false}
        onClose={() => setIsMoreMenuOpen(false)}
        onTogglePin={toggleActiveHistoryPin}
        onClearConversation={clearLocalConversation}
      />
    </div>
  );
}

function mergeHistoryItems(
  storedConversations: StoredMobileConversation[],
  sessions: SessionMetadata[],
  pinned: Set<string>,
): SidebarHistoryItem[] {
  const items = new Map<string, SidebarHistoryItem>();
  for (const conversation of storedConversations) {
    items.set(conversation.id, {
      id: conversation.id,
      pinned: pinned.has(conversation.id),
      title: conversation.title.trim() || sessionFallbackTitle(conversation.id),
      updatedAt: conversation.updated_at,
    });
  }
  for (const session of sessions) {
    if (items.has(session.id)) {
      continue;
    }
    items.set(session.id, {
      id: session.id,
      pinned: pinned.has(session.id),
      title: session.title.trim() || sessionFallbackTitle(session.id),
      updatedAt: session.updated_at,
    });
  }
  return [...items.values()];
}

function createUserConversationMessage(text: string, sessionId: string): MobileConversationMessage {
  return {
    id: `${sessionId || "pending"}:user:${Date.now()}`,
    role: "user",
    sessionId: sessionId || undefined,
    text,
  };
}

function createAssistantConversationMessage(reply: AgentPayload, sessionId: string): MobileConversationMessage {
  return {
    id: `${sessionId}:assistant:${Date.now()}`,
    role: "assistant",
    sessionId,
    text: reply.message,
    thinking: reply.thinking,
  };
}

function normalizeConversationSessionIds(
  messages: MobileConversationMessage[],
  sessionId: string,
): MobileConversationMessage[] {
  return messages.map((message) => ({
    ...message,
    sessionId,
  }));
}

function sessionDetailToConversationMessages(detail: SessionDetail): MobileConversationMessage[] {
  return detail.messages
    .filter((message) => message.role === "user" || message.role === "assistant")
    .map((message) => sessionMessageToConversationMessage(message, detail.id))
    .filter(isConversationMessage);
}

function sessionMessageToConversationMessage(
  message: SessionMessage,
  sessionId: string,
): MobileConversationMessage | null {
  if (message.role !== "user" && message.role !== "assistant") {
    return null;
  }
  const text = sessionMessageText(message);
  if (!text.trim() && !message.thinking?.trim()) {
    return null;
  }
  return {
    id: `${sessionId}:${message.index}:${message.role}`,
    role: message.role,
    sessionId,
    text,
    thinking: message.thinking,
  };
}

function sessionMessageText(message: SessionMessage): string {
  if (message.text !== undefined) {
    return message.text;
  }
  return (message.content ?? [])
    .map((part) => part.text ?? "")
    .filter(Boolean)
    .join("\n");
}

function conversationMessageToAgentPayload(message: MobileConversationMessage): AgentPayload {
  return {
    message: message.text,
    session_ended: false,
    session_id: message.sessionId ?? "",
    thinking: message.thinking,
  };
}

function isConversationMessage(
  message: MobileConversationMessage | null,
): message is MobileConversationMessage {
  return message !== null;
}

export default App;
