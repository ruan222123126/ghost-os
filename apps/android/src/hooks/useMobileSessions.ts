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
  MOBILE_PERSISTED_SESSION_PAGE_LIMIT,
  recentMobileBridgeSessions,
  trimMobileSessionViews,
  trimStoredMobileConversations,
} from "../lib/mobileSessionLimits";
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
  isActiveSessionTurnDraft,
  isAgentRunCancellationMessage,
  mergeHistoryItems,
  normalizeConversationSessionIds,
  reconcileStoredConversationsWithBridge,
  runStateToStatus,
  sessionTurnDraftToAgentPayload,
  sessionTurnDraftToRunState,
  sessionDetailToConversationMessages,
  sessionFallbackTitle,
  statusToRunState,
  syncSessionViewTitles,
  upsertSessionView,
} from "../lib/mobileSessionProjection";
import type {
  AgentPayload,
  ChatSelectedSkill,
  MobileConversationMessage,
  MobileSessionRunState,
  MobileSessionView,
  SessionDetail,
  SessionGetOptions,
  SessionMetadata,
  SessionRuntimeSelection,
  StatusMessage,
  StoredMobileConversation,
} from "../mobileTypes";

export { mergeHistoryItems, reconcileStoredConversationsWithBridge } from "../lib/mobileSessionProjection";

interface SendAgentMessageOptions {
  history: MobileConversationMessage[];
  message: string;
  onReply: (reply: AgentPayload) => void;
  onSessionId: (sessionId: string) => void;
  onStatus: (status: StatusMessage) => void;
  requestId?: string;
  selectedSkill?: ChatSelectedSkill;
  sessionId?: string;
  shouldStreamRealtime?: (sessionId: string) => boolean;
  traceId?: string;
}

interface SendAgentMessageResult {
  mode?: "local" | "remote";
  ok: boolean;
  reply?: AgentPayload;
  sessionId?: string;
}

interface StopAgentRunInput {
  sessionId?: string;
  traceId?: string;
}

interface StopAgentRunResult {
  ok: boolean;
  sessionId?: string;
  status?: "stopped" | "not_running";
}

interface AppendSessionMessagesInput {
  expectedHead?: number;
  messages: MobileConversationMessage[];
  sessionId: string;
  title: string;
}

interface AppendSessionMessagesResult {
  messageCount: number;
  status: "appended" | "conflict";
  updatedAt: string;
}

interface UseMobileSessionsOptions {
  appendSessionMessages?: (input: AppendSessionMessagesInput) => Promise<AppendSessionMessagesResult>;
  bridgeConnected: boolean;
  computerSessionSyncScope?: string;
  getFullSession: (sessionId: string) => Promise<SessionDetail>;
  getSession: (sessionId: string, options?: SessionGetOptions) => Promise<SessionDetail>;
  onSessionRuntimeSelection?: (selection: SessionRuntimeSelection | null | undefined) => Promise<void> | void;
  pinnedHistoryIds: string[];
  persistComputerSessionsEnabled: boolean;
  sendAvailable?: boolean;
  sendAgentMessage: (options: SendAgentMessageOptions) => Promise<SendAgentMessageResult>;
  sessions: SessionMetadata[];
  sessionsLoaded: boolean;
  stopAgentRun: (input: StopAgentRunInput) => Promise<StopAgentRunResult>;
}

interface PersistConversationInput {
  createdAt?: string;
  bridgeMessageCount?: number;
  id: string;
  messages: MobileConversationMessage[];
  preserveExistingTitle?: boolean;
  sourceMessageCount?: number;
  syncedMessageCount?: number;
  title: string;
  updatedAt?: string;
}

interface SyncComputerSessionsResult {
  syncedCount: number;
  totalCount: number;
}

interface PostSendFocusRequest {
  messageId: string;
  token: number;
}

const HOME_IDLE_STATUS: StatusMessage = { tone: "idle", text: "首页" };
const PERSIST_DISABLED_STATUS: StatusMessage = { tone: "idle", text: "未开启" };
const EXTERNAL_RUNNING_SESSION_POLL_INTERVAL_MS = 1500;
const SESSION_MESSAGES_LOADING_STATUS_TEXT = "正在加载历史会话";
const SESSION_MESSAGES_LOADED_STATUS_TEXT = "历史会话已加载";

export function useMobileSessions(options: UseMobileSessionsOptions) {
  const [activeSessionId, setActiveSessionId] = useState<string>();
  const [homeMessages, setHomeMessages] = useState<MobileConversationMessage[]>([]);
  const [homeReply, setHomeReply] = useState<AgentPayload>();
  const [homeRun, setHomeRun] = useState<MobileSessionRunState>(() => createIdleRunState());
  const [sessionViews, setSessionViews] = useState<Record<string, MobileSessionView>>({});
  const [storedConversations, setStoredConversations] = useState<StoredMobileConversation[]>(() => initialStoredConversations());
  const [storageLoaded, setStorageLoaded] = useState(() => !hasTauriRuntime());
  const [postSendFocusRequest, setPostSendFocusRequest] = useState<PostSendFocusRequest | null>(null);
  const [computerSessionPersistStatus, setComputerSessionPersistStatus] =
    useState<StatusMessage>(PERSIST_DISABLED_STATUS);
  const activeSessionIdRef = useRef<string | undefined>(undefined);
  const postSendFocusTokenRef = useRef(0);
  const stoppingRunKeysRef = useRef<Set<string>>(new Set());
  const sessionViewsRef = useRef<Record<string, MobileSessionView>>(sessionViews);
  const storedConversationsRef = useRef<StoredMobileConversation[]>(storedConversations);
  const syncRunIdRef = useRef(0);

  useEffect(() => {
    activeSessionIdRef.current = activeSessionId;
  }, [activeSessionId]);

  useEffect(() => {
    storedConversationsRef.current = storedConversations;
  }, [storedConversations]);

  useEffect(() => {
    sessionViewsRef.current = sessionViews;
  }, [sessionViews]);

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
        const next = trimStoredConversationsForState(conversations);
        setStoredConversations(next);
        if (next.length !== conversations.length) {
          saveConversationsInBackground(next);
        }
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

    const keepIds = new Set<string>();
    if (activeSessionId) {
      keepIds.add(activeSessionId);
    }

    setStoredConversations((current) => {
      const next = trimStoredConversationsForState(
        reconcileStoredConversationsWithBridge(current, options.sessions, keepIds),
      );
      saveConversationsInBackground(next);
      return next;
    });
    setSessionViews((current) => trimSessionViewsForState(syncSessionViewTitles(current, options.sessions)));
  }, [
    activeSessionId,
    options.bridgeConnected,
    options.pinnedHistoryIds,
    options.sessions,
    options.sessionsLoaded,
    sessionViews,
    storageLoaded,
  ]);

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
      .then((syncResult) => {
        if (!cancelled && syncRunIdRef.current === runId) {
          setComputerSessionPersistStatus({ tone: "success", text: syncComputerSessionsStatusText(syncResult) });
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
    options.getSession,
    options.pinnedHistoryIds,
    options.persistComputerSessionsEnabled,
    options.sessions,
    options.sessionsLoaded,
    storageLoaded,
  ]);

  const activeView = activeSessionId ? sessionViews[activeSessionId] : undefined;
  const activeMessages = activeView?.messages ?? homeMessages;
  const activeReply = activeView?.reply ?? homeReply;
  const activeRun = activeView?.run ?? homeRun;
  const hasOlderHistory = Boolean(activeView?.hasOlderHistory);
  const loadingOlderHistory = Boolean(activeView?.loadingOlderHistory);
  const loadingSessionMessages = Boolean(
    activeSessionId
      && activeView?.run.status === "running"
      && activeView.run.statusText === SESSION_MESSAGES_LOADING_STATUS_TEXT,
  );
  const activeStatus = runStateToStatus(activeRun, HOME_IDLE_STATUS);
  const sendAvailable = options.sendAvailable ?? options.bridgeConnected;
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
  const activeExternalRunningSessionId = useMemo(() => {
    if (!activeSessionId) {
      return "";
    }
    const view = sessionViews[activeSessionId];
    return view && shouldPollExternalRunningSession(view) ? view.id : "";
  }, [activeSessionId, sessionViews]);
  const activeSessionBridgeVersion = useMemo(() => {
    if (!activeSessionId) {
      return "";
    }
    const metadata = options.sessions.find((session) => session.id === activeSessionId);
    return metadata ? `${metadata.id}:${metadata.updated_at}:${metadata.message_count}` : "";
  }, [activeSessionId, options.sessions]);

  useEffect(() => {
    if (!options.bridgeConnected || !storageLoaded || !activeExternalRunningSessionId) {
      return;
    }

    let cancelled = false;
    const inFlight = new Set<string>();
    const poll = () => {
      void pollActiveRunningSession(activeExternalRunningSessionId, inFlight, () => cancelled);
    };

    poll();
    const timer = window.setInterval(poll, EXTERNAL_RUNNING_SESSION_POLL_INTERVAL_MS);
    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, [activeExternalRunningSessionId, options.bridgeConnected, options.getSession, storageLoaded]);

  useEffect(() => {
    if (
      !activeSessionId
      || !activeSessionBridgeVersion
      || !options.bridgeConnected
      || !options.sessionsLoaded
      || !storageLoaded
    ) {
      return;
    }

    const metadata = options.sessions.find((session) => session.id === activeSessionId);
    const view = sessionViewsRef.current[activeSessionId];
    const stored = storedConversationsRef.current.find((conversation) => conversation.id === activeSessionId);
    const metadataAlreadyLoaded = Boolean(
      metadata
        && view
        && view.updatedAt >= metadata.updated_at
        && stored?.source_message_count === metadata.message_count,
    );
    if (
      !metadata
      || !view?.bridgeOwned
      || view.run.status === "running"
      || metadataAlreadyLoaded
    ) {
      return;
    }

    let cancelled = false;
    void syncActiveSessionUpdateFromBridge(metadata.id, () => cancelled);
    return () => {
      cancelled = true;
    };
  }, [
    activeSessionBridgeVersion,
    activeSessionId,
    options.bridgeConnected,
    options.getSession,
    options.sessions,
    options.sessionsLoaded,
    storageLoaded,
  ]);

  async function sendMessage(text: string, selectedSkill?: ChatSelectedSkill | null): Promise<boolean> {
    const trimmed = text.trim();
    const resolvedSelectedSkill = selectedSkill ?? undefined;
    if ((!trimmed && !resolvedSelectedSkill) || !sendAvailable || activeRun.status === "running") {
      return false;
    }

    const initialSessionId = activeSessionId?.trim() || "";
    if (initialSessionId) {
      activeSessionIdRef.current = initialSessionId;
    }
    let targetSessionId = initialSessionId;
    const requestId = createClientRunId("request");
    const traceId = createClientRunId("trace");
    const displayTitle = trimmed || resolvedSelectedSkill?.name || sessionFallbackTitle(initialSessionId);
    const userMessage = createUserConversationMessage(trimmed, initialSessionId, resolvedSelectedSkill);
    const optimisticMessages = [...activeMessages, userMessage];
    postSendFocusTokenRef.current += 1;
    setPostSendFocusRequest({
      messageId: userMessage.id,
      token: postSendFocusTokenRef.current,
    });

    if (initialSessionId) {
      setSessionViews((current) =>
        trimSessionViewsForState(
          upsertSessionView(current, initialSessionId, {
            messages: optimisticMessages,
            reply: undefined,
            run: createRunningRunState(requestId, traceId, "发送中"),
            title: current[initialSessionId]?.title || findStoredTitle(storedConversations, initialSessionId) || displayTitle,
            unread: false,
            bridgeOwned: true,
          }),
        ),
      );
    } else {
      setHomeMessages(optimisticMessages);
      setHomeReply(undefined);
      setHomeRun(createRunningRunState(requestId, traceId, "发送中"));
    }

    let result: SendAgentMessageResult = { ok: false };
    try {
      result = await options.sendAgentMessage({
        message: trimmed,
        history: optimisticMessages,
        onReply: (reply) => {
          applyReply(targetSessionId, reply);
        },
        onSessionId: (sessionId) => {
          const resolvedSessionId = sessionId.trim();
          if (!resolvedSessionId || targetSessionId === resolvedSessionId) {
            return;
          }
          targetSessionId = resolvedSessionId;
          commitCreatedSession(resolvedSessionId, displayTitle, optimisticMessages, {
            activate: shouldActivateResolvedSession(initialSessionId),
            run: createRunningRunState(requestId, traceId, "发送中"),
          });
        },
        onStatus: (status) => {
          applyRunStatus(targetSessionId, normalizeRunStatus(status, targetSessionId, requestId, traceId), requestId, traceId);
        },
        requestId,
        selectedSkill: resolvedSelectedSkill,
        sessionId: initialSessionId || undefined,
        shouldStreamRealtime: (sessionId) => shouldStreamRealtimeForRun(sessionId, initialSessionId),
        traceId,
      });
    } finally {
      clearStoppingRun(initialSessionId, requestId, traceId);
      if (targetSessionId !== initialSessionId) {
        clearStoppingRun(targetSessionId, requestId, traceId);
      }
    }

    if (!result.ok) {
      return false;
    }

    const resolvedSessionId = result.sessionId?.trim() || targetSessionId;
    if (!resolvedSessionId) {
      return true;
    }
    if (!shouldHandleSessionUpdate(resolvedSessionId, initialSessionId)) {
      return true;
    }

    if (!targetSessionId) {
      targetSessionId = resolvedSessionId;
      commitCreatedSession(resolvedSessionId, displayTitle, optimisticMessages, {
        activate: shouldActivateResolvedSession(initialSessionId),
        run: createRunningRunState(requestId, traceId, "发送中"),
      });
    }
    const finalMessages = resolveFinalConversationMessages(
      resolvedSessionId,
      optimisticMessages,
      result.reply,
    );
    commitFinalReply(resolvedSessionId, result.reply, displayTitle, finalMessages);
    if (result.mode === "local" && options.bridgeConnected && options.appendSessionMessages) {
      await syncLocalTurnToBridge(resolvedSessionId, displayTitle, finalMessages);
    }
    return true;
  }

  async function stopCurrentRun(): Promise<boolean> {
    const sessionId = activeSessionId?.trim() || "";
    const run = activeRun;
    if (run.status !== "running" || run.stopPending || isRunStopping(sessionId, run.requestId, run.traceId)) {
      return false;
    }

    rememberStoppingRun(sessionId, run.requestId, run.traceId);
    setRunStopPending(sessionId, true);
    const result = await options.stopAgentRun({
      sessionId: sessionId || undefined,
      traceId: run.traceId || undefined,
    });
    if (!result.ok) {
      clearStoppingRun(sessionId, run.requestId, run.traceId);
      setRunStopPending(sessionId, false);
      return false;
    }

    const resolvedSessionId = result.sessionId?.trim() || sessionId;
    const statusText = result.status === "not_running" ? "没有运行中的任务" : "已停止";
    applyStoppedRun(sessionId, resolvedSessionId, statusText);
    if (resolvedSessionId && result.status === "stopped") {
      await syncStoppedSession(resolvedSessionId, statusText);
    }
    return true;
  }

  async function selectSession(sessionId: string): Promise<void> {
    const trimmedSessionId = sessionId.trim();
    if (!trimmedSessionId) {
      return;
    }

    const previousSessionId = activeSessionIdRef.current?.trim() || "";
    if (previousSessionId && previousSessionId !== trimmedSessionId) {
      deactivateSessionView(previousSessionId);
    }
    setPostSendFocusRequest(null);
    const stored = storedConversations.find((conversation) => conversation.id === trimmedSessionId);
    const existing = sessionViews[trimmedSessionId];
    const shouldLoadBridgeSnapshot = options.bridgeConnected;
    const initialMessages = shouldLoadBridgeSnapshot ? [] : existing?.messages ?? stored?.messages ?? [];
    const initialReply = shouldLoadBridgeSnapshot ? undefined : existing?.reply;
    const initialRun = shouldLoadBridgeSnapshot
      ? createRunningRunState(undefined, undefined, SESSION_MESSAGES_LOADING_STATUS_TEXT)
      : existing?.run ?? createIdleRunState(stored ? SESSION_MESSAGES_LOADED_STATUS_TEXT : SESSION_MESSAGES_LOADING_STATUS_TEXT);
    activeSessionIdRef.current = trimmedSessionId;
    setActiveSessionId(trimmedSessionId);
    setHomeReply(undefined);
    setHomeRun(createIdleRunState());
    setSessionViews((current) =>
      trimSessionViewsForState(
        upsertSessionView(current, trimmedSessionId, {
          messages: initialMessages,
          reply: initialReply,
          run: initialRun,
          title: existing?.title || stored?.title || sessionFallbackTitle(trimmedSessionId),
          unread: false,
          bridgeOwned: shouldLoadBridgeSnapshot || Boolean(existing?.bridgeOwned),
          hasOlderHistory: shouldLoadBridgeSnapshot ? false : existing?.hasOlderHistory ?? false,
          loadingOlderHistory: false,
          nextHistoryBefore: shouldLoadBridgeSnapshot ? null : existing?.nextHistoryBefore ?? null,
        }),
      ),
    );

    if (!options.bridgeConnected) {
      if (!stored && !existing) {
        applyRunStatus(trimmedSessionId, { tone: "error", text: "需要先连接电脑端才能加载该历史会话" });
      }
      return;
    }
    applyRunStatus(trimmedSessionId, { tone: "loading", text: SESSION_MESSAGES_LOADING_STATUS_TEXT });
    try {
      const detail = await options.getSession(trimmedSessionId);
      if (!isActiveSession(trimmedSessionId)) {
        return;
      }
      applySessionRuntimeSelection(detail.last_runtime_selection ?? null);
      commitSessionDetail(detail, {
        run: createSuccessRunState(SESSION_MESSAGES_LOADED_STATUS_TEXT),
        unread: false,
      });
    } catch (error) {
      if (!isActiveSession(trimmedSessionId)) {
        return;
      }
      applyRunStatus(trimmedSessionId, { tone: "error", text: errorMessage(error) });
    }
  }

  async function loadOlderHistory(): Promise<void> {
    const sessionId = activeSessionIdRef.current?.trim() || "";
    if (!sessionId || !options.bridgeConnected) {
      return;
    }

    const view = sessionViewsRef.current[sessionId];
    const nextBefore = view?.nextHistoryBefore ?? null;
    if (!view?.hasOlderHistory || view.loadingOlderHistory || nextBefore === null) {
      return;
    }

    setSessionViews((current) =>
      trimSessionViewsForState(
        upsertSessionView(current, sessionId, {
          loadingOlderHistory: true,
        }),
      ),
    );

    try {
      const detail = await options.getSession(sessionId, { before: nextBefore });
      if (!isActiveSession(sessionId)) {
        return;
      }
      const currentView = sessionViewsRef.current[detail.id] ?? sessionViewsRef.current[sessionId];
      if (!currentView) {
        setSessionViews((current) =>
          trimSessionViewsForState(
            upsertSessionView(current, sessionId, {
              loadingOlderHistory: false,
            }),
          ),
        );
        return;
      }
      const messages = prependUniqueConversationMessages(
        currentView.messages,
        sessionDetailToConversationMessages(detail),
      );
      setSessionViews((current) =>
        trimSessionViewsForState(
          upsertSessionView(current, detail.id, {
            bridgeOwned: true,
            hasOlderHistory: detail.page.has_more_before,
            loadingOlderHistory: false,
            messages,
            nextHistoryBefore: detail.page.next_before ?? null,
            title: detail.title.trim() || currentView.title || sessionFallbackTitle(detail.id),
            unread: activeSessionIdRef.current !== detail.id,
            updatedAt: detail.updated_at,
          }),
        ),
      );
      persistConversation({
        createdAt: detail.created_at,
        id: detail.id,
        messages,
        sourceMessageCount: detail.message_count,
        syncedMessageCount: messages.length,
        title: detail.title.trim() || currentView.title || sessionFallbackTitle(detail.id),
        updatedAt: detail.updated_at,
      });
    } catch (error) {
      if (!isActiveSession(sessionId)) {
        return;
      }
      setSessionViews((current) =>
        trimSessionViewsForState(
          upsertSessionView(current, sessionId, {
            loadingOlderHistory: false,
          }),
        ),
      );
      applyRunStatus(sessionId, { tone: "error", text: `历史消息加载失败：${errorMessage(error)}` });
    }
  }

  function startNewSession(): void {
    deactivateSessionView(activeSessionIdRef.current);
    activeSessionIdRef.current = undefined;
    setActiveSessionId(undefined);
    setHomeMessages([]);
    setHomeReply(undefined);
    setHomeRun(createIdleRunState("新会话"));
    setPostSendFocusRequest(null);
  }

  function clearCurrentConversation(): void {
    deactivateSessionView(activeSessionIdRef.current);
    activeSessionIdRef.current = undefined;
    setActiveSessionId(undefined);
    setHomeMessages([]);
    setHomeReply(undefined);
    setHomeRun(createIdleRunState("本地消息已清空"));
    setPostSendFocusRequest(null);
  }

  function applyReply(sessionId: string, reply: AgentPayload): void {
    if (!sessionId) {
      if (activeSessionIdRef.current) {
        return;
      }
      setHomeReply(reply);
      return;
    }
    if (!isActiveSession(sessionId)) {
      return;
    }

    setSessionViews((current) =>
      trimSessionViewsForState(
        upsertSessionView(current, sessionId, {
          reply,
          run: current[sessionId]?.run ?? createRunningRunState(),
          unread: activeSessionIdRef.current !== sessionId,
          updatedAt: new Date().toISOString(),
        }),
      ),
    );
  }

  function applyRunStatus(sessionId: string, status: StatusMessage, requestId?: string, traceId?: string): void {
    const run = statusToRunState(status, requestId, traceId);
    if (run.status === "running" && isRunStopping(sessionId, requestId, traceId)) {
      run.stopPending = true;
    }
    if (!sessionId) {
      if (activeSessionIdRef.current) {
        return;
      }
      setHomeRun(run);
      return;
    }
    if (!isActiveSession(sessionId)) {
      return;
    }

    setSessionViews((current) =>
      trimSessionViewsForState(
        upsertSessionView(current, sessionId, {
          run,
          unread: activeSessionIdRef.current !== sessionId && status.tone !== "loading",
          updatedAt: new Date().toISOString(),
        }),
      ),
    );
  }

  function commitSessionDetail(
    detail: SessionDetail,
    input: {
      run: MobileSessionRunState;
      unread: boolean;
    },
  ): void {
    const messages = sessionDetailToConversationMessages(detail);
    const draftRun = detail.turn_draft ? sessionTurnDraftToRunState(detail.turn_draft) : undefined;
    const draftReply = detail.turn_draft
      ? sessionTurnDraftToAgentPayload(detail.turn_draft, detail.id, detail.last_runtime_selection)
      : undefined;
    const title = detail.title.trim()
      || findStoredTitle(storedConversationsRef.current, detail.id)
      || sessionFallbackTitle(detail.id);
    setSessionViews((current) =>
      trimSessionViewsForState(
        upsertSessionView(current, detail.id, {
          bridgeOwned: true,
          hasOlderHistory: detail.page.has_more_before,
          loadingOlderHistory: false,
          messages,
          nextHistoryBefore: detail.page.next_before ?? null,
          reply: draftReply,
          run: draftRun ?? input.run,
          title,
          unread: input.unread,
          updatedAt: detail.updated_at,
        }),
      ),
    );
    persistConversation({
      createdAt: detail.created_at,
      id: detail.id,
      messages,
      sourceMessageCount: detail.message_count,
      syncedMessageCount: messages.length,
      title,
      updatedAt: detail.updated_at,
    });
  }

  function deactivateSessionView(sessionId: string | undefined): void {
    const trimmedSessionId = sessionId?.trim() || "";
    if (!trimmedSessionId) {
      return;
    }
    setSessionViews((current) => {
      const existing = current[trimmedSessionId];
      if (!existing || existing.run.status !== "running") {
        return current;
      }
      return trimSessionViewsForState(
        upsertSessionView(current, trimmedSessionId, {
          loadingOlderHistory: false,
          reply: undefined,
          run: createIdleRunState(),
          unread: false,
        }),
      );
    });
  }

  function setRunStopPending(sessionId: string, stopPending: boolean): void {
    if (!sessionId) {
      setHomeRun((current) => ({ ...current, stopPending }));
      return;
    }

    setSessionViews((current) => {
      const existing = current[sessionId];
      if (!existing) {
        return current;
      }
      return trimSessionViewsForState(
        upsertSessionView(current, sessionId, {
          run: {
            ...existing.run,
            stopPending,
          },
        }),
      );
    });
  }

  function applyStoppedRun(fromSessionId: string, resolvedSessionId: string, statusText: string): void {
    if (resolvedSessionId && !fromSessionId) {
      commitCreatedSession(resolvedSessionId, activeMessages[0]?.text ?? sessionFallbackTitle(resolvedSessionId), activeMessages, {
        activate: true,
      });
    }
    applyRunStatus(resolvedSessionId || fromSessionId, { tone: "success", text: statusText });
  }

  async function syncStoppedSession(sessionId: string, statusText: string): Promise<void> {
    try {
      const detail = await options.getSession(sessionId);
      if (!isActiveSession(detail.id)) {
        return;
      }
      const messages = sessionDetailToConversationMessages(detail);
      const title = detail.title.trim() || findStoredTitle(storedConversationsRef.current, detail.id) || sessionFallbackTitle(detail.id);
      setSessionViews((current) =>
        trimSessionViewsForState(
          upsertSessionView(current, detail.id, {
            bridgeOwned: true,
            messages,
            reply: undefined,
            run: createSuccessRunState(statusText),
            title,
            unread: activeSessionIdRef.current !== detail.id,
            updatedAt: detail.updated_at,
            hasOlderHistory: detail.page.has_more_before,
            loadingOlderHistory: false,
            nextHistoryBefore: detail.page.next_before ?? null,
          }),
        ),
      );
      persistConversation({
        createdAt: detail.created_at,
        id: detail.id,
        messages,
        sourceMessageCount: detail.message_count,
        syncedMessageCount: messages.length,
        title,
        updatedAt: detail.updated_at,
      });
    } catch (error) {
      if (!isActiveSession(sessionId)) {
        return;
      }
      applyRunStatus(sessionId, { tone: "error", text: `停止后同步失败：${errorMessage(error)}` });
    }
  }

  function commitCreatedSession(
    sessionId: string,
    title: string,
    messages: MobileConversationMessage[],
    input: { activate: boolean; run?: MobileSessionRunState },
  ): void {
    const normalizedMessages = normalizeConversationSessionIds(messages, sessionId);
    if (input.activate) {
      activeSessionIdRef.current = sessionId;
      setActiveSessionId(sessionId);
      setHomeMessages([]);
      setHomeReply(undefined);
      setHomeRun(createIdleRunState());
    } else {
      return;
    }
    setSessionViews((current) =>
      trimSessionViewsForState(
        upsertSessionView(current, sessionId, {
          messages: normalizedMessages,
          reply: input.activate ? homeReply : undefined,
          run: input.run ?? (homeRun.status === "running" ? homeRun : createRunningRunState()),
          title,
          unread: !input.activate,
          bridgeOwned: true,
        }),
      ),
    );
    persistConversation({
      id: sessionId,
      messages: normalizedMessages,
      preserveExistingTitle: true,
      title,
    });
  }

  function shouldActivateResolvedSession(initialSessionId: string): boolean {
    const currentSessionId = activeSessionIdRef.current?.trim() || "";
    if (!initialSessionId) {
      return !currentSessionId;
    }
    return currentSessionId === initialSessionId;
  }

  function shouldHandleSessionUpdate(sessionId: string, initialSessionId: string): boolean {
    const currentSessionId = activeSessionIdRef.current?.trim() || "";
    if (currentSessionId) {
      return currentSessionId === sessionId.trim();
    }
    return !initialSessionId.trim();
  }

  function shouldStreamRealtimeForRun(sessionId: string, initialSessionId: string): boolean {
    const resolvedSessionId = sessionId.trim() || initialSessionId.trim();
    const currentSessionId = activeSessionIdRef.current?.trim() || "";
    if (!resolvedSessionId) {
      return !currentSessionId;
    }
    if (!initialSessionId.trim() && !currentSessionId) {
      return true;
    }
    return currentSessionId === resolvedSessionId;
  }

  function isActiveSession(sessionId: string): boolean {
    const trimmedSessionId = sessionId.trim();
    return Boolean(trimmedSessionId && activeSessionIdRef.current?.trim() === trimmedSessionId);
  }

  function commitFinalReply(
    sessionId: string,
    reply: AgentPayload | undefined,
    title: string,
    finalMessages: MobileConversationMessage[],
  ): void {
    setSessionViews((current) => {
      const existing = current[sessionId];
      const next = upsertSessionView(current, sessionId, {
        messages: finalMessages,
        reply: undefined,
        run: createSuccessRunState(reply?.session_ended ? "会话已结束" : "回复已返回", reply?.session_ended),
        title: existing?.title || findStoredTitle(storedConversations, sessionId) || title,
        unread: activeSessionIdRef.current !== sessionId,
        bridgeOwned: true,
      });
      persistConversation({
        id: sessionId,
        messages: finalMessages,
        preserveExistingTitle: true,
        title,
      });
      return trimSessionViewsForState(next);
    });
  }

  function persistConversation(input: PersistConversationInput): void {
    setStoredConversations((current) => {
      const next = trimStoredConversationsForState(upsertStoredMobileConversation(current, input));
      saveConversationsInBackground(next);
      return next;
    });
  }

  async function syncLocalTurnToBridge(
    sessionId: string,
    title: string,
    finalMessages: MobileConversationMessage[],
  ): Promise<void> {
    const stored = storedConversationsRef.current.find((conversation) => conversation.id === sessionId);
    const expectedHead = stored?.source_message_count ?? 0;
    const syncedMessageCount = stored?.synced_message_count ?? 0;
    const unsyncedMessages = finalMessages.slice(syncedMessageCount);
    if (unsyncedMessages.length === 0 || !options.appendSessionMessages) {
      return;
    }

    const result = await options.appendSessionMessages({
      expectedHead,
      messages: unsyncedMessages,
      sessionId,
      title: findStoredTitle(storedConversationsRef.current, sessionId) || title,
    });
    if (result.status === "conflict") {
      await replaceConversationFromBridge(sessionId, "电脑端已有更新，本地回合未同步");
      return;
    }

    persistConversation({
      bridgeMessageCount: result.messageCount,
      id: sessionId,
      messages: finalMessages,
      preserveExistingTitle: true,
      syncedMessageCount: finalMessages.length,
      title,
      updatedAt: result.updatedAt,
    });
  }

  async function replaceConversationFromBridge(sessionId: string, statusText: string): Promise<void> {
    const detail = await options.getFullSession(sessionId);
    if (!isActiveSession(detail.id)) {
      return;
    }
    const messages = sessionDetailToConversationMessages(detail);
    const title = detail.title.trim() || findStoredTitle(storedConversationsRef.current, detail.id) || sessionFallbackTitle(detail.id);
    setSessionViews((current) =>
      trimSessionViewsForState(
        upsertSessionView(current, detail.id, {
          bridgeOwned: true,
          messages,
          reply: undefined,
          run: createSuccessRunState(statusText),
          title,
          unread: activeSessionIdRef.current !== detail.id,
          updatedAt: detail.updated_at,
          hasOlderHistory: false,
          loadingOlderHistory: false,
          nextHistoryBefore: null,
        }),
      ),
    );
    persistConversation({
      bridgeMessageCount: detail.message_count,
      createdAt: detail.created_at,
      id: detail.id,
      messages,
      sourceMessageCount: detail.message_count,
      syncedMessageCount: messages.length,
      title,
      updatedAt: detail.updated_at,
    });
  }

  async function pollActiveRunningSession(
    sessionId: string,
    inFlight: Set<string>,
    isCancelled: () => boolean,
  ): Promise<void> {
    if (inFlight.has(sessionId)) {
      return;
    }
    inFlight.add(sessionId);
    try {
      const detail = await options.getSession(sessionId);
      if (isCancelled() || !isActiveSession(detail.id)) {
        return;
      }
      applySessionRuntimeSelection(detail.last_runtime_selection ?? null);
      commitSessionDetail(detail, {
        run: isActiveSessionTurnDraft(detail.turn_draft)
          ? sessionTurnDraftToRunState(detail.turn_draft)
          : createSuccessRunState("回复已返回"),
        unread: activeSessionIdRef.current !== detail.id,
      });
    } catch (error) {
      if (!isCancelled() && isActiveSession(sessionId)) {
        applyRunStatus(sessionId, { tone: "error", text: `运行会话同步失败：${errorMessage(error)}` });
      }
    } finally {
      inFlight.delete(sessionId);
    }
  }

  async function syncActiveSessionUpdateFromBridge(
    sessionId: string,
    isCancelled: () => boolean,
  ): Promise<void> {
    try {
      const detail = await options.getSession(sessionId);
      if (isCancelled()) {
        return;
      }
      applySessionRuntimeSelection(detail.last_runtime_selection ?? null);
      commitSessionDetail(detail, {
        run: createSuccessRunState("历史会话已更新"),
        unread: false,
      });
    } catch (error) {
      if (!isCancelled()) {
        applyRunStatus(sessionId, { tone: "error", text: `会话更新同步失败：${errorMessage(error)}` });
      }
    }
  }

  function trimSessionViewsForState(views: Record<string, MobileSessionView>): Record<string, MobileSessionView> {
    return trimMobileSessionViews(views, {
      activeSessionId: activeSessionIdRef.current,
      pinnedSessionIds: new Set(options.pinnedHistoryIds),
    });
  }

  function trimStoredConversationsForState(conversations: StoredMobileConversation[]): StoredMobileConversation[] {
    return trimStoredMobileConversations(conversations, {
      activeSessionId: activeSessionIdRef.current,
      bridgeSessionIds: new Set(options.sessions.map((session) => session.id)),
      pinnedSessionIds: new Set(options.pinnedHistoryIds),
      runningSessionIds: runningSessionIds(sessionViewsRef.current),
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

  function applySessionRuntimeSelection(selection: SessionRuntimeSelection | null | undefined): void {
    if (!options.onSessionRuntimeSelection) {
      return;
    }
    void Promise.resolve(options.onSessionRuntimeSelection(selection)).catch((error: unknown) => {
      console.error("[useMobileSessions] session runtime selection failed", error);
    });
  }

  async function syncComputerSessions(runId: number): Promise<SyncComputerSessionsResult> {
    const changed: StoredMobileConversation[] = [];
    const currentById = new Map(storedConversationsRef.current.map((conversation) => [conversation.id, conversation]));
    const sessionsToSync = recentMobileBridgeSessions(options.sessions);

    if (options.appendSessionMessages) {
      for (const conversation of storedConversationsRef.current) {
        if (syncRunIdRef.current !== runId) {
          return { syncedCount: 0, totalCount: options.sessions.length };
        }
        const syncedMessageCount = conversation.synced_message_count ?? 0;
        const unsyncedMessages = conversation.messages.slice(syncedMessageCount);
        if (unsyncedMessages.length === 0) {
          continue;
        }

        const result = await options.appendSessionMessages({
          expectedHead: conversation.source_message_count ?? 0,
          messages: unsyncedMessages,
          sessionId: conversation.id,
          title: conversation.title,
        });
        if (result.status === "conflict") {
          const detail = await options.getSession(conversation.id, { limit: MOBILE_PERSISTED_SESSION_PAGE_LIMIT });
          const syncedConversation = storedConversationFromSessionDetail(detail);
          changed.push(syncedConversation);
          currentById.set(detail.id, syncedConversation);
          continue;
        }

        const syncedConversation: StoredMobileConversation = {
          ...conversation,
          source_message_count: result.messageCount,
          synced_message_count: conversation.messages.length,
          updated_at: result.updatedAt,
        };
        changed.push(syncedConversation);
        currentById.set(conversation.id, syncedConversation);
      }
    }

    for (const session of sessionsToSync) {
      if (syncRunIdRef.current !== runId) {
        return { syncedCount: 0, totalCount: options.sessions.length };
      }
      const cached = currentById.get(session.id);
      if (cached && isStoredConversationCurrent(cached, session)) {
        continue;
      }

      const detail = await options.getSession(session.id, { limit: MOBILE_PERSISTED_SESSION_PAGE_LIMIT });
      const syncedConversation = storedConversationFromSessionDetail(detail);
      changed.push(syncedConversation);
      currentById.set(detail.id, syncedConversation);
    }

    if (syncRunIdRef.current !== runId) {
      return { syncedCount: 0, totalCount: options.sessions.length };
    }

    const next = trimStoredConversationsForState(mergeSyncedConversations(storedConversationsRef.current, changed));
    await savePersistedMobileConversations(next);
    if (syncRunIdRef.current === runId) {
      setStoredConversations(next);
    }
    return { syncedCount: sessionsToSync.length, totalCount: options.sessions.length };
  }

  return {
    activeMessages,
    activeReply,
    activeSessionId,
    activeStatus,
    canStop: activeRun.status === "running" && !activeRun.stopPending,
    canSend: sendAvailable && activeRun.status !== "running",
    clearCurrentConversation,
    computerSessionPersistStatus,
    hasConversation: activeMessages.length > 0 || Boolean(activeReply),
    hasOlderHistory,
    historyItems,
    loadOlderHistory,
    loadingOlderHistory,
    loadingSessionMessages,
    postSendFocusRequest,
    selectSession,
    sendMessage,
    startNewSession,
    stopCurrentRun,
  };

  function normalizeRunStatus(
    status: StatusMessage,
    sessionId: string,
    requestId: string,
    traceId: string,
  ): StatusMessage {
    if (isCancellationStatus(status) && isRunStopping(sessionId, requestId, traceId)) {
      return { tone: "success", text: "已停止" };
    }
    return status;
  }

  function rememberStoppingRun(sessionId: string, requestId?: string, traceId?: string): void {
    for (const key of runKeys(sessionId, requestId, traceId)) {
      stoppingRunKeysRef.current.add(key);
    }
  }

  function clearStoppingRun(sessionId: string, requestId?: string, traceId?: string): void {
    for (const key of runKeys(sessionId, requestId, traceId)) {
      stoppingRunKeysRef.current.delete(key);
    }
  }

  function isRunStopping(sessionId: string, requestId?: string, traceId?: string): boolean {
    return runKeys(sessionId, requestId, traceId).some((key) => stoppingRunKeysRef.current.has(key));
  }
}

function runKeys(sessionId: string, requestId?: string, traceId?: string): string[] {
  return [
    sessionId.trim() ? `session:${sessionId.trim()}` : "",
    requestId?.trim() ? `request:${requestId.trim()}` : "",
    traceId?.trim() ? `trace:${traceId.trim()}` : "",
  ].filter(Boolean);
}

function isCancellationStatus(status: StatusMessage): boolean {
  if (status.tone !== "error") {
    return false;
  }
  return isAgentRunCancellationMessage(status.text);
}

function isStoredConversationCurrent(conversation: StoredMobileConversation, session: SessionMetadata): boolean {
  return conversation.updated_at === session.updated_at
    && conversation.source_message_count === session.message_count
    && conversation.synced_message_count === conversation.messages.length;
}

function shouldPollExternalRunningSession(view: MobileSessionView): boolean {
  return view.bridgeOwned && view.run.status === "running" && !view.run.requestId && Boolean(view.run.traceId);
}

function storedConversationFromSessionDetail(detail: SessionDetail): StoredMobileConversation {
  const messages = sessionDetailToConversationMessages(detail);
  return {
    created_at: detail.created_at,
    id: detail.id,
    messages,
    source_message_count: detail.message_count,
    synced_message_count: messages.length,
    title: detail.title.trim() || sessionFallbackTitle(detail.id),
    updated_at: detail.updated_at,
  };
}

function syncComputerSessionsStatusText(result: SyncComputerSessionsResult): string {
  if (result.syncedCount < result.totalCount) {
    return `已同步 ${result.syncedCount}/${result.totalCount} 个最近会话`;
  }
  return `已同步 ${result.syncedCount} 个`;
}

function runningSessionIds(views: Record<string, MobileSessionView>): Set<string> {
  return new Set(
    Object.values(views)
      .filter((view) => view.run.status === "running")
      .map((view) => view.id),
  );
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

function prependUniqueConversationMessages(
  current: MobileConversationMessage[],
  older: MobileConversationMessage[],
): MobileConversationMessage[] {
  const currentIds = new Set(current.map((message) => message.id));
  return [...older.filter((message) => !currentIds.has(message.id)), ...current];
}

function initialStoredConversations(): StoredMobileConversation[] {
  return hasTauriRuntime() ? [] : loadStoredMobileConversations();
}

function resolveFinalConversationMessages(
  sessionId: string,
  baseMessages: MobileConversationMessage[],
  reply: AgentPayload | undefined,
): MobileConversationMessage[] {
  const normalizedMessages = normalizeConversationSessionIds(baseMessages, sessionId);
  const assistantMessage = reply ? createAssistantConversationMessage(reply, sessionId) : undefined;
  if (!assistantMessage) {
    return normalizedMessages;
  }
  return appendStoredMobileMessages({ ...emptyStoredConversation(sessionId), messages: normalizedMessages }, [
    ...normalizedMessages,
    assistantMessage,
  ]);
}
