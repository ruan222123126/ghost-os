import type { ComponentType, ReactNode } from "react";

type SettingsIconComponent = ComponentType<{
  className?: string;
  "aria-hidden"?: true;
  strokeWidth?: number;
}>;

export function SettingsSection(props: { title: string; children: ReactNode }) {
  return (
    <section className="mobile-settings-section">
      <h2>{props.title}</h2>
      {props.children}
    </section>
  );
}

export function SettingsButton(props: {
  disabled?: boolean;
  icon: SettingsIconComponent;
  label: string;
  onClick?: () => void;
  sublabel?: string;
}) {
  const Icon = props.icon;

  return (
    <button className="mobile-settings-item-button" type="button" disabled={props.disabled} onClick={props.onClick}>
      <Icon className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.5} />
      <span className="mobile-settings-item-copy">
        <span>{props.label}</span>
        {props.sublabel ? <small>{props.sublabel}</small> : null}
      </span>
    </button>
  );
}

export function SettingsSwitchRow(props: {
  checked: boolean;
  disabled?: boolean;
  icon: SettingsIconComponent;
  label: string;
  onChange: (checked: boolean) => void;
  sublabel?: string;
}) {
  const Icon = props.icon;

  return (
    <button
      className="mobile-settings-auto-connect-row"
      type="button"
      role="switch"
      aria-checked={props.checked}
      disabled={props.disabled}
      onClick={() => props.onChange(!props.checked)}
    >
      <span className="mobile-settings-auto-connect-copy">
        <Icon className="mobile-settings-icon" aria-hidden={true} strokeWidth={1.5} />
        <span>
          <span>{props.label}</span>
          {props.sublabel ? <small>{props.sublabel}</small> : null}
        </span>
      </span>
      <span className="mobile-settings-toggle-track" aria-hidden={true}>
        <span className="mobile-settings-toggle-thumb" />
      </span>
    </button>
  );
}

export function SettingsStaticRow(props: {
  icon: SettingsIconComponent;
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
