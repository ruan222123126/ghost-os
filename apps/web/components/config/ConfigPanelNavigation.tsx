import type { FC } from 'react';

export type SettingsTab = 'general' | 'provider' | 'tasks' | 'appearance' | 'data' | 'notifications' | 'security';

interface IconProps {
  size?: number;
}

interface TabDefinition {
  id: SettingsTab;
  label: string;
  group: 'system' | 'preferences';
  icon: FC<IconProps>;
}

export const CloseIcon: FC<IconProps> = ({ size = 18 }) => (
  <svg width={size} height={size} viewBox="0 0 20 20" fill="none" aria-hidden="true">
    <path d="M5 5 15 15" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" />
    <path d="M15 5 5 15" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" />
  </svg>
);

const SettingsIcon: FC<IconProps> = ({ size = 16 }) => (
  <svg width={size} height={size} viewBox="0 0 20 20" fill="none" aria-hidden="true">
    <path
      d="M10 3.25 11.35 2.5l1.1 1.9 1.6.4 1.55-1.15 1.4 1.4-1.15 1.55.4 1.6 1.9 1.1-.75 1.35.75 1.35-1.9 1.1-.4 1.6 1.15 1.55-1.4 1.4-1.55-1.15-1.6.4-1.1 1.9L10 16.75l-1.35.75-1.1-1.9-1.6-.4-1.55 1.15-1.4-1.4 1.15-1.55-.4-1.6-1.9-1.1.75-1.35-.75-1.35 1.9-1.1.4-1.6L3 5.05l1.4-1.4 1.55 1.15 1.6-.4 1.1-1.9L10 3.25Z"
      stroke="currentColor"
      strokeWidth="1.2"
      strokeLinejoin="round"
    />
    <circle cx="10" cy="10" r="2.6" stroke="currentColor" strokeWidth="1.3" />
  </svg>
);

const ServerIcon: FC<IconProps> = ({ size = 16 }) => (
  <svg width={size} height={size} viewBox="0 0 20 20" fill="none" aria-hidden="true">
    <rect x="3" y="4" width="14" height="4" rx="1.5" stroke="currentColor" strokeWidth="1.3" />
    <rect x="3" y="12" width="14" height="4" rx="1.5" stroke="currentColor" strokeWidth="1.3" />
    <circle cx="6" cy="6" r="0.9" fill="currentColor" />
    <circle cx="6" cy="14" r="0.9" fill="currentColor" />
  </svg>
);

const PaletteIcon: FC<IconProps> = ({ size = 16 }) => (
  <svg width={size} height={size} viewBox="0 0 20 20" fill="none" aria-hidden="true">
    <path
      d="M10 3a7 7 0 1 0 0 14h1.4a2.6 2.6 0 0 0 0-5.2h-1.2a1.8 1.8 0 1 1 0-3.6h4A2.8 2.8 0 0 0 17 5.4C15.9 3.9 13.9 3 10 3Z"
      stroke="currentColor"
      strokeWidth="1.3"
      strokeLinecap="round"
      strokeLinejoin="round"
    />
    <circle cx="6.8" cy="8" r="0.9" fill="currentColor" />
    <circle cx="9.7" cy="6.8" r="0.9" fill="currentColor" />
    <circle cx="12.7" cy="7.4" r="0.9" fill="currentColor" />
  </svg>
);

const TaskIcon: FC<IconProps> = ({ size = 16 }) => (
  <svg width={size} height={size} viewBox="0 0 20 20" fill="none" aria-hidden="true">
    <rect x="3" y="4" width="14" height="12" rx="2" stroke="currentColor" strokeWidth="1.3" />
    <path d="M7 8h6" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" />
    <path d="M7 11h6" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" />
  </svg>
);

const DatabaseIcon: FC<IconProps> = ({ size = 16 }) => (
  <svg width={size} height={size} viewBox="0 0 20 20" fill="none" aria-hidden="true">
    <ellipse cx="10" cy="5" rx="6.5" ry="2.5" stroke="currentColor" strokeWidth="1.3" />
    <path d="M3.5 5v8c0 1.4 2.9 2.5 6.5 2.5s6.5-1.1 6.5-2.5V5" stroke="currentColor" strokeWidth="1.3" />
    <path d="M3.5 9.2C3.5 10.6 6.4 11.7 10 11.7s6.5-1.1 6.5-2.5" stroke="currentColor" strokeWidth="1.3" />
  </svg>
);

const BellIcon: FC<IconProps> = ({ size = 16 }) => (
  <svg width={size} height={size} viewBox="0 0 20 20" fill="none" aria-hidden="true">
    <path d="M10 3.5a4 4 0 0 0-4 4V10l-1.5 2.6h11L14 10V7.5a4 4 0 0 0-4-4Z" stroke="currentColor" strokeWidth="1.3" strokeLinejoin="round" />
    <path d="M8.2 14.6a2 2 0 0 0 3.6 0" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" />
  </svg>
);

const ShieldIcon: FC<IconProps> = ({ size = 16 }) => (
  <svg width={size} height={size} viewBox="0 0 20 20" fill="none" aria-hidden="true">
    <path d="M10 2.8 15.8 5v4.6c0 3.2-2.1 5.7-5.8 7.6-3.7-1.9-5.8-4.4-5.8-7.6V5L10 2.8Z" stroke="currentColor" strokeWidth="1.3" strokeLinejoin="round" />
    <path d="m7.7 9.9 1.6 1.6 3.1-3.1" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" strokeLinejoin="round" />
  </svg>
);

const tabs: TabDefinition[] = [
  { id: 'general', label: 'General', group: 'system', icon: SettingsIcon },
  { id: 'provider', label: 'Provider', group: 'system', icon: ServerIcon },
  { id: 'tasks', label: 'Tasks', group: 'system', icon: TaskIcon },
  { id: 'appearance', label: 'Appearance', group: 'system', icon: PaletteIcon },
  { id: 'data', label: 'Data & Memory', group: 'preferences', icon: DatabaseIcon },
  { id: 'notifications', label: 'Notifications', group: 'preferences', icon: BellIcon },
  { id: 'security', label: 'Security', group: 'preferences', icon: ShieldIcon },
];

const systemTabs = tabs.filter((tab) => tab.group === 'system');
const preferenceTabs = tabs.filter((tab) => tab.group === 'preferences');

function Kicker(props: { children: string }) {
  return (
    <h3 className="mb-3 px-3 text-[11px] font-bold uppercase tracking-[0.15em] text-[#737373]">
      {props.children}
    </h3>
  );
}

export function SettingsNavigation(props: {
  activeTab: SettingsTab;
  onSelectTab: (tab: SettingsTab) => void;
}) {
  const { activeTab, onSelectTab } = props;

  return (
    <aside className="w-[240px] shrink-0 border-r border-[#E5E5E5] bg-[#FAFAFA]">
      <div className="px-8 pb-4 pt-8">
        <h2 id="settings-title" className="text-[18px] font-semibold tracking-tight text-[#111111]">
          Configuration
        </h2>
      </div>

      <nav className="max-h-full space-y-6 overflow-y-auto px-4 pb-8">
        <SettingsNavGroup title="System" tabs={systemTabs} activeTab={activeTab} onSelectTab={onSelectTab} />
        <SettingsNavGroup title="Preferences" tabs={preferenceTabs} activeTab={activeTab} onSelectTab={onSelectTab} />
      </nav>
    </aside>
  );
}

function SettingsNavGroup(props: {
  title: string;
  tabs: TabDefinition[];
  activeTab: SettingsTab;
  onSelectTab: (tab: SettingsTab) => void;
}) {
  const { title, tabs: groupTabs, activeTab, onSelectTab } = props;

  return (
    <div>
      <Kicker>{title}</Kicker>
      <div className="space-y-1">
        {groupTabs.map((tab) => (
          <SettingsNavItem
            key={tab.id}
            tab={tab}
            active={activeTab === tab.id}
            onSelectTab={onSelectTab}
          />
        ))}
      </div>
    </div>
  );
}

function SettingsNavItem(props: {
  tab: TabDefinition;
  active: boolean;
  onSelectTab: (tab: SettingsTab) => void;
}) {
  const { tab, active, onSelectTab } = props;
  const Icon = tab.icon;

  return (
    <button
      type="button"
      onClick={() => onSelectTab(tab.id)}
      className={`flex w-full items-center gap-3 rounded-[12px] border px-4 py-2.5 text-left text-[13px] font-medium transition-all ${
        active
          ? 'border-[#E5E5E5] bg-white text-[#111111] shadow-sm'
          : 'border-transparent text-[#737373] hover:bg-[#E5E5E5]/50 hover:text-[#111111]'
      }`}
    >
      <Icon size={16} />
      <span>{tab.label}</span>
    </button>
  );
}

export function ComingSoonPanel(props: { tab: SettingsTab }) {
  const currentTab = tabs.find((tab) => tab.id === props.tab);

  return (
    <section className="py-20 text-center">
      <div className="mb-4 inline-flex h-12 w-12 items-center justify-center rounded-full bg-[#F5F5F5] text-[#737373]">
        <SettingsIcon size={24} />
      </div>
      <h3 className="mb-1 text-[16px] font-medium text-[#111111]">{currentTab?.label ?? 'Section'}</h3>
      <p className="text-[13px] text-[#737373]">Under active development.</p>
    </section>
  );
}
