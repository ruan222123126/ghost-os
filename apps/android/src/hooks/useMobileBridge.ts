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
import { parseSessionDetail, parseSessionMetadataList } from "../lib/sessionPayloadParser";
import { MobileWebRTCBridge } from "../lib/mobileWebRTC";
import { loadSettings, normalizeBridgeUrl, saveSettings } from "../lib/settingsStorage";
import type {
  AgentPayload,
  ConfigPayload,
  HostProfile,
  ProviderConfigInputPayload,
  ProviderListPayload,
  SessionDetail,
  SessionMetadata,
  SkillPayload,
  StatusMessage,
  StoredConnectionSnapshot,
  StoredSettings,
} from "../mobileTypes";

const SESSION_DETAIL_PAGE_LIMIT = 100;
const UNSUPPORTED_SKILL_MANAGEMENT_TEXT = "电脑端不支持技能管理";

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

function firstProviderModel(providerList: ProviderListPayload | undefined, providerName: string): string | undefined {
  const provider = providerList?.providers.find((item) => stringsEqualIgnoreCase(item.name, providerName));
  return provider?.models?.map((item) => item.trim()).find(Boolean);
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

export function useMobileBridge() {
  const [settings, setSettings] = useState<StoredSettings>(() => loadSettings());
  const [host, setHost] = useState<HostProfile>();
  const [config, setConfig] = useState<ConfigPayload>();
  const [providerList, setProviderList] = useState<ProviderListPayload>();
  const [skillList, setSkillList] = useState<SkillPayload[]>();
  const [skillListError, setSkillListError] = useState("");
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

  const loadSkills = useCallback(async (): Promise<SkillPayload[]> => {
    setSkillListError("");
    const payload = await requestBridge<SkillPayload[]>("SKILL_LIST", {});
    setSkillList(payload);
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

  const refreshProviders = useCallback(async (): Promise<boolean> => {
    setStatus({ tone: "loading", text: "供应商刷新中" });
    try {
      await loadProviders();
      setStatus({ tone: "success", text: "供应商已刷新" });
      return true;
    } catch (error) {
      setStatus({ tone: "error", text: errorMessage(error) });
      return false;
    }
  }, [loadProviders]);

  const syncConfigAfterProviderWrite = useCallback(async (): Promise<void> => {
    const payload = await requestBridge<ConfigPayload>("CONFIG_GET", {});
    setConfig(payload);
  }, [requestBridge]);

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

  const createProvider = useCallback(
    async (provider: ProviderConfigInputPayload): Promise<boolean> => {
      setStatus({ tone: "loading", text: "供应商保存中" });
      try {
        const payload = await requestBridge<ProviderListPayload>("CONFIG_PROVIDER_CREATE", { ...provider });
        setProviderList(payload);
        await syncConfigAfterProviderWrite();
        setStatus({ tone: "success", text: "供应商已新增" });
        return true;
      } catch (error) {
        setStatus({ tone: "error", text: errorMessage(error) });
        return false;
      }
    },
    [requestBridge, syncConfigAfterProviderWrite],
  );

  const updateProvider = useCallback(
    async (name: string, provider: ProviderConfigInputPayload): Promise<boolean> => {
      setStatus({ tone: "loading", text: "供应商保存中" });
      try {
        const payload = await requestBridge<ProviderListPayload>("CONFIG_PROVIDER_UPDATE", { name, provider });
        setProviderList(payload);
        await syncConfigAfterProviderWrite();
        setStatus({ tone: "success", text: "供应商已保存" });
        return true;
      } catch (error) {
        setStatus({ tone: "error", text: errorMessage(error) });
        return false;
      }
    },
    [requestBridge, syncConfigAfterProviderWrite],
  );

  const deleteProvider = useCallback(
    async (name: string): Promise<boolean> => {
      setStatus({ tone: "loading", text: "供应商删除中" });
      try {
        const payload = await requestBridge<ProviderListPayload>("CONFIG_PROVIDER_DELETE", { name });
        setProviderList(payload);
        await syncConfigAfterProviderWrite();
        setStatus({ tone: "success", text: "供应商已删除" });
        return true;
      } catch (error) {
        setStatus({ tone: "error", text: errorMessage(error) });
        return false;
      }
    },
    [requestBridge, syncConfigAfterProviderWrite],
  );

  const activateProvider = useCallback(
    async (name: string): Promise<boolean> => {
      const trimmed = name.trim();
      if (!trimmed) {
        return false;
      }
      setStatus({ tone: "loading", text: "供应商切换中" });
      try {
        const params: Record<string, unknown> = { provider: trimmed };
        const firstModel = firstProviderModel(providerList, trimmed);
        if (config?.model_selection_enabled !== false && firstModel) {
          params.model = firstModel;
        }
        const configPayload = await requestBridge<ConfigPayload>("CONFIG_UPDATE", params);
        setConfig(configPayload);
        await loadProviders();
        setStatus({ tone: "success", text: "供应商已激活" });
        return true;
      } catch (error) {
        setStatus({ tone: "error", text: errorMessage(error) });
        return false;
      }
    },
    [config?.model_selection_enabled, loadProviders, providerList, requestBridge],
  );

  const refreshSessions = useCallback(async (): Promise<SessionMetadata[]> => {
    const payload = parseSessionMetadataList(await requestBridge<unknown>("SESSIONS_LIST", {}));
    setSessions(payload);
    setSessionsLoaded(true);
    return payload;
  }, [requestBridge]);

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
      refreshSessionsAfterConnect();
    } catch (error) {
      connectedTargetRef.current = undefined;
      webRTCClientRef.current?.close();
      webRTCClientRef.current = undefined;
      setConfig(undefined);
      setProviderList(undefined);
      setSkillList(undefined);
      setSkillListError("");
      setSessions([]);
      setSessionsLoaded(false);
      setConnectionStatus({ tone: "error", text: errorMessage(error) });
    }
  }, [
    currentConnectionTarget,
    bridgeUrl,
    refreshSkillsAfterConnect,
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
    [config?.model, loadProviders, requestBridge],
  );

  const sendAgentMessage = useCallback(
    async (options: SendAgentMessageOptions): Promise<SendAgentMessageResult> => {
      const traceId = createTraceId("android-agent-stream");
      const requestId = createTraceId("android-agent-stream-request");
      const initialSessionId = options.sessionId?.trim() || "";
      const runtime = createMobileAgentStreamRuntime(initialSessionId);
      const params = {
        message: options.message,
        ...(initialSessionId ? { session_id: initialSessionId } : {}),
      };
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
          ok: true,
          reply: createAgentPayloadFromRuntime(runtime),
          sessionId: runtime.sessionId || resolvedSessionId,
        };
      } catch (error) {
        options.onStatus({ tone: "error", text: errorMessage(error) });
        return { ok: false };
      }
    },
    [apiToken, bridgeUrl, refreshSessionsInBackground, settings.connectionMode],
  );

  return {
    activateProvider,
    bridgeUrl,
    config,
    connectBridge,
    connectionStatus,
    createProvider,
    deleteProvider,
    getSession,
    host,
    providerList,
    refreshProviders,
    refreshSkills,
    sendAgentMessage,
    sessions,
    sessionsLoaded,
    setSettings,
    setStatus,
    settings,
    skillList,
    skillListError,
    switchModel,
    status,
    updateSkill,
    updateProvider,
    deleteSkill,
  };
}
