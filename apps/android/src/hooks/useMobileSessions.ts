import { useEffect, useMemo, useRef, useState } from "react";
import {
  appendStoredMobileMessages,
  loadStoredMobileConversations,
  saveStoredMobileConversations,
  upsertStoredMobileConversation,
} from "../lib/mobileSessionStorage";
import {
  createAssistantConversationMessage,
  createClientRunId,
  createIdleRunState,
  createRunningRunState,
  createSuccessRunState,
  createUserConversationMessage,
  emptyStoredConversation,
  errorMessage,
  findStoredTitle,
  mergeHistoryItems,
  normalizeConversationSessionIds,
  reconcileStoredConversationsWithBridge,
  runStateToStatus,
  sessionDetailToConversationMessages,
  sessionFallbackTitle,
  statusToRunState,
  syncSessionViewTitles,
  upsertSessionView,
} from "../lib/mobileSessionProjection";
import type {
  AgentPayload,
  MobileConversationMessage,
  MobileSessionRunState,
  MobileSessionView,
  SessionDetail,
  SessionMetadata,
  StatusMessage,
  StoredMobileConversation,
} from "../mobileTypes";

export { mergeHistoryItems, reconcileStoredConversationsWithBridge } from "../lib/mobileSessionProjection";

interface SendAgentMessageOptions {
  message: string;
  onReply: (reply: AgentPayload) => void;
  onSessionId: (sessionId: string) => void;
  onStatus: (status: StatusMessage) => void;
  sessionId?: string;
}

interface SendAgentMessageResult {
  ok: boolean;
  reply?: AgentPayload;
  sessionId?: string;
}

interface UseMobileSessionsOptions {
  bridgeConnected: boolean;
  getSession: (sessionId: string) => Promise<SessionDetail>;
  pinnedHistoryIds: string[];
  sendAgentMessage: (options: SendAgentMessageOptions) => Promise<SendAgentMessageResult>;
  sessions: SessionMetadata[];
  sessionsLoaded: boolean;
}

interface PersistConversationInput {
  createdAt?: string;
  id: string;
  messages: MobileConversationMessage[];
  preserveExistingTitle?: boolean;
  title: string;
  updatedAt?: string;
}

const HOME_IDLE_STATUS: StatusMessage = { tone: "idle", text: "首页" };

export function useMobileSessions(options: UseMobileSessionsOptions) {
  const [activeSessionId, setActiveSessionId] = useState<string>();
  const [homeMessages, setHomeMessages] = useState<MobileConversationMessage[]>([]);
  const [homeReply, setHomeReply] = useState<AgentPayload>();
  const [homeRun, setHomeRun] = useState<MobileSessionRunState>(() => createIdleRunState());
  const [sessionViews, setSessionViews] = useState<Record<string, MobileSessionView>>({});
  const [storedConversations, setStoredConversations] = useState<StoredMobileConversation[]>(() =>
    loadStoredMobileConversations(),
  );
  const activeSessionIdRef = useRef<string | undefined>(undefined);

  useEffect(() => {
    activeSessionIdRef.current = activeSessionId;
  }, [activeSessionId]);

  useEffect(() => {
    if (!options.bridgeConnected || !options.sessionsLoaded) {
      return;
    }

    const keepIds = new Set(
      Object.values(sessionViews)
        .filter((view) => view.run.status === "running")
        .map((view) => view.id),
    );
    if (activeSessionId) {
      keepIds.add(activeSessionId);
    }

    setStoredConversations((current) => {
      const next = reconcileStoredConversationsWithBridge(current, options.sessions, keepIds);
      saveStoredMobileConversations(next);
      return next;
    });
    setSessionViews((current) => syncSessionViewTitles(current, options.sessions));
  }, [activeSessionId, options.bridgeConnected, options.sessions, options.sessionsLoaded, sessionViews]);

  const activeView = activeSessionId ? sessionViews[activeSessionId] : undefined;
  const activeMessages = activeView?.messages ?? homeMessages;
  const activeReply = activeView?.reply ?? homeReply;
  const activeRun = activeView?.run ?? homeRun;
  const activeStatus = runStateToStatus(activeRun, HOME_IDLE_STATUS);
  const historyItems = useMemo(
    () =>
      mergeHistoryItems({
        bridgeConnected: options.bridgeConnected,
        pinned: new Set(options.pinnedHistoryIds),
        sessionViews,
        sessions: options.sessions,
        sessionsLoaded: options.sessionsLoaded,
        storedConversations,
      }),
    [
      options.bridgeConnected,
      options.pinnedHistoryIds,
      options.sessions,
      options.sessionsLoaded,
      sessionViews,
      storedConversations,
    ],
  );

  async function sendMessage(text: string): Promise<boolean> {
    const trimmed = text.trim();
    if (!trimmed || !options.bridgeConnected || activeRun.status === "running") {
      return false;
    }

    const initialSessionId = activeSessionId?.trim() || "";
    let targetSessionId = initialSessionId;
    const requestId = createClientRunId("request");
    const traceId = createClientRunId("trace");
    const userMessage = createUserConversationMessage(trimmed, initialSessionId);
    const optimisticMessages = [...activeMessages, userMessage];

    if (initialSessionId) {
      setSessionViews((current) =>
        upsertSessionView(current, initialSessionId, {
          messages: optimisticMessages,
          reply: undefined,
          run: createRunningRunState(requestId, traceId, "发送中"),
          title: current[initialSessionId]?.title || findStoredTitle(storedConversations, initialSessionId) || trimmed,
          unread: false,
          bridgeOwned: true,
        }),
      );
    } else {
      setHomeMessages(optimisticMessages);
      setHomeReply(undefined);
      setHomeRun(createRunningRunState(requestId, traceId, "发送中"));
    }

    const result = await options.sendAgentMessage({
      message: trimmed,
      onReply: (reply) => {
        applyReply(targetSessionId, reply);
      },
      onSessionId: (sessionId) => {
        const resolvedSessionId = sessionId.trim();
        if (!resolvedSessionId || targetSessionId === resolvedSessionId) {
          return;
        }
        targetSessionId = resolvedSessionId;
        activateCreatedSession(resolvedSessionId, trimmed, optimisticMessages);
      },
      onStatus: (status) => {
        applyRunStatus(targetSessionId, status, requestId, traceId);
      },
      sessionId: initialSessionId || undefined,
    });

    if (!result.ok) {
      return false;
    }

    const resolvedSessionId = result.sessionId?.trim() || targetSessionId;
    if (!resolvedSessionId) {
      return true;
    }

    if (!targetSessionId) {
      targetSessionId = resolvedSessionId;
      activateCreatedSession(resolvedSessionId, trimmed, optimisticMessages);
    }
    commitFinalReply(resolvedSessionId, result.reply, trimmed);
    return true;
  }

  async function selectSession(sessionId: string): Promise<void> {
    const trimmedSessionId = sessionId.trim();
    if (!trimmedSessionId) {
      return;
    }

    const stored = storedConversations.find((conversation) => conversation.id === trimmedSessionId);
    const existing = sessionViews[trimmedSessionId];
    activeSessionIdRef.current = trimmedSessionId;
    setActiveSessionId(trimmedSessionId);
    setHomeReply(undefined);
    setHomeRun(createIdleRunState());
    setSessionViews((current) =>
      upsertSessionView(current, trimmedSessionId, {
        messages: existing?.messages ?? stored?.messages ?? [],
        run: existing?.run ?? createIdleRunState(stored ? "历史会话已加载" : "正在加载历史会话"),
        title: existing?.title || stored?.title || sessionFallbackTitle(trimmedSessionId),
        unread: false,
        bridgeOwned: existing?.bridgeOwned ?? false,
      }),
    );

    if (!options.bridgeConnected) {
      if (!stored && !existing) {
        applyRunStatus(trimmedSessionId, { tone: "error", text: "需要先连接电脑端才能加载该历史会话" });
      }
      return;
    }
    if (existing?.run.status === "running") {
      return;
    }

    applyRunStatus(trimmedSessionId, { tone: "loading", text: "正在加载历史会话" });
    try {
      const detail = await options.getSession(trimmedSessionId);
      const messages = sessionDetailToConversationMessages(detail);
      const title = detail.title.trim() || stored?.title || sessionFallbackTitle(detail.id);
      setSessionViews((current) =>
        upsertSessionView(current, detail.id, {
          messages,
          run: createSuccessRunState("历史会话已加载"),
          title,
          unread: false,
          bridgeOwned: true,
          updatedAt: detail.updated_at,
        }),
      );
      persistConversation({
        createdAt: detail.created_at,
        id: detail.id,
        messages,
        title,
        updatedAt: detail.updated_at,
      });
    } catch (error) {
      applyRunStatus(trimmedSessionId, { tone: "error", text: errorMessage(error) });
    }
  }

  function startNewSession(): void {
    activeSessionIdRef.current = undefined;
    setActiveSessionId(undefined);
    setHomeMessages([]);
    setHomeReply(undefined);
    setHomeRun(createIdleRunState("新会话"));
  }

  function clearCurrentConversation(): void {
    activeSessionIdRef.current = undefined;
    setActiveSessionId(undefined);
    setHomeMessages([]);
    setHomeReply(undefined);
    setHomeRun(createIdleRunState("本地消息已清空"));
  }

  function applyReply(sessionId: string, reply: AgentPayload): void {
    if (!sessionId) {
      setHomeReply(reply);
      return;
    }

    setSessionViews((current) =>
      upsertSessionView(current, sessionId, {
        reply,
        run: current[sessionId]?.run ?? createRunningRunState(),
        unread: activeSessionIdRef.current !== sessionId,
        updatedAt: new Date().toISOString(),
      }),
    );
  }

  function applyRunStatus(sessionId: string, status: StatusMessage, requestId?: string, traceId?: string): void {
    const run = statusToRunState(status, requestId, traceId);
    if (!sessionId) {
      setHomeRun(run);
      return;
    }

    setSessionViews((current) =>
      upsertSessionView(current, sessionId, {
        run,
        unread: activeSessionIdRef.current !== sessionId && status.tone !== "loading",
        updatedAt: new Date().toISOString(),
      }),
    );
  }

  function activateCreatedSession(
    sessionId: string,
    title: string,
    messages: MobileConversationMessage[],
  ): void {
    const normalizedMessages = normalizeConversationSessionIds(messages, sessionId);
    activeSessionIdRef.current = sessionId;
    setActiveSessionId(sessionId);
    setHomeMessages([]);
    setHomeReply(undefined);
    setHomeRun(createIdleRunState());
    setSessionViews((current) =>
      upsertSessionView(current, sessionId, {
        messages: normalizedMessages,
        reply: homeReply,
        run: homeRun.status === "running" ? homeRun : createRunningRunState(),
        title,
        unread: false,
        bridgeOwned: true,
      }),
    );
    persistConversation({
      id: sessionId,
      messages: normalizedMessages,
      preserveExistingTitle: true,
      title,
    });
  }

  function commitFinalReply(sessionId: string, reply: AgentPayload | undefined, title: string): void {
    setSessionViews((current) => {
      const existing = current[sessionId];
      const baseMessages = normalizeConversationSessionIds(existing?.messages ?? activeMessages, sessionId);
      const assistantMessage = reply ? createAssistantConversationMessage(reply, sessionId) : undefined;
      const nextMessages = assistantMessage
        ? appendStoredMobileMessages({ ...emptyStoredConversation(sessionId), messages: baseMessages }, [
            ...baseMessages,
            assistantMessage,
          ])
        : baseMessages;
      const next = upsertSessionView(current, sessionId, {
        messages: nextMessages,
        reply: undefined,
        run: createSuccessRunState(reply?.session_ended ? "会话已结束" : "回复已返回", reply?.session_ended),
        title: existing?.title || findStoredTitle(storedConversations, sessionId) || title,
        unread: activeSessionIdRef.current !== sessionId,
        bridgeOwned: true,
      });
      persistConversation({
        id: sessionId,
        messages: nextMessages,
        preserveExistingTitle: true,
        title,
      });
      return next;
    });
  }

  function persistConversation(input: PersistConversationInput): void {
    setStoredConversations((current) => {
      const next = upsertStoredMobileConversation(current, input);
      saveStoredMobileConversations(next);
      return next;
    });
  }

  return {
    activeMessages,
    activeReply,
    activeSessionId,
    activeStatus,
    canSend: options.bridgeConnected && activeRun.status !== "running",
    clearCurrentConversation,
    hasConversation: activeMessages.length > 0 || Boolean(activeReply),
    historyItems,
    selectSession,
    sendMessage,
    startNewSession,
  };
}
