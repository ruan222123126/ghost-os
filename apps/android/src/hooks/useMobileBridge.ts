import { invoke } from "@tauri-apps/api/core";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createTraceId, errorMessage, hasTauriRuntime } from "../lib/bridgeBus";
import type { BridgeBusCommand, BridgeEnvelope } from "../lib/bridgeBus";
import { streamAgentMessageHTTP } from "../lib/agentStream";
import type { AgentStreamEvent } from "../lib/agentStream";
import {
  createAgentPayloadFromRuntime,
  createMobileAgentStreamRuntime,
  projectMobileAgentStreamEvent,
  streamAgentMessageWebRTC,
} from "../lib/mobileAgentStreamRuntime";
import type { MobileAgentStreamProjector } from "../lib/mobileAgentStreamRuntime";
import { loadMobileCredential } from "../lib/mobileCredentials";
import { sendLocalLLMMessage } from "../lib/mobileLocalLLM";
import {
  createOrUpdateLocalProvider,
  deleteLocalProvider as deleteStoredLocalProvider,
  exportLocalProvider,
  loadLocalProviderList,
} from "../lib/mobileLocalProviders";
import { parseSessionDetail, parseSessionMetadataList } from "../lib/sessionPayloadParser";
import { MobileWebRTCBridge } from "../lib/mobileWebRTC";
import { loadSettings, normalizeBridgeUrl, saveSettings } from "../lib/settingsStorage";
import type {
  AgentPayload,
  ConfigPayload,
  HostProfile,
  MobileConversationMessage,
  OrchestrationTaskPayload,
  ProviderConfigInputPayload,
  ProviderConfigPayload,
  ProviderListPayload,
  SessionDetail,
  SessionMetadata,
  SkillPayload,
  StatusMessage,
  StoredConnectionSnapshot,
  StoredSettings,
  TaskPayload,
  UnknownTaskPayload,
} from "../mobileTypes";

const SESSION_DETAIL_PAGE_LIMIT = 100;
const SESSION_FULL_PAGE_LIMIT = 200;
const UNSUPPORTED_SKILL_MANAGEMENT_TEXT = "电脑端不支持技能管理";

interface SendAgentMessageOptions {
  message: string;
  history: MobileConversationMessage[];
  onReply: (reply: AgentPayload) => void;
  onSessionId: (sessionId: string) => void;
  onStatus: (status: StatusMessage) => void;
  requestId?: string;
  sessionId?: string;
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

interface AgentStopPayload {
  message: string;
  session_id?: string;
  status: "stopped" | "not_running";
}

function connectionTargetKey(settings: StoredSettings, bridgeUrl: string): string {
  if (settings.connectionMode === "http") {
    return `http:${bridgeUrl}:${settings.apiToken?.trim() ?? ""}`;
  }

  const pairing = settings.pairing;
  if (!pairing) {
    return "webrtc:";
  }

  return `webrtc:${pairing.deviceId}:${pairing.pcId}:${pairing.signalingUrl}`;
}

function connectionSnapshotFromSettings(
  settings: StoredSettings,
  bridgeUrl: string,
): StoredConnectionSnapshot {
  const snapshot: StoredConnectionSnapshot = {
    apiToken: settings.apiToken?.trim() || "",
    bridgeUrl,
    connectionMode: settings.connectionMode,
  };
  if (settings.connectionMode === "webrtc") {
    snapshot.pairing = settings.pairing;
  }
  return snapshot;
}

function settingsWithConnectionSnapshot(
  settings: StoredSettings,
  snapshot: StoredConnectionSnapshot,
): StoredSettings {
  return {
    ...settings,
    apiToken: snapshot.apiToken?.trim() || "",
    bridgeUrl: snapshot.bridgeUrl,
    connectionMode: snapshot.connectionMode,
    pairing: snapshot.connectionMode === "webrtc" ? snapshot.pairing : settings.pairing,
  };
}

function connectionSettingsMatchSnapshot(
  settings: StoredSettings,
  snapshot: StoredConnectionSnapshot,
): boolean {
  if (
    settings.connectionMode !== snapshot.connectionMode
    || normalizeBridgeUrl(settings.bridgeUrl) !== normalizeBridgeUrl(snapshot.bridgeUrl)
    || (settings.apiToken?.trim() || "") !== (snapshot.apiToken?.trim() || "")
  ) {
    return false;
  }

  if (snapshot.connectionMode === "http") {
    return true;
  }
  return Boolean(
    settings.pairing
    && snapshot.pairing
    && connectionPairingKey(settings.pairing) === connectionPairingKey(snapshot.pairing),
  );
}

function connectionPairingKey(pairing: StoredSettings["pairing"]): string {
  if (!pairing) {
    return "";
  }
  return `${pairing.deviceId}:${pairing.pcId}:${pairing.signalingUrl}:${pairing.signalingToken}`;
}

function resolveConnectedWebRTCClient(client: MobileWebRTCBridge | undefined): MobileWebRTCBridge {
  if (!client) {
    throw new Error("WebRTC 尚未连接");
  }
  return client;
}

function connectedStatusText(mode: StoredSettings["connectionMode"]): string {
  return mode === "webrtc" ? "WebRTC 已连接" : "HTTP fallback 已连接";
}

function stringsEqualIgnoreCase(left: string, right: string): boolean {
  return left.trim().toLowerCase() === right.trim().toLowerCase();
}

function withLocalProviderSelection(
  providerList: ProviderListPayload | undefined,
  settings: StoredSettings,
): ProviderListPayload | undefined {
  if (!providerList) {
    return undefined;
  }
  const activeProvider = providerList.providers.find((provider) => provider.provider_id === settings.localProviderId)
    ?? providerList.providers.find((provider) => !provider.deleted_at);
  return {
    ...providerList,
    active_provider: activeProvider?.name || "",
    active_provider_id: activeProvider?.provider_id,
  };
}

function mergeLocalProvidersWithRemote(
  localProviderList: ProviderListPayload | undefined,
  remoteProviderList: ProviderListPayload | undefined,
): ProviderListPayload | undefined {
  if (!localProviderList && !remoteProviderList) {
    return undefined;
  }

  const merged = new Map<string, ProviderConfigPayload>();
  for (const provider of localProviderList?.providers ?? []) {
    merged.set(provider.provider_id, provider);
  }
  for (const provider of remoteProviderList?.providers ?? []) {
    const current = merged.get(provider.provider_id);
    if (!current || provider.updated_at >= current.updated_at) {
      merged.set(provider.provider_id, {
        ...provider,
        sync_state: "synced",
      });
    }
  }

  return {
    active_provider: remoteProviderList?.active_provider || localProviderList?.active_provider || "",
    active_provider_id: remoteProviderList?.active_provider_id || localProviderList?.active_provider_id,
    provider_sync_records: remoteProviderList?.provider_sync_records ?? localProviderList?.provider_sync_records,
    providers: [...merged.values()],
  };
}

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function isUnsupportedActionError(error: unknown, action: string): boolean {
  const pattern = new RegExp(`unsupported action\\s*:?[\\s"']+${escapeRegExp(action)}(?:\\b|["'])`, "iu");
  return pattern.test(errorMessage(error));
}

function skillListErrorText(error: unknown): string {
  if (isUnsupportedActionError(error, "SKILL_LIST")) {
    return UNSUPPORTED_SKILL_MANAGEMENT_TEXT;
  }
  return `技能列表加载失败：${errorMessage(error)}`;
}

function isUserTask(task: TaskPayload | UnknownTaskPayload): task is TaskPayload {
  return task.task_kind === "agent_message" || task.task_kind === "workflow";
}

function filterUserTasks(tasks: Array<TaskPayload | UnknownTaskPayload>): TaskPayload[] {
  return tasks.filter(isUserTask);
}

function isOrchestrationTask(
  task: OrchestrationTaskPayload | TaskPayload | UnknownTaskPayload,
): task is OrchestrationTaskPayload {
  return task.task_kind === "orchestration";
}

function filterOrchestrationTasks(
  tasks: Array<OrchestrationTaskPayload | TaskPayload | UnknownTaskPayload>,
): OrchestrationTaskPayload[] {
  return tasks.filter(isOrchestrationTask);
}

function upsertById<TItem extends { id: string }>(
  current: TItem[] | undefined,
  task: TItem,
): TItem[] {
  const existing = current ?? [];
  if (existing.some((item) => item.id === task.id)) {
    return existing.map((item) => (item.id === task.id ? task : item));
  }
  return [task, ...existing];
}

export function useMobileBridge() {
  const [settings, setSettings] = useState<StoredSettings>(() => loadSettings());
  const [host, setHost] = useState<HostProfile>();
  const [config, setConfig] = useState<ConfigPayload>();
  const [providerList, setProviderList] = useState<ProviderListPayload>();
  const [localProviderList, setLocalProviderList] = useState<ProviderListPayload>();
  const [skillList, setSkillList] = useState<SkillPayload[]>();
  const [skillListError, setSkillListError] = useState("");
  const [taskList, setTaskList] = useState<TaskPayload[]>();
  const [taskListError, setTaskListError] = useState("");
  const [orchestrationList, setOrchestrationList] = useState<OrchestrationTaskPayload[]>();
  const [orchestrationListError, setOrchestrationListError] = useState("");
  const [runningTaskId, setRunningTaskId] = useState("");
  const [sessions, setSessions] = useState<SessionMetadata[]>([]);
  const [sessionsLoaded, setSessionsLoaded] = useState(false);
  const [status, setStatus] = useState<StatusMessage>({
    tone: "idle",
    text: "未连接",
  });
  const [connectionStatus, setConnectionStatus] = useState<StatusMessage>({
    tone: "idle",
    text: "未连接",
  });
  const webRTCClientRef = useRef<MobileWebRTCBridge | undefined>(undefined);
  const connectedTargetRef = useRef<string | undefined>(undefined);
  const autoConnectAttemptedRef = useRef(false);
  const pendingAutoConnectTargetRef = useRef<string | undefined>(undefined);

  const bridgeUrl = useMemo(() => normalizeBridgeUrl(settings.bridgeUrl), [settings.bridgeUrl]);
  const apiToken = useMemo(() => settings.apiToken?.trim() || "", [settings.apiToken]);
  const currentConnectionTarget = useMemo(() => connectionTargetKey(settings, bridgeUrl), [bridgeUrl, settings]);

  useEffect(() => {
    if (!hasTauriRuntime()) {
      return;
    }

    try {
      void invoke<HostProfile>("host_profile")
        .then(setHost)
        .catch((error: unknown) => {
          setStatus({ tone: "error", text: errorMessage(error) });
        });
    } catch (error) {
      setStatus({ tone: "error", text: errorMessage(error) });
    }
  }, []);

  useEffect(() => {
    saveSettings(settings);
  }, [settings]);

  useEffect(() => {
    void loadLocalProviderList()
      .then((payload) => {
        setLocalProviderList(payload);
      })
      .catch((error: unknown) => {
        console.error("[useMobileBridge] load local providers failed", error);
      });
  }, []);

  useEffect(() => {
    return () => {
      webRTCClientRef.current?.close();
    };
  }, []);

  useEffect(() => {
    if (connectionStatus.tone !== "success" || connectedTargetRef.current === currentConnectionTarget) {
      return;
    }

    connectedTargetRef.current = undefined;
    webRTCClientRef.current?.close();
    webRTCClientRef.current = undefined;
    setConfig(undefined);
    setProviderList(undefined);
    setSkillList(undefined);
    setSkillListError("");
    setTaskList(undefined);
    setTaskListError("");
    setOrchestrationList(undefined);
    setOrchestrationListError("");
    setRunningTaskId("");
    setSessions([]);
    setSessionsLoaded(false);
    setConnectionStatus({ tone: "idle", text: "未连接" });
  }, [connectionStatus.tone, currentConnectionTarget]);

  const requestBridgeHTTP = useCallback(
    async <TPayload,>(action: string, params: Record<string, unknown>): Promise<TPayload> => {
      const traceId = createTraceId(`android-${action.toLowerCase()}`);

      const request: BridgeBusCommand = {
        baseUrl: bridgeUrl,
        apiToken: apiToken.trim() || undefined,
        action,
        params,
        traceId,
      };
      const envelope = await invoke<BridgeEnvelope<TPayload>>("bridge_bus_request", { request });
      if (envelope.status === "error") {
        throw new Error(envelope.error || "Bridge returned an error envelope");
      }
      return envelope.payload;
    },
    [apiToken, bridgeUrl],
  );

  const requestBridgeWebRTC = useCallback(
    async <TPayload,>(action: string, params: Record<string, unknown>): Promise<TPayload> => {
      const client = webRTCClientRef.current;
      if (!client) {
        throw new Error("WebRTC 尚未连接");
      }
      const traceId = createTraceId(`android-${action.toLowerCase()}`);
      return client.request<TPayload>(action, params, traceId);
    },
    [],
  );

  const requestBridge = useCallback(
    async <TPayload,>(action: string, params: Record<string, unknown>): Promise<TPayload> => {
      if (settings.connectionMode === "http") {
        return requestBridgeHTTP<TPayload>(action, params);
      }
      return requestBridgeWebRTC<TPayload>(action, params);
    },
    [requestBridgeHTTP, requestBridgeWebRTC, settings.connectionMode],
  );

  const loadProviders = useCallback(async (): Promise<ProviderListPayload> => {
    const payload = await requestBridge<ProviderListPayload>("CONFIG_PROVIDERS_GET", {});
    setProviderList(payload);
    return payload;
  }, [requestBridge]);

  const syncLocalProvidersToRemote = useCallback(
    async (remote: ProviderListPayload): Promise<void> => {
      const local = await loadLocalProviderList();
      const remoteById = new Map((remote.provider_sync_records ?? []).map((record) => [record.provider_id, record]));

      for (const provider of local.providers) {
        const remoteRecord = remoteById.get(provider.provider_id);
        if (remoteRecord && remoteRecord.updated_at >= provider.updated_at) {
          continue;
        }

        const exported = await exportLocalProvider(provider.provider_id);
        if (exported.deleted_at) {
          await requestBridge<ProviderListPayload>("CONFIG_PROVIDER_DELETE", {
            deleted_at: exported.deleted_at,
            name: exported.name,
            provider_id: exported.provider_id,
            updated_at: exported.updated_at,
          });
          continue;
        }

        if (remoteRecord) {
          await requestBridge<ProviderListPayload>("CONFIG_PROVIDER_UPDATE", {
            name: exported.name,
            provider: { ...exported } as Record<string, unknown>,
          });
          continue;
        }

        await requestBridge<ProviderListPayload>("CONFIG_PROVIDER_CREATE", { ...exported });
      }
    },
    [requestBridge],
  );

  const syncRemoteProvidersToLocal = useCallback(async (remote: ProviderListPayload): Promise<void> => {
    const local = await loadLocalProviderList();
    const localById = new Map(local.providers.map((provider) => [provider.provider_id, provider]));

    for (const record of remote.provider_sync_records ?? []) {
      const localProvider = localById.get(record.provider_id);
      if (localProvider && localProvider.updated_at > record.updated_at) {
        continue;
      }

      await createOrUpdateLocalProvider({
        base_url: record.base_url,
        deleted_at: record.deleted_at,
        model_context_window_tokens: record.model_context_window_tokens,
        model_response_reserve_tokens: record.model_response_reserve_tokens,
        models: record.models,
        name: record.name || localProvider?.name || record.provider_id,
        provider_id: record.provider_id,
        response_reserve_tokens: record.response_reserve_tokens,
        type: record.type ?? localProvider?.type ?? "openai",
        updated_at: record.updated_at,
      });
    }

    setLocalProviderList(await loadLocalProviderList());
  }, []);

  const loadSkills = useCallback(async (): Promise<SkillPayload[]> => {
    setSkillListError("");
    const payload = await requestBridge<SkillPayload[]>("SKILL_LIST", {});
    setSkillList(payload);
    return payload;
  }, [requestBridge]);

  const loadTasks = useCallback(async (): Promise<TaskPayload[]> => {
    setTaskListError("");
    const payload = filterUserTasks(
      await requestBridge<Array<TaskPayload | UnknownTaskPayload>>("TASK_LIST", { scope: "user" }),
    );
    setTaskList(payload);
    return payload;
  }, [requestBridge]);

  const loadOrchestrations = useCallback(async (): Promise<OrchestrationTaskPayload[]> => {
    setOrchestrationListError("");
    const payload = filterOrchestrationTasks(
      await requestBridge<Array<OrchestrationTaskPayload | TaskPayload | UnknownTaskPayload>>(
        "TASK_LIST",
        { scope: "orchestration" },
      ),
    );
    setOrchestrationList(payload);
    return payload;
  }, [requestBridge]);

  const refreshRuntimeConfig = useCallback(async (): Promise<ConfigPayload> => {
    const [configPayload, providersPayload] = await Promise.all([
      requestBridge<ConfigPayload>("CONFIG_GET", {}),
      loadProviders(),
    ]);
    setConfig(configPayload);
    setProviderList(providersPayload);
    return configPayload;
  }, [loadProviders, requestBridge]);

  const refreshLocalProviders = useCallback(async (): Promise<ProviderListPayload> => {
    const payload = await loadLocalProviderList();
    setLocalProviderList(payload);
    return payload;
  }, []);

  const refreshProviders = useCallback(async (): Promise<boolean> => {
    setStatus({ tone: "loading", text: "供应商刷新中" });
    try {
      await Promise.all([
        refreshLocalProviders(),
        connectionStatus.tone === "success" ? loadProviders() : Promise.resolve(undefined),
      ]);
      setStatus({ tone: "success", text: "供应商已刷新" });
      return true;
    } catch (error) {
      setStatus({ tone: "error", text: errorMessage(error) });
      return false;
    }
  }, [connectionStatus.tone, loadProviders, refreshLocalProviders]);

  const syncProvidersBidirectionally = useCallback(async (): Promise<void> => {
    if (connectionStatus.tone !== "success") {
      return;
    }
    const remote = await loadProviders();
    await syncLocalProvidersToRemote(remote);
    const refreshedRemote = await loadProviders();
    await syncRemoteProvidersToLocal(refreshedRemote);
  }, [connectionStatus.tone, loadProviders, syncLocalProvidersToRemote, syncRemoteProvidersToLocal]);

  useEffect(() => {
    if (connectionStatus.tone === "success") {
      void syncProvidersBidirectionally().catch((error: unknown) => {
        console.error("[useMobileBridge] provider sync failed", error);
      });
      return;
    }
    setSettings((current) => (current.remoteExecutionEnabled
      ? { ...current, remoteExecutionEnabled: false }
      : current));
  }, [connectionStatus.tone, syncProvidersBidirectionally]);

  const selectedLocalProviderList = useMemo(
    () => withLocalProviderSelection(localProviderList, settings),
    [localProviderList, settings],
  );
  const mergedProviderList = useMemo(
    () => mergeLocalProvidersWithRemote(selectedLocalProviderList, providerList),
    [providerList, selectedLocalProviderList],
  );

  const refreshSkills = useCallback(async (): Promise<boolean> => {
    setStatus({ tone: "loading", text: "技能刷新中" });
    try {
      await loadSkills();
      setStatus({ tone: "success", text: "技能已刷新" });
      return true;
    } catch (error) {
      const text = skillListErrorText(error);
      setSkillListError(text);
      if (isUnsupportedActionError(error, "SKILL_LIST")) {
        setSkillList(undefined);
      }
      setStatus({ tone: "error", text });
      return false;
    }
  }, [loadSkills]);

  const refreshTasks = useCallback(async (): Promise<boolean> => {
    setStatus({ tone: "loading", text: "任务刷新中" });
    try {
      await loadTasks();
      setStatus({ tone: "success", text: "任务已刷新" });
      return true;
    } catch (error) {
      const text = `任务列表加载失败：${errorMessage(error)}`;
      setTaskListError(text);
      setStatus({ tone: "error", text });
      return false;
    }
  }, [loadTasks]);

  const refreshOrchestrations = useCallback(async (): Promise<boolean> => {
    setStatus({ tone: "loading", text: "编排刷新中" });
    try {
      await loadOrchestrations();
      setStatus({ tone: "success", text: "编排已刷新" });
      return true;
    } catch (error) {
      const text = `编排列表加载失败：${errorMessage(error)}`;
      setOrchestrationListError(text);
      setStatus({ tone: "error", text });
      return false;
    }
  }, [loadOrchestrations]);

  const refreshSkillsAfterConnect = useCallback((): void => {
    void loadSkills().catch((error: unknown) => {
      const text = skillListErrorText(error);
      console.error("[useMobileBridge] load skills after connect failed", error);
      setSkillListError(text);
      if (isUnsupportedActionError(error, "SKILL_LIST")) {
        setSkillList(undefined);
      }
    });
  }, [loadSkills]);

  const refreshTasksAfterConnect = useCallback((): void => {
    void loadTasks().catch((error: unknown) => {
      const text = `任务列表加载失败：${errorMessage(error)}`;
      console.error("[useMobileBridge] load tasks after connect failed", error);
      setTaskListError(text);
    });
  }, [loadTasks]);

  const refreshOrchestrationsAfterConnect = useCallback((): void => {
    void loadOrchestrations().catch((error: unknown) => {
      const text = `编排列表加载失败：${errorMessage(error)}`;
      console.error("[useMobileBridge] load orchestrations after connect failed", error);
      setOrchestrationListError(text);
    });
  }, [loadOrchestrations]);

  const updateSkill = useCallback(
    async (id: string, enabled: boolean): Promise<boolean> => {
      setStatus({ tone: "loading", text: enabled ? "技能启用中" : "技能停用中" });
      try {
        const payload = await requestBridge<SkillPayload>("SKILL_UPDATE", { id, enabled });
        setSkillList((current) => current?.map((item) => (item.id === payload.id ? payload : item)) ?? [payload]);
        setStatus({ tone: "success", text: enabled ? "技能已启用" : "技能已停用" });
        return true;
      } catch (error) {
        setStatus({ tone: "error", text: errorMessage(error) });
        return false;
      }
    },
    [requestBridge],
  );

  const deleteSkill = useCallback(
    async (id: string): Promise<boolean> => {
      setStatus({ tone: "loading", text: "技能删除中" });
      try {
        await requestBridge<unknown>("SKILL_DELETE", { id });
        setSkillList((current) => current?.filter((item) => item.id !== id));
        setStatus({ tone: "success", text: "技能已删除" });
        return true;
      } catch (error) {
        setStatus({ tone: "error", text: errorMessage(error) });
        return false;
      }
    },
    [requestBridge],
  );

  const setTaskEnabled = useCallback(
    async (id: string, enabled: boolean): Promise<boolean> => {
      setStatus({ tone: "loading", text: enabled ? "任务启用中" : "任务停用中" });
      try {
        const payload = await requestBridge<TaskPayload>("TASK_UPDATE", { id, scope: "user", enabled });
        if (isUserTask(payload)) {
          setTaskList((current) => upsertById(current, payload));
        } else {
          setTaskList((current) => current?.map((item) => (item.id === id ? { ...item, enabled } : item)));
        }
        setStatus({ tone: "success", text: enabled ? "任务已启用" : "任务已停用" });
        return true;
      } catch (error) {
        setStatus({ tone: "error", text: errorMessage(error) });
        return false;
      }
    },
    [requestBridge],
  );

  const setOrchestrationEnabled = useCallback(
    async (id: string, enabled: boolean): Promise<boolean> => {
      setStatus({ tone: "loading", text: enabled ? "编排启用中" : "编排停用中" });
      try {
        const payload = await requestBridge<OrchestrationTaskPayload>("TASK_UPDATE", {
          id,
          scope: "orchestration",
          enabled,
        });
        if (isOrchestrationTask(payload)) {
          setOrchestrationList((current) => upsertById(current, payload));
        } else {
          setOrchestrationList((current) => current?.map((item) => (item.id === id ? { ...item, enabled } : item)));
        }
        setStatus({ tone: "success", text: enabled ? "编排已启用" : "编排已停用" });
        return true;
      } catch (error) {
        setStatus({ tone: "error", text: errorMessage(error) });
        return false;
      }
    },
    [requestBridge],
  );

  const runTaskNow = useCallback(
    async (id: string): Promise<boolean> => {
      if (runningTaskId) {
        return false;
      }
      setRunningTaskId(id);
      setStatus({ tone: "loading", text: "任务启动中" });
      try {
        await requestBridge<unknown>("TASK_RUN_NOW", { id, scope: "user", start_only: true });
        setStatus({ tone: "success", text: "任务已启动" });
        return true;
      } catch (error) {
        setStatus({ tone: "error", text: errorMessage(error) });
        return false;
      } finally {
        setRunningTaskId("");
      }
    },
    [requestBridge, runningTaskId],
  );

  const deleteTask = useCallback(
    async (id: string): Promise<boolean> => {
      setStatus({ tone: "loading", text: "任务删除中" });
      try {
        await requestBridge<unknown>("TASK_DELETE", { id, scope: "user" });
        setTaskList((current) => current?.filter((item) => item.id !== id));
        setStatus({ tone: "success", text: "任务已删除" });
        return true;
      } catch (error) {
        setStatus({ tone: "error", text: errorMessage(error) });
        return false;
      }
    },
    [requestBridge],
  );

  const deleteOrchestration = useCallback(
    async (id: string): Promise<boolean> => {
      setStatus({ tone: "loading", text: "编排删除中" });
      try {
        await requestBridge<unknown>("TASK_DELETE", { id, scope: "orchestration" });
        setOrchestrationList((current) => current?.filter((item) => item.id !== id));
        setStatus({ tone: "success", text: "编排已删除" });
        return true;
      } catch (error) {
        setStatus({ tone: "error", text: errorMessage(error) });
        return false;
      }
    },
    [requestBridge],
  );

  const createProvider = useCallback(
    async (provider: ProviderConfigInputPayload): Promise<boolean> => {
      setStatus({ tone: "loading", text: "本地供应商保存中" });
      const payload = await createOrUpdateLocalProvider(provider);
      setLocalProviderList(payload);
      setStatus({ tone: "success", text: "本地供应商已新增" });
      return true;
    },
    [],
  );

  const updateProvider = useCallback(
    async (name: string, provider: ProviderConfigInputPayload): Promise<boolean> => {
      const existingProvider = mergedProviderList?.providers.find((item) => stringsEqualIgnoreCase(item.name, name));
      setStatus({ tone: "loading", text: "本地供应商保存中" });
      const payload = await createOrUpdateLocalProvider({
        ...provider,
        provider_id: provider.provider_id ?? existingProvider?.provider_id,
      });
      setLocalProviderList(payload);
      setStatus({ tone: "success", text: "本地供应商已保存" });
      return true;
    },
    [mergedProviderList?.providers],
  );

  const deleteProvider = useCallback(
    async (name: string): Promise<boolean> => {
      setStatus({ tone: "loading", text: "本地供应商删除中" });
      const provider = mergedProviderList?.providers.find((item) => stringsEqualIgnoreCase(item.name, name));
      if (!provider) {
        return false;
      }
      const hasLocalProvider = localProviderList?.providers.some((item) => item.provider_id === provider.provider_id);
      const payload = hasLocalProvider
        ? await deleteStoredLocalProvider(provider.provider_id)
        : await createOrUpdateLocalProvider({
          base_url: provider.base_url,
          deleted_at: new Date().toISOString(),
          model_context_window_tokens: provider.model_context_window_tokens,
          model_response_reserve_tokens: provider.model_response_reserve_tokens,
          models: provider.models,
          name: provider.name,
          provider_id: provider.provider_id,
          response_reserve_tokens: provider.response_reserve_tokens,
          type: provider.type,
        });
      setLocalProviderList(payload);
      setStatus({ tone: "success", text: "本地供应商已删除" });
      return true;
    },
    [localProviderList?.providers, mergedProviderList?.providers],
  );

  const activateProvider = useCallback(
    async (name: string): Promise<boolean> => {
      const trimmed = name.trim();
      if (!trimmed) {
        return false;
      }
      const activeProvider = mergedProviderList?.providers.find((provider) => stringsEqualIgnoreCase(provider.name, trimmed));
      if (!activeProvider) {
        return false;
      }
      setSettings((current) => ({
        ...current,
        localModel: current.localModel?.trim() || activeProvider.models?.[0]?.trim() || "",
        localProviderId: activeProvider.provider_id,
      }));
      setStatus({ tone: "success", text: "本地供应商已激活" });
      return true;
    },
    [mergedProviderList?.providers],
  );

  const refreshSessions = useCallback(async (): Promise<SessionMetadata[]> => {
    const payload = parseSessionMetadataList(await requestBridge<unknown>("SESSIONS_LIST", {}));
    setSessions(payload);
    setSessionsLoaded(true);
    return payload;
  }, [requestBridge]);

  const searchSessions = useCallback(
    async (query: string): Promise<SessionMetadata[]> => {
      const payload = parseSessionMetadataList(
        await requestBridge<unknown>("SESSIONS_SEARCH", { query: query.trim() }),
      );
      return payload;
    },
    [requestBridge],
  );

  const refreshSessionsInBackground = useCallback((): void => {
    void refreshSessions().catch((error: unknown) => {
      console.error("[useMobileBridge] refresh sessions failed", error);
    });
  }, [refreshSessions]);

  const refreshSessionsAfterConnect = useCallback((): void => {
    void refreshSessions().catch((error: unknown) => {
      const text = `会话列表加载失败：${errorMessage(error)}`;
      console.error("[useMobileBridge] refresh sessions after connect failed", error);
      setStatus({ tone: "error", text });
    });
  }, [refreshSessions]);

  const getSession = useCallback(
    async (sessionId: string): Promise<SessionDetail> => {
      return parseSessionDetail(
        await requestBridge<unknown>("SESSION_GET", { id: sessionId.trim(), limit: SESSION_DETAIL_PAGE_LIMIT }),
      );
    },
    [requestBridge],
  );

  const getFullSession = useCallback(
    async (sessionId: string): Promise<SessionDetail> => {
      const id = sessionId.trim();
      const messagesByIndex = new Map<number, SessionDetail["messages"][number]>();
      let before: number | null | undefined;
      let merged: SessionDetail | undefined;

      for (;;) {
        const detail = parseSessionDetail(
          await requestBridge<unknown>("SESSION_GET", {
            id,
            limit: SESSION_FULL_PAGE_LIMIT,
            ...(before === undefined || before === null ? {} : { before }),
          }),
        );
        merged = detail;
        for (const message of detail.messages) {
          messagesByIndex.set(message.index, message);
        }

        if (!detail.page.has_more_before) {
          break;
        }
        before = detail.page.next_before;
        if (before === undefined || before === null) {
          throw new Error("SESSION_GET page.next_before is required when has_more_before is true");
        }
      }

      if (!merged) {
        throw new Error("SESSION_GET returned no pages");
      }

      return {
        ...merged,
        messages: [...messagesByIndex.values()].sort((left, right) => left.index - right.index),
        page: {
          ...merged.page,
          before: undefined,
          end_index: messagesByIndex.size > 0 ? Math.max(...messagesByIndex.keys()) : null,
          has_more_before: false,
          limit: SESSION_FULL_PAGE_LIMIT,
          next_before: null,
          start_index: messagesByIndex.size > 0 ? Math.min(...messagesByIndex.keys()) : null,
        },
      };
    },
    [requestBridge],
  );

  const appendSessionMessages = useCallback(
    async (input: AppendSessionMessagesInput): Promise<AppendSessionMessagesResult> => {
      const payload = await requestBridge<{
        message_count: number;
        status: "appended" | "conflict";
        updated_at: string;
      }>("SESSION_APPEND", {
        expected_head: input.expectedHead,
        messages: input.messages.map((message) => ({
          role: message.role,
          text: message.text,
        })),
        session_id: input.sessionId,
        title: input.title,
      });
      if (payload.status === "appended") {
        refreshSessionsInBackground();
      }
      return {
        messageCount: payload.message_count,
        status: payload.status,
        updatedAt: payload.updated_at,
      };
    },
    [refreshSessionsInBackground, requestBridge],
  );

  const connectBridge = useCallback(async (): Promise<void> => {
    setConnectionStatus({ tone: "loading", text: "连接中" });
    try {
      if (settings.connectionMode === "webrtc") {
        if (!settings.pairing) {
          throw new Error("请先导入 WebRTC 配对 URI");
        }
        const secret = await loadMobileCredential(settings.pairing.deviceId);
        webRTCClientRef.current?.close();
        const client = new MobileWebRTCBridge(settings.pairing, secret);
        await client.connect();
        webRTCClientRef.current = client;
      }
      await refreshRuntimeConfig();
      connectedTargetRef.current = currentConnectionTarget;
      setSettings((current) => ({
        ...current,
        lastSuccessfulConnection: connectionSnapshotFromSettings(settings, bridgeUrl),
      }));
      setConnectionStatus({ tone: "success", text: connectedStatusText(settings.connectionMode) });
      refreshSkillsAfterConnect();
      refreshTasksAfterConnect();
      refreshOrchestrationsAfterConnect();
      refreshSessionsAfterConnect();
    } catch (error) {
      connectedTargetRef.current = undefined;
      webRTCClientRef.current?.close();
      webRTCClientRef.current = undefined;
      setConfig(undefined);
      setProviderList(undefined);
      setSkillList(undefined);
      setSkillListError("");
      setTaskList(undefined);
      setTaskListError("");
      setOrchestrationList(undefined);
      setOrchestrationListError("");
      setRunningTaskId("");
      setSessions([]);
      setSessionsLoaded(false);
      setConnectionStatus({ tone: "error", text: errorMessage(error) });
    }
  }, [
    currentConnectionTarget,
    bridgeUrl,
    refreshSkillsAfterConnect,
    refreshTasksAfterConnect,
    refreshOrchestrationsAfterConnect,
    refreshRuntimeConfig,
    refreshSessionsAfterConnect,
    settings,
  ]);

  useEffect(() => {
    const pendingTarget = pendingAutoConnectTargetRef.current;
    if (pendingTarget) {
      if (currentConnectionTarget !== pendingTarget) {
        return;
      }
      pendingAutoConnectTargetRef.current = undefined;
      void connectBridge();
      return;
    }

    const snapshot = settings.lastSuccessfulConnection;
    if (autoConnectAttemptedRef.current || !settings.autoConnectEnabled || !snapshot) {
      return;
    }

    autoConnectAttemptedRef.current = true;
    if (connectionSettingsMatchSnapshot(settings, snapshot)) {
      void connectBridge();
      return;
    }

    const restoredSettings = settingsWithConnectionSnapshot(settings, snapshot);
    pendingAutoConnectTargetRef.current = connectionTargetKey(
      restoredSettings,
      normalizeBridgeUrl(restoredSettings.bridgeUrl),
    );
    setSettings(restoredSettings);
  }, [connectBridge, currentConnectionTarget, settings]);

  const switchModel = useCallback(
    async (model: string): Promise<boolean> => {
      const trimmed = model.trim();
      if (!trimmed) {
        return false;
      }
      if (!settings.remoteExecutionEnabled) {
        setSettings((current) => ({ ...current, localModel: trimmed }));
        setStatus({ tone: "success", text: "本地模型已切换" });
        return true;
      }
      if (config?.model === trimmed) {
        return true;
      }

      setStatus({ tone: "loading", text: "模型切换中" });
      try {
        const payload = await requestBridge<ConfigPayload>("CONFIG_UPDATE", { model: trimmed });
        setConfig(payload);
        await loadProviders();
        setStatus({ tone: "success", text: "模型已切换" });
        return true;
      } catch (error) {
        setStatus({ tone: "error", text: errorMessage(error) });
        return false;
      }
    },
    [config?.model, loadProviders, requestBridge, settings.remoteExecutionEnabled],
  );

  const sendAgentMessage = useCallback(
    async (options: SendAgentMessageOptions): Promise<SendAgentMessageResult> => {
      const traceId = options.traceId?.trim() || createTraceId("android-agent-stream");
      const requestId = options.requestId?.trim() || createTraceId("android-agent-stream-request");
      const initialSessionId = options.sessionId?.trim() || "";
      if (!settings.remoteExecutionEnabled) {
        const localProvider = localProviderList?.providers.find((provider) => provider.provider_id === settings.localProviderId)
          ?? localProviderList?.providers.find((provider) => !provider.deleted_at);
        const model = settings.localModel?.trim() || localProvider?.models?.[0]?.trim() || "";
        if (!localProvider || !model) {
          options.onStatus({ tone: "error", text: "请先配置本地 provider 和模型" });
          return { ok: false };
        }
        options.onStatus({ tone: "loading", text: "本地生成中" });
        try {
          const history = options.history.map((message) => ({
            role: message.role,
            text: message.text,
          }));
          const response = await sendLocalLLMMessage({
            history,
            model,
            provider: localProvider,
            sessionId: initialSessionId || `mobile-${requestId}`,
            traceId,
          });
          const sessionId = initialSessionId || `mobile-${requestId}`;
          options.onSessionId(sessionId);
          options.onReply({
            message: response.message,
            session_ended: false,
            session_id: sessionId,
          });
          options.onStatus({ tone: "success", text: "回复已返回" });
          return {
            mode: "local",
            ok: true,
            reply: {
              message: response.message,
              session_ended: false,
              session_id: sessionId,
            },
            sessionId,
          };
        } catch (error) {
          options.onStatus({ tone: "error", text: errorMessage(error) });
          return { ok: false };
        }
      }
      const runtime = createMobileAgentStreamRuntime(initialSessionId);
      const params: Record<string, unknown> = {
        message: options.message,
        ...(initialSessionId ? { session_id: initialSessionId } : {}),
      };
      if (config?.provider && config?.model) {
        params.runtime_overrides = {
          model: config.model,
          provider_name: config.provider,
        };
      }
      const projector: MobileAgentStreamProjector = {
        commitReply: (nextRuntime) => {
          options.onReply(createAgentPayloadFromRuntime(nextRuntime));
        },
        commitSessionId: (sessionId) => {
          const trimmedSessionId = sessionId.trim();
          if (!trimmedSessionId) {
            return;
          }
          options.onSessionId(trimmedSessionId);
          refreshSessionsInBackground();
        },
        setStatus: options.onStatus,
      };

      options.onReply(createAgentPayloadFromRuntime(runtime));
      options.onStatus({ tone: "loading", text: "发送中" });
      try {
        const applyEvent = (event: AgentStreamEvent) => {
          projectMobileAgentStreamEvent(event, runtime, projector);
        };
        const result = settings.connectionMode === "http"
          ? await streamAgentMessageHTTP({
            apiToken: apiToken.trim() || undefined,
            baseUrl: bridgeUrl,
            message: options.message,
            onEvent: applyEvent,
            requestId,
            sessionId: initialSessionId,
            traceId,
          })
          : await streamAgentMessageWebRTC(resolveConnectedWebRTCClient(webRTCClientRef.current), params, traceId, applyEvent);
        const resolvedSessionId = result.sessionId?.trim();
        if (resolvedSessionId) {
          runtime.sessionId = resolvedSessionId;
          projector.commitSessionId(resolvedSessionId);
          projector.commitReply(runtime);
        }
        if (!result.awaitingHuman) {
          options.onStatus({ tone: "success", text: result.sessionEnded ? "会话已结束" : "回复已返回" });
        }
        return {
          mode: "remote",
          ok: true,
          reply: createAgentPayloadFromRuntime(runtime),
          sessionId: runtime.sessionId || resolvedSessionId,
        };
      } catch (error) {
        options.onStatus({ tone: "error", text: errorMessage(error) });
        return { ok: false };
      }
    },
    [
      apiToken,
      bridgeUrl,
      config?.model,
      config?.provider,
      localProviderList?.providers,
      refreshSessionsInBackground,
      settings.connectionMode,
      settings.localModel,
      settings.localProviderId,
      settings.remoteExecutionEnabled,
    ],
  );

  const stopAgentRun = useCallback(
    async (input: StopAgentRunInput): Promise<StopAgentRunResult> => {
      const sessionId = input.sessionId?.trim() || "";
      const traceId = input.traceId?.trim() || "";
      if (!sessionId && !traceId) {
        return { ok: false };
      }

      setStatus({ tone: "loading", text: "停止中" });
      try {
        const payload = await requestBridge<AgentStopPayload>("AGENT_STOP", {
          ...(sessionId ? { session_id: sessionId } : {}),
          ...(traceId ? { trace_id: traceId } : {}),
        });
        refreshSessionsInBackground();
        setStatus({
          tone: "success",
          text: payload.status === "not_running" ? "没有运行中的任务" : "已停止",
        });
        return {
          ok: true,
          sessionId: payload.session_id?.trim() || sessionId,
          status: payload.status,
        };
      } catch (error) {
        setStatus({ tone: "error", text: errorMessage(error) });
        return { ok: false };
      }
    },
    [refreshSessionsInBackground, requestBridge],
  );

  return {
    activateProvider,
    appendSessionMessages,
    bridgeUrl,
    config,
    connectBridge,
    connectionStatus,
    createProvider,
    deleteOrchestration,
    deleteProvider,
    deleteTask,
    getFullSession,
    getSession,
    host,
    providerList: mergedProviderList,
    localProviderList: selectedLocalProviderList,
    orchestrationList,
    orchestrationListError,
    refreshProviders,
    refreshLocalProviders,
    refreshOrchestrations,
    refreshSkills,
    refreshTasks,
    runTaskNow,
    runningTaskId,
    searchSessions,
    sendAgentMessage,
    sessions,
    sessionsLoaded,
    setSettings,
    setStatus,
    setOrchestrationEnabled,
    setTaskEnabled,
    settings,
    skillList,
    skillListError,
    stopAgentRun,
    switchModel,
    taskList,
    taskListError,
    status,
    updateSkill,
    updateProvider,
    deleteSkill,
  };
}
