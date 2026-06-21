import { useEffect, useMemo, useRef, useState } from "react";
import {
  appendStoredMobileMessages,
  loadPersistedMobileConversations,
  loadStoredMobileConversations,
  savePersistedMobileConversations,
  saveStoredMobileConversations,
  upsertStoredMobileConversation,
} from "../lib/mobileSessionStorage";
import { hasTauriRuntime } from "../lib/bridgeBus";
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
  computerSessionSyncScope?: string;
  getFullSession: (sessionId: string) => Promise<SessionDetail>;
  getSession: (sessionId: string) => Promise<SessionDetail>;
  pinnedHistoryIds: string[];
  persistComputerSessionsEnabled: boolean;
  sendAgentMessage: (options: SendAgentMessageOptions) => Promise<SendAgentMessageResult>;
  sessions: SessionMetadata[];
  sessionsLoaded: boolean;
}

interface PersistConversationInput {
  createdAt?: string;
  id: string;
  messages: MobileConversationMessage[];
  preserveExistingTitle?: boolean;
  sourceMessageCount?: number;
  title: string;
  updatedAt?: string;
}

const HOME_IDLE_STATUS: StatusMessage = { tone: "idle", text: "首页" };
const PERSIST_DISABLED_STATUS: StatusMessage = { tone: "idle", text: "未开启" };

export function useMobileSessions(options: UseMobileSessionsOptions) {
  const [activeSessionId, setActiveSessionId] = useState<string>();
  const [homeMessages, setHomeMessages] = useState<MobileConversationMessage[]>([]);
  const [homeReply, setHomeReply] = useState<AgentPayload>();
  const [homeRun, setHomeRun] = useState<MobileSessionRunState>(() => createIdleRunState());
  const [sessionViews, setSessionViews] = useState<Record<string, MobileSessionView>>({});
  const [storedConversations, setStoredConversations] = useState<StoredMobileConversation[]>(() => initialStoredConversations());
  const [storageLoaded, setStorageLoaded] = useState(() => !hasTauriRuntime());
  const [computerSessionPersistStatus, setComputerSessionPersistStatus] =
    useState<StatusMessage>(PERSIST_DISABLED_STATUS);
  const activeSessionIdRef = useRef<string | undefined>(undefined);
  const storedConversationsRef = useRef<StoredMobileConversation[]>(storedConversations);
  const syncRunIdRef = useRef(0);

  useEffect(() => {
    activeSessionIdRef.current = activeSessionId;
  }, [activeSessionId]);

  useEffect(() => {
    storedConversationsRef.current = storedConversations;
  }, [storedConversations]);

  useEffect(() => {
    if (!hasTauriRuntime()) {
      return;
    }

    let cancelled = false;
    void loadPersistedMobileConversations()
      .then((conversations) => {
        if (cancelled) {
          return;
        }
        setStoredConversations(conversations);
        setStorageLoaded(true);
      })
      .catch((error: unknown) => {
        if (!cancelled) {
          setComputerSessionPersistStatus({ tone: "error", text: `同步失败：${errorMessage(error)}` });
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (!options.bridgeConnected || !options.sessionsLoaded || !storageLoaded) {
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
      saveConversationsInBackground(next);
      return next;
    });
    setSessionViews((current) => syncSessionViewTitles(current, options.sessions));
  }, [activeSessionId, options.bridgeConnected, options.sessions, options.sessionsLoaded, sessionViews, storageLoaded]);

  useEffect(() => {
    if (!options.persistComputerSessionsEnabled) {
      syncRunIdRef.current += 1;
      setComputerSessionPersistStatus(PERSIST_DISABLED_STATUS);
      return;
    }
    if (!options.bridgeConnected || !options.sessionsLoaded || !storageLoaded) {
      syncRunIdRef.current += 1;
      setComputerSessionPersistStatus({ tone: "idle", text: "等待连接" });
      return;
    }

    const runId = syncRunIdRef.current + 1;
    syncRunIdRef.current = runId;
    let cancelled = false;
    setComputerSessionPersistStatus({ tone: "loading", text: "同步中" });

    void syncComputerSessions(runId)
      .then((syncedCount) => {
        if (!cancelled && syncRunIdRef.current === runId) {
          setComputerSessionPersistStatus({ tone: "success", text: `已同步 ${syncedCount} 个` });
        }
      })
      .catch((error: unknown) => {
        if (!cancelled && syncRunIdRef.current === runId) {
          setComputerSessionPersistStatus({ tone: "error", text: `同步失败：${errorMessage(error)}` });
        }
      });

    return () => {
      cancelled = true;
      syncRunIdRef.current += 1;
    };
  }, [
    options.bridgeConnected,
    options.computerSessionSyncScope,
    options.getFullSession,
    options.persistComputerSessionsEnabled,
    options.sessions,
    options.sessionsLoaded,
    storageLoaded,
  ]);

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
        sourceMessageCount: detail.message_count,
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
      saveConversationsInBackground(next);
      return next;
    });
  }

  function saveConversationsInBackground(conversations: StoredMobileConversation[]): void {
    if (!hasTauriRuntime()) {
      saveStoredMobileConversations(conversations);
      return;
    }
    void savePersistedMobileConversations(conversations).catch((error: unknown) => {
      console.error("[useMobileSessions] save conversations failed", error);
    });
  }

  async function syncComputerSessions(runId: number): Promise<number> {
    const changed: StoredMobileConversation[] = [];
    const currentById = new Map(storedConversationsRef.current.map((conversation) => [conversation.id, conversation]));

    for (const session of options.sessions) {
      if (syncRunIdRef.current !== runId) {
        return 0;
      }
      const cached = currentById.get(session.id);
      if (cached && isStoredConversationCurrent(cached, session)) {
        continue;
      }

      const detail = await options.getFullSession(session.id);
      changed.push({
        created_at: detail.created_at,
        id: detail.id,
        messages: sessionDetailToConversationMessages(detail),
        source_message_count: detail.message_count,
        title: detail.title.trim() || sessionFallbackTitle(detail.id),
        updated_at: detail.updated_at,
      });
    }

    if (syncRunIdRef.current !== runId) {
      return 0;
    }

    const next = mergeSyncedConversations(storedConversationsRef.current, changed);
    await savePersistedMobileConversations(next);
    if (syncRunIdRef.current === runId) {
      setStoredConversations(next);
    }
    return options.sessions.length;
  }

  return {
    activeMessages,
    activeReply,
    activeSessionId,
    activeStatus,
    canSend: options.bridgeConnected && activeRun.status !== "running",
    clearCurrentConversation,
    computerSessionPersistStatus,
    hasConversation: activeMessages.length > 0 || Boolean(activeReply),
    historyItems,
    selectSession,
    sendMessage,
    startNewSession,
  };
}

function isStoredConversationCurrent(conversation: StoredMobileConversation, session: SessionMetadata): boolean {
  return conversation.updated_at === session.updated_at && conversation.source_message_count === session.message_count;
}

function mergeSyncedConversations(
  current: StoredMobileConversation[],
  synced: StoredMobileConversation[],
): StoredMobileConversation[] {
  if (synced.length === 0) {
    return current;
  }

  const nextById = new Map(current.map((conversation) => [conversation.id, conversation]));
  for (const conversation of synced) {
    nextById.set(conversation.id, conversation);
  }
  return [...nextById.values()].sort(compareStoredConversations);
}

function compareStoredConversations(left: StoredMobileConversation, right: StoredMobileConversation): number {
  const updatedOrder = right.updated_at.localeCompare(left.updated_at);
  if (updatedOrder !== 0) {
    return updatedOrder;
  }
  return left.title.localeCompare(right.title, "zh-Hans");
}

function initialStoredConversations(): StoredMobileConversation[] {
  return hasTauriRuntime() ? [] : loadStoredMobileConversations();
}
