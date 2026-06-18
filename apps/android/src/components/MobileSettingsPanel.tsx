import { useState } from "react";
import type { ComponentType, ReactNode } from "react";
import { Archive, ArrowLeft, Globe, Info, Megaphone } from "lucide-react";

interface MobileSettingsPanelProps {
  open: boolean;
  onClose: () => void;
}

export function MobileSettingsPanel(props: MobileSettingsPanelProps) {
  const [improveModelEnabled, setImproveModelEnabled] = useState(true);

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
          <SettingsSection title="数据控制">
            <div className="mobile-settings-card">
              <div className="mobile-settings-model-row">
                <div className="mobile-settings-item-title">
                  <Archive className="mobile-settings-icon" aria-hidden="true" strokeWidth={1.5} />
                  <span>为所有用户改进模型</span>
                </div>
                <button
                  className={`mobile-settings-switch ${improveModelEnabled ? "is-on" : ""}`}
                  type="button"
                  role="switch"
                  aria-checked={improveModelEnabled}
                  aria-label="为所有用户改进模型"
                  onClick={() => setImproveModelEnabled((current) => !current)}
                >
                  <span />
                </button>
              </div>
              <p className="mobile-settings-description">
                允许我们将你的内容用于改善你和其他用户的模型使用体验。我们将采取措施保护你的隐私。
                <a href="#learn-more">了解更多。</a>
              </p>

              <CardDivider />

              <SettingsButton icon={Megaphone} label="广告控制" />
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
