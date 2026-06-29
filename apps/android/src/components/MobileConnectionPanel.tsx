import type { Dispatch, FormEvent, KeyboardEvent, SetStateAction } from "react";
import { useEffect, useState } from "react";
import { ArrowLeft, Database, Key, Link2, Server, Trash2, Wifi } from "lucide-react";
import { deleteMobileCredential, saveMobileCredential } from "../lib/mobileCredentials";
import { hasTurnServer, parsePairingUri } from "../lib/mobileWebRTC";
import type { StatusMessage, StoredSettings } from "../mobileTypes";
import { SettingsButton, SettingsSection, SettingsStaticRow, SettingsSwitchRow } from "./MobileSettingsControls";
import "./MobileSettingsPanel.css";

interface MobileConnectionPanelProps {
  computerSessionPersistStatus: StatusMessage;
  connectionStatus: StatusMessage;
  open: boolean;
  settings: StoredSettings;
  onClose: () => void;
  onConnect: () => Promise<void>;
  onSettingsChange: Dispatch<SetStateAction<StoredSettings>>;
}

export function MobileConnectionPanel(props: MobileConnectionPanelProps) {
  const [view, setView] = useState<"root" | "detail">("root");
  const [pairingUri, setPairingUri] = useState("");
  const [bridgeUrlDraft, setBridgeUrlDraft] = useState(props.settings.bridgeUrl);
  const [apiTokenDraft, setAPITokenDraft] = useState(props.settings.apiToken ?? "");
  const [pairingError, setPairingError] = useState("");
  const [pairingWarning, setPairingWarning] = useState("");

  useEffect(() => {
    if (!props.open) {
      return;
    }

    setBridgeUrlDraft(props.settings.bridgeUrl);
    setAPITokenDraft(props.settings.apiToken ?? "");
    setView("root");
    setPairingUri("");
    setPairingError("");
    setPairingWarning("");
  }, [props.open]);

  async function importPairing(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    setPairingError("");
    setPairingWarning("");
    try {
      const parsed = parsePairingUri(pairingUri);
      await saveMobileCredential(parsed.pairing.deviceId, parsed.secret);
      props.onSettingsChange((current) => ({
        ...current,
        connectionMode: "webrtc",
        pairing: parsed.pairing,
      }));
      setPairingUri("");
      if (!hasTurnServer(parsed.pairing)) {
        setPairingWarning("当前配对未包含 TURN，公网连接可能 ICE 失败");
      }
    } catch (error) {
      setPairingError(error instanceof Error ? error.message : String(error));
    }
  }

  async function removePairing(): Promise<void> {
    try {
      const deviceId = props.settings.pairing?.deviceId;
      if (deviceId) {
        await deleteMobileCredential(deviceId);
      }
      props.onSettingsChange((current) => ({
        ...current,
        lastSuccessfulConnection:
          current.lastSuccessfulConnection?.connectionMode === "webrtc"
          && current.lastSuccessfulConnection.pairing?.deviceId === deviceId
            ? undefined
            : current.lastSuccessfulConnection,
        pairing: undefined,
      }));
    } catch (error) {
      setPairingError(error instanceof Error ? error.message : String(error));
    }
  }

  function setAutoConnectEnabled(enabled: boolean): void {
    props.onSettingsChange((current) => ({ ...current, autoConnectEnabled: enabled }));
  }

  function setPersistComputerSessionsEnabled(enabled: boolean): void {
    props.onSettingsChange((current) => ({ ...current, persistComputerSessionsEnabled: enabled }));
  }

  function setRemoteExecutionEnabled(enabled: boolean): void {
    props.onSettingsChange((current) => ({ ...current, remoteExecutionEnabled: enabled }));
  }

  function setConnectionMode(mode: StoredSettings["connectionMode"]): void {
    props.onSettingsChange((current) => ({ ...current, connectionMode: mode }));
  }

  function saveBridgeURL(value: string): void {
    setBridgeUrlDraft(value);
    props.onSettingsChange((current) => ({ ...current, bridgeUrl: value }));
  }

  function saveAPIToken(value: string): void {
    setAPITokenDraft(value);
    props.onSettingsChange((current) => ({ ...current, apiToken: value }));
  }

  function handleBack(): void {
    if (view === "detail") {
      setView("root");
      return;
    }

    props.onClose();
  }

  return (
    <section
      className={`mobile-settings-panel ${props.open ? "is-open" : ""}`}
      role="dialog"
      aria-modal="true"
      aria-hidden={!props.open}
      aria-labelledby="mobile-connection-title"
      inert={props.open ? undefined : true}
    >
      <div className="mobile-settings-frame">
        <header className="mobile-settings-header">
          <button className="mobile-settings-back" type="button" aria-label="返回" onClick={handleBack}>
            <ArrowLeft className="mobile-settings-icon" aria-hidden="true" strokeWidth={2} />
          </button>
          <h1 id="mobile-connection-title">{view === "detail" ? "连接" : "连接设置"}</h1>
          <span className="mobile-settings-header-spacer" aria-hidden="true" />
        </header>

        <div className="mobile-settings-body">
          {view === "detail" ? (
            <ConnectionSettings
              apiTokenDraft={apiTokenDraft}
              bridgeUrlDraft={bridgeUrlDraft}
              pairingError={pairingError}
              pairingUri={pairingUri}
              pairingWarning={pairingWarning}
              connectionStatus={props.connectionStatus}
              settings={props.settings}
              onConnect={props.onConnect}
              onImportPairing={importPairing}
              onPairingUriChange={setPairingUri}
              onRemovePairing={removePairing}
              onSaveAPIToken={saveAPIToken}
              onSaveBridgeURL={saveBridgeURL}
              onSetConnectionMode={setConnectionMode}
            />
          ) : (
            <>
              <SettingsSection title="连接偏好">
                <div className="mobile-settings-card">
                  <SettingsSwitchRow
                    checked={props.settings.autoConnectEnabled}
                    icon={Wifi}
                    label="是否自动连接"
                    onChange={setAutoConnectEnabled}
                  />
                  <SettingsSwitchRow
                    checked={props.settings.persistComputerSessionsEnabled}
                    icon={Database}
                    label="是否持久化电脑会话内容"
                    sublabel={props.computerSessionPersistStatus.text}
                    onChange={setPersistComputerSessionsEnabled}
                  />
                  <SettingsSwitchRow
                    checked={props.settings.remoteExecutionEnabled}
                    disabled={props.connectionStatus.tone !== "success"}
                    icon={Server}
                    label="跟随电脑模型"
                    sublabel={remoteExecutionSublabel(
                      props.settings.remoteExecutionEnabled,
                      props.connectionStatus.tone !== "success",
                    )}
                    onChange={setRemoteExecutionEnabled}
                  />
                </div>
              </SettingsSection>

              <SettingsSection title="连接">
                <div className="mobile-settings-card">
                  <SettingsButton
                    icon={Link2}
                    label="连接"
                    sublabel={connectionSublabel(props.settings, props.connectionStatus)}
                    onClick={() => setView("detail")}
                  />
                </div>
              </SettingsSection>
            </>
          )}
        </div>
      </div>
    </section>
  );
}

interface ConnectionSettingsProps {
  apiTokenDraft: string;
  bridgeUrlDraft: string;
  connectionStatus: StatusMessage;
  pairingError: string;
  pairingUri: string;
  pairingWarning: string;
  settings: StoredSettings;
  onConnect: () => Promise<void>;
  onImportPairing: (event: FormEvent<HTMLFormElement>) => Promise<void>;
  onPairingUriChange: (value: string) => void;
  onRemovePairing: () => Promise<void>;
  onSaveAPIToken: (value: string) => void;
  onSaveBridgeURL: (value: string) => void;
  onSetConnectionMode: (mode: StoredSettings["connectionMode"]) => void;
}

function ConnectionSettings(props: ConnectionSettingsProps) {
  const isWebRTC = props.settings.connectionMode === "webrtc";
  const isConnecting = props.connectionStatus.tone === "loading";
  const isConnected = props.connectionStatus.tone === "success";
  const connectButtonText = isConnecting ? "连接中" : isConnected ? "已连接" : "连接 Bridge";

  return (
    <section className="mobile-settings-connection-shell" aria-labelledby="mobile-settings-connection-title">
      <div className="mobile-settings-connection-title" id="mobile-settings-connection-title">
        连接方式
      </div>

      <div className="mobile-settings-connection-card">
        <div className="mobile-settings-mode-switch" role="group" aria-label="连接模式">
          <button
            className={isWebRTC ? "is-active" : ""}
            type="button"
            onClick={() => props.onSetConnectionMode("webrtc")}
            aria-pressed={isWebRTC}
          >
            <WebRTCSignalIcon />
            WebRTC
          </button>
          <button
            className={!isWebRTC ? "is-active" : ""}
            type="button"
            onClick={() => props.onSetConnectionMode("http")}
            aria-pressed={!isWebRTC}
          >
            <Server className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.5} />
            HTTP
          </button>
        </div>

        <div className="mobile-settings-connection-content">
          {isWebRTC ? (
            <div className="mobile-settings-connection-panel">
              <SettingsStaticRow
                icon={Wifi}
                label="WebRTC"
                sublabel={props.settings.pairing ? props.settings.pairing.pcId : "未配对"}
              />
              <form className="mobile-settings-pair-form" onSubmit={(event) => void props.onImportPairing(event)}>
                <div className="mobile-settings-textarea-wrap">
                  <textarea
                    value={props.pairingUri}
                    rows={3}
                    aria-label="WebRTC 配对 URI"
                    placeholder="ghost-os://mobile-pair?..."
                    onChange={(event) => props.onPairingUriChange(event.target.value)}
                    onKeyDown={(event) => handleEditableBackspace(event, props.pairingUri, props.onPairingUriChange)}
                  />
                  <span className="mobile-settings-resize-mark" aria-hidden={true}>
                    <span />
                    <span />
                  </span>
                </div>
                <button type="submit" disabled={!props.pairingUri.trim()}>
                  <Key className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.5} />
                  导入配对
                </button>
              </form>
              {props.pairingWarning ? <p className="mobile-settings-warning">{props.pairingWarning}</p> : null}
              {props.pairingError ? <p className="mobile-settings-error">{props.pairingError}</p> : null}
              {props.settings.pairing ? (
                <button
                  className="mobile-settings-danger-button"
                  type="button"
                  onClick={() => void props.onRemovePairing()}
                >
                  <Trash2 className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.5} />
                  删除配对
                </button>
              ) : null}
            </div>
          ) : (
            <div className="mobile-settings-connection-panel">
              <SettingsStaticRow icon={Server} label="HTTP" sublabel={props.bridgeUrlDraft || "未配置"} />
              <label className="mobile-settings-url-field">
                <span>Bridge URL</span>
                <input
                  value={props.bridgeUrlDraft}
                  onChange={(event) => props.onSaveBridgeURL(event.target.value)}
                  onKeyDown={(event) => handleEditableBackspace(event, props.bridgeUrlDraft, props.onSaveBridgeURL)}
                />
              </label>
              <label className="mobile-settings-url-field mobile-settings-api-token-field">
                <span>API Token</span>
                <input
                  value={props.apiTokenDraft}
                  type="password"
                  autoCapitalize="none"
                  autoComplete="off"
                  spellCheck={false}
                  onChange={(event) => props.onSaveAPIToken(event.target.value)}
                  onKeyDown={(event) => handleEditableBackspace(event, props.apiTokenDraft, props.onSaveAPIToken)}
                />
              </label>
            </div>
          )}

          <ConnectionStatusMessage status={props.connectionStatus} />

          <button
            className={`mobile-settings-connect-button ${isConnected ? "is-connected" : ""}`}
            type="button"
            disabled={isConnecting}
            onClick={() => void props.onConnect()}
          >
            <Link2 className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.5} />
            {connectButtonText}
          </button>
        </div>
      </div>
    </section>
  );
}

function ConnectionStatusMessage(props: { status: StatusMessage }) {
  if (props.status.tone === "idle") {
    return null;
  }

  const role = props.status.tone === "error" ? "alert" : "status";

  return (
    <p className={`mobile-settings-connection-status is-${props.status.tone}`} role={role}>
      <span className="mobile-settings-status-dot" aria-hidden={true} />
      <span>{props.status.text}</span>
    </p>
  );
}

function WebRTCSignalIcon() {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      className="mobile-settings-icon"
      aria-hidden={true}
    >
      <circle cx="12" cy="12" r="2" />
      <path d="M16 8.5a5 5 0 0 1 0 7" />
      <path d="M8 8.5a5 5 0 0 0 0 7" />
    </svg>
  );
}

function remoteExecutionSublabel(enabled: boolean, locked: boolean): string {
  if (locked) {
    return "连接电脑后可开启";
  }
  return enabled ? "使用电脑端当前激活模型" : "可在手机端自行选择模型";
}

function connectionSublabel(settings: StoredSettings, connectionStatus: StatusMessage): string {
  if (connectionStatus.tone === "success") {
    return connectionStatus.text;
  }

  if (settings.connectionMode === "webrtc") {
    return settings.pairing ? `WebRTC / ${settings.pairing.pcId}` : "WebRTC / 未配对";
  }

  return settings.bridgeUrl.trim() || "HTTP / 未配置";
}

type EditableFieldElement = HTMLInputElement | HTMLTextAreaElement;

function handleEditableBackspace(
  event: KeyboardEvent<EditableFieldElement>,
  value: string,
  onChange: (value: string) => void,
): void {
  if (event.key !== "Backspace" || event.nativeEvent.isComposing) {
    return;
  }

  const field = event.currentTarget;
  if (field.disabled || field.readOnly) {
    return;
  }

  // Android WebView may interpret Backspace as page-back in this sheet unless the edit stays local.
  event.preventDefault();
  event.stopPropagation();

  const nextState = deleteBackward(value, field.selectionStart, field.selectionEnd);
  onChange(nextState.value);

  window.requestAnimationFrame(() => {
    if (document.activeElement !== field) {
      return;
    }
    field.setSelectionRange(nextState.caret, nextState.caret);
  });
}

function deleteBackward(value: string, selectionStart: number | null, selectionEnd: number | null) {
  const start = selectionStart ?? value.length;
  const end = selectionEnd ?? value.length;
  if (start !== end) {
    return {
      caret: start,
      value: `${value.slice(0, start)}${value.slice(end)}`,
    };
  }

  if (start === 0) {
    return {
      caret: 0,
      value,
    };
  }

  const caret = start - 1;
  return {
    caret,
    value: `${value.slice(0, caret)}${value.slice(start)}`,
  };
}
