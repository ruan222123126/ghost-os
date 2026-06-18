import { invoke } from "@tauri-apps/api/core";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createTraceId, errorMessage, hasTauriRuntime } from "../lib/bridgeBus";
import type { BridgeBusCommand, BridgeEnvelope } from "../lib/bridgeBus";
import { loadMobileCredential } from "../lib/mobileCredentials";
import { MobileWebRTCBridge } from "../lib/mobileWebRTC";
import { loadSettings, normalizeBridgeUrl, saveSettings } from "../lib/settingsStorage";
import type { AgentPayload, ConfigPayload, HostProfile, StatusMessage, StoredSettings } from "../mobileTypes";

interface SendAgentMessageOptions {
  message: string;
}

export function useMobileBridge() {
  const [settings, setSettings] = useState<StoredSettings>(() => loadSettings());
  const [apiToken] = useState("");
  const [host, setHost] = useState<HostProfile>();
  const [config, setConfig] = useState<ConfigPayload>();
  const [reply, setReply] = useState<AgentPayload>();
  const [status, setStatus] = useState<StatusMessage>({
    tone: "idle",
    text: "未连接",
  });
  const webRTCClientRef = useRef<MobileWebRTCBridge | undefined>(undefined);

  const bridgeUrl = useMemo(() => normalizeBridgeUrl(settings.bridgeUrl), [settings.bridgeUrl]);

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

  const connectBridge = useCallback(async (): Promise<void> => {
    setStatus({ tone: "loading", text: "连接中" });
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
      const payload = await requestBridge<ConfigPayload>("CONFIG_GET", {});
      setConfig(payload);
      setStatus({
        tone: "success",
        text: settings.connectionMode === "webrtc" ? "WebRTC 已连接" : "HTTP fallback 已连接",
      });
    } catch (error) {
      webRTCClientRef.current?.close();
      webRTCClientRef.current = undefined;
      setConfig(undefined);
      setStatus({ tone: "error", text: errorMessage(error) });
    }
  }, [requestBridge, settings.connectionMode, settings.pairing]);

  const sendAgentMessage = useCallback(
    async (options: SendAgentMessageOptions): Promise<boolean> => {
      setStatus({ tone: "loading", text: "发送中" });
      try {
        const payload = await requestBridge<AgentPayload>("AGENT_SEND", {
          message: options.message,
          ...(settings.sessionId.trim() ? { session_id: settings.sessionId.trim() } : {}),
        });
        setReply(payload);
        setSettings((current) => ({
          ...current,
          sessionId: payload.session_id || current.sessionId,
        }));
        setStatus({ tone: "success", text: "回复已返回" });
        return true;
      } catch (error) {
        setReply(undefined);
        setStatus({ tone: "error", text: errorMessage(error) });
        return false;
      }
    },
    [requestBridge, settings.sessionId],
  );

  return {
    bridgeUrl,
    config,
    connectBridge,
    host,
    reply,
    sendAgentMessage,
    setReply,
    setSettings,
    setStatus,
    settings,
    status,
  };
}
