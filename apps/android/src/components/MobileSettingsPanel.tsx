import type { ComponentType, ReactNode } from "react";
import { ArrowLeft, Globe, Info, Link2 } from "lucide-react";

interface MobileSettingsPanelProps {
  open: boolean;
  onClose: () => void;
}

export function MobileSettingsPanel(props: MobileSettingsPanelProps) {
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
          <button className="mobile-settings-back" type="button" aria-label="关闭设置" onClick={props.onClose}>
            <ArrowLeft className="mobile-settings-icon" aria-hidden="true" strokeWidth={2} />
          </button>
          <h1 id="mobile-settings-title">设置</h1>
          <span className="mobile-settings-header-spacer" aria-hidden="true" />
        </header>

        <div className="mobile-settings-body">
          <SettingsSection title="连接">
            <div className="mobile-settings-card">
              <SettingsButton icon={Link2} label="连接" />
            </div>
          </SettingsSection>

          <SettingsSection title="应用">
            <div className="mobile-settings-card">
              <SettingsButton icon={Globe} label="语言" sublabel="中文" />
              <CardDivider />
              <SettingsButton icon={Info} label="关于" />
            </div>
          </SettingsSection>
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
  sublabel?: string;
}) {
  const Icon = props.icon;

  return (
    <button className="mobile-settings-item-button" type="button">
      <Icon className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.5} />
      <span className="mobile-settings-item-copy">
        <span>{props.label}</span>
        {props.sublabel ? <small>{props.sublabel}</small> : null}
      </span>
    </button>
  );
}

function CardDivider() {
  return <div className="mobile-settings-divider" aria-hidden="true" />;
}
