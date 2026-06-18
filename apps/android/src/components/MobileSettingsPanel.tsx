import type { ComponentType, Dispatch, FormEvent, ReactNode, SetStateAction } from "react";
import { useEffect, useState } from "react";
import { ArrowLeft, Globe, KeyRound, Link2, Radio, Server, Trash2, Wifi } from "lucide-react";
import { deleteMobileCredential, saveMobileCredential } from "../lib/mobileCredentials";
import { hasTurnServer, parsePairingUri } from "../lib/mobileWebRTC";
import type { StatusMessage, StoredSettings } from "../mobileTypes";
import "./MobileSettingsPanel.css";

interface MobileSettingsPanelProps {
  open: boolean;
  onClose: () => void;
  onConnect: () => Promise<void>;
  onSettingsChange: Dispatch<SetStateAction<StoredSettings>>;
  settings: StoredSettings;
  status: StatusMessage;
}

type SettingsView = "root" | "connection";

export function MobileSettingsPanel(props: MobileSettingsPanelProps) {
  const [view, setView] = useState<SettingsView>("root");
  const [pairingUri, setPairingUri] = useState("");
  const [bridgeUrlDraft, setBridgeUrlDraft] = useState(props.settings.bridgeUrl);
  const [pairingError, setPairingError] = useState("");
  const [pairingWarning, setPairingWarning] = useState("");

  useEffect(() => {
    if (props.open) {
      setView("root");
      setBridgeUrlDraft(props.settings.bridgeUrl);
      setPairingError("");
      setPairingWarning("");
    }
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
      props.onSettingsChange((current) => ({ ...current, pairing: undefined }));
    } catch (error) {
      setPairingError(error instanceof Error ? error.message : String(error));
    }
  }

  function setConnectionMode(mode: StoredSettings["connectionMode"]): void {
    props.onSettingsChange((current) => ({ ...current, connectionMode: mode }));
  }

  function saveBridgeURL(value: string): void {
    setBridgeUrlDraft(value);
    props.onSettingsChange((current) => ({ ...current, bridgeUrl: value }));
  }

  function handleBack(): void {
    if (view === "connection") {
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
      aria-labelledby="mobile-settings-title"
      inert={props.open ? undefined : true}
    >
      <div className="mobile-settings-frame">
        <header className="mobile-settings-header">
          <button className="mobile-settings-back" type="button" aria-label="返回" onClick={handleBack}>
            <ArrowLeft className="mobile-settings-icon" aria-hidden="true" strokeWidth={2} />
          </button>
          <h1 id="mobile-settings-title">{view === "connection" ? "连接" : "设置"}</h1>
          <span className="mobile-settings-header-spacer" aria-hidden="true" />
        </header>

        <div className="mobile-settings-body">
          {view === "connection" ? (
            <ConnectionSettings
              bridgeUrlDraft={bridgeUrlDraft}
              pairingError={pairingError}
              pairingUri={pairingUri}
              pairingWarning={pairingWarning}
              settings={props.settings}
              status={props.status}
              onConnect={props.onConnect}
              onImportPairing={importPairing}
              onPairingUriChange={setPairingUri}
              onRemovePairing={removePairing}
              onSaveBridgeURL={saveBridgeURL}
              onSetConnectionMode={setConnectionMode}
            />
          ) : (
            <SettingsRoot
              connectionSublabel={connectionSublabel(props.settings)}
              onOpenConnection={() => setView("connection")}
            />
          )}
        </div>
      </div>
    </section>
  );
}

function SettingsSection(props: { title: string; children: ReactNode }) {
  return (
    <section className="mobile-settings-section">
      <h2>{props.title}</h2>
      {props.children}
    </section>
  );
}

function SettingsButton(props: {
  icon: ComponentType<{ className?: string; "aria-hidden"?: true; strokeWidth?: number }>;
  label: string;
  onClick?: () => void;
  sublabel?: string;
}) {
  const Icon = props.icon;

  return (
    <button className="mobile-settings-item-button" type="button" onClick={props.onClick}>
      <Icon className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.5} />
      <span className="mobile-settings-item-copy">
        <span>{props.label}</span>
        {props.sublabel ? <small>{props.sublabel}</small> : null}
      </span>
    </button>
  );
}

function SettingsRoot(props: { connectionSublabel: string; onOpenConnection: () => void }) {
  return (
    <>
      <SettingsSection title="连接">
        <div className="mobile-settings-card">
          <SettingsButton icon={Link2} label="连接" sublabel={props.connectionSublabel} onClick={props.onOpenConnection} />
        </div>
      </SettingsSection>

      <SettingsSection title="应用">
        <div className="mobile-settings-card">
          <SettingsButton icon={Globe} label="语言" sublabel="中文" />
        </div>
      </SettingsSection>
    </>
  );
}

interface ConnectionSettingsProps {
  bridgeUrlDraft: string;
  pairingError: string;
  pairingUri: string;
  pairingWarning: string;
  settings: StoredSettings;
  status: StatusMessage;
  onConnect: () => Promise<void>;
  onImportPairing: (event: FormEvent<HTMLFormElement>) => Promise<void>;
  onPairingUriChange: (value: string) => void;
  onRemovePairing: () => Promise<void>;
  onSaveBridgeURL: (value: string) => void;
  onSetConnectionMode: (mode: StoredSettings["connectionMode"]) => void;
}

function ConnectionSettings(props: ConnectionSettingsProps) {
  return (
    <SettingsSection title="连接">
      <div className="mobile-settings-card">
        <div className="mobile-settings-mode-switch" role="group" aria-label="连接模式">
          <button
            className={props.settings.connectionMode === "webrtc" ? "is-active" : ""}
            type="button"
            onClick={() => props.onSetConnectionMode("webrtc")}
          >
            <Radio className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.5} />
            WebRTC
          </button>
          <button
            className={props.settings.connectionMode === "http" ? "is-active" : ""}
            type="button"
            onClick={() => props.onSetConnectionMode("http")}
          >
            <Server className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.5} />
            HTTP
          </button>
        </div>

        {props.settings.connectionMode === "webrtc" ? (
          <div className="mobile-settings-connection-panel">
            <SettingsStaticRow
              icon={Wifi}
              label="WebRTC"
              sublabel={props.settings.pairing ? props.settings.pairing.pcId : "未配对"}
            />
            <form className="mobile-settings-pair-form" onSubmit={(event) => void props.onImportPairing(event)}>
              <textarea
                value={props.pairingUri}
                rows={3}
                placeholder="ghost-os://mobile-pair?..."
                onChange={(event) => props.onPairingUriChange(event.target.value)}
              />
              <button type="submit" disabled={!props.pairingUri.trim()}>
                <KeyRound className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.5} />
                导入配对
              </button>
            </form>
            {props.pairingWarning ? <p className="mobile-settings-warning">{props.pairingWarning}</p> : null}
            {props.pairingError ? <p className="mobile-settings-error">{props.pairingError}</p> : null}
            {props.settings.pairing ? (
              <button className="mobile-settings-danger-button" type="button" onClick={() => void props.onRemovePairing()}>
                <Trash2 className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.5} />
                删除配对
              </button>
            ) : null}
          </div>
        ) : (
          <label className="mobile-settings-url-field">
            <span>Bridge URL</span>
            <input value={props.bridgeUrlDraft} onChange={(event) => props.onSaveBridgeURL(event.target.value)} />
          </label>
        )}

        <button className="mobile-settings-connect-button" type="button" onClick={() => void props.onConnect()}>
          <Link2 className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.5} />
          {props.status.tone === "loading" ? "连接中" : "连接 Bridge"}
        </button>
      </div>
    </SettingsSection>
  );
}

function connectionSublabel(settings: StoredSettings): string {
  if (settings.connectionMode === "http") {
    return "HTTP fallback";
  }
  return settings.pairing ? `WebRTC / ${settings.pairing.pcId}` : "WebRTC / 未配对";
}

function SettingsStaticRow(props: {
  icon: ComponentType<{ className?: string; "aria-hidden"?: true; strokeWidth?: number }>;
  label: string;
  sublabel?: string;
}) {
  const Icon = props.icon;

  return (
    <div className="mobile-settings-static-row">
      <Icon className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.5} />
      <span className="mobile-settings-item-copy">
        <span>{props.label}</span>
        {props.sublabel ? <small>{props.sublabel}</small> : null}
      </span>
    </div>
  );
}
