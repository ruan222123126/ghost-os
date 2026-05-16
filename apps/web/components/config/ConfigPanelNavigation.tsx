import type { FC } from 'react';
import { useWebLocale } from '@/lib/i18n/provider';

export type SettingsTab =
  | 'general'
  | 'provider'
  | 'relay'
  | 'tasks'
  | 'orchestration'
  | 'skills'
  | 'tools'
  | 'presets'
  | 'prompts_library'
  | 'prompts_preview';

interface IconProps {
  size?: number;
}

interface TabDefinition {
  id: SettingsTab;
  group: 'system' | 'prompts';
  icon: FC<IconProps>;
}

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

const TaskIcon: FC<IconProps> = ({ size = 16 }) => (
  <svg width={size} height={size} viewBox="0 0 20 20" fill="none" aria-hidden="true">
    <rect x="3" y="4" width="14" height="12" rx="2" stroke="currentColor" strokeWidth="1.3" />
    <path d="M7 8h6" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" />
    <path d="M7 11h6" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" />
  </svg>
);

const RelayIcon: FC<IconProps> = ({ size = 16 }) => (
  <svg width={size} height={size} viewBox="0 0 20 20" fill="none" aria-hidden="true">
    <path d="M5 8.2a5 5 0 0 1 8.4-2.8L15 7" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" strokeLinejoin="round" />
    <path d="M15 4.2V7h-2.8" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" strokeLinejoin="round" />
    <path d="M15 11.8a5 5 0 0 1-8.4 2.8L5 13" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" strokeLinejoin="round" />
    <path d="M5 15.8V13h2.8" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" strokeLinejoin="round" />
  </svg>
);

const OrchestrationIcon: FC<IconProps> = ({ size = 16 }) => (
  <svg width={size} height={size} viewBox="0 0 20 20" fill="none" aria-hidden="true">
    <rect x="3" y="4" width="5" height="5" rx="1.2" stroke="currentColor" strokeWidth="1.3" />
    <rect x="12" y="4" width="5" height="5" rx="1.2" stroke="currentColor" strokeWidth="1.3" />
    <rect x="7.5" y="12" width="5" height="4" rx="1.2" stroke="currentColor" strokeWidth="1.3" />
    <path d="M10 9v3M8 12h4" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" />
  </svg>
);

const SkillIcon: FC<IconProps> = ({ size = 16 }) => (
  <svg width={size} height={size} viewBox="0 0 20 20" fill="none" aria-hidden="true">
    <path d="M10 3.2 11.5 7l3.8 1.5-3.8 1.5-1.5 3.8-1.5-3.8L4.7 8.5 8.5 7 10 3.2Z" stroke="currentColor" strokeWidth="1.3" strokeLinejoin="round" />
    <path d="M14.8 12.8 15.5 14.5l1.7.7-1.7.7-.7 1.7-.7-1.7-1.7-.7 1.7-.7.7-1.7Z" fill="currentColor" />
  </svg>
);

const ToolIcon: FC<IconProps> = ({ size = 16 }) => (
  <svg width={size} height={size} viewBox="0 0 20 20" fill="none" aria-hidden="true">
    <path d="m12.4 3.7 3.9 3.9-3 3-3.9-3.9 3-3Z" stroke="currentColor" strokeWidth="1.3" strokeLinejoin="round" />
    <path d="m9.8 8.3-5.3 5.3a1.8 1.8 0 1 0 2.6 2.6l5.3-5.3" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" />
  </svg>
);

const PromptIcon: FC<IconProps> = ({ size = 16 }) => (
  <svg width={size} height={size} viewBox="0 0 20 20" fill="none" aria-hidden="true">
    <path d="M4 4.5h12v8H8.4L4 16v-3.5H4z" stroke="currentColor" strokeWidth="1.3" strokeLinejoin="round" />
    <path d="M7 7.2h6M7 9.8h4" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" />
  </svg>
);

const tabs: TabDefinition[] = [
  { id: 'general', group: 'system', icon: SettingsIcon },
  { id: 'provider', group: 'system', icon: ServerIcon },
  { id: 'relay', group: 'system', icon: RelayIcon },
  { id: 'tasks', group: 'system', icon: TaskIcon },
  { id: 'orchestration', group: 'system', icon: OrchestrationIcon },
  { id: 'skills', group: 'system', icon: SkillIcon },
  { id: 'tools', group: 'system', icon: ToolIcon },
  { id: 'presets', group: 'system', icon: PromptIcon },
  { id: 'prompts_library', group: 'prompts', icon: PromptIcon },
  { id: 'prompts_preview', group: 'prompts', icon: PromptIcon },
];

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
  const { copy } = useWebLocale();
  const { activeTab, onSelectTab } = props;
  const systemTabs = tabs.filter((tab) => tab.group === 'system');
  const promptsTabs = tabs.filter((tab) => tab.group === 'prompts');

  return (
    <aside className="w-[240px] shrink-0 border-r border-[#E5E5E5] bg-[#FAFAFA]">
      <div className="px-8 pb-4 pt-8">
        <h2 id="settings-title" className="text-[18px] font-semibold tracking-tight text-[#111111]">
          {copy.settings.panelTitle}
        </h2>
      </div>

      <nav className="max-h-full space-y-6 overflow-y-auto overscroll-contain touch-pan-y px-4 pb-8">
        <SettingsNavGroup
          groupID="system"
          title={copy.settings.groupSystem}
          tabs={systemTabs}
          activeTab={activeTab}
          onSelectTab={onSelectTab}
        />
        <SettingsNavGroup
          groupID="prompts"
          title={copy.settings.groupPrompts}
          tabs={promptsTabs}
          activeTab={activeTab}
          onSelectTab={onSelectTab}
        />
      </nav>
    </aside>
  );
}

function SettingsNavGroup(props: {
  groupID: TabDefinition['group'];
  title: string;
  tabs: TabDefinition[];
  activeTab: SettingsTab;
  onSelectTab: (tab: SettingsTab) => void;
}) {
  const { groupID, title, tabs: groupTabs, activeTab, onSelectTab } = props;

  return (
    <div data-testid={`settings-nav-group-${groupID}`}>
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
  const { copy } = useWebLocale();
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
      <span>{labelForTab(copy, tab.id)}</span>
    </button>
  );
}

export function ComingSoonPanel(props: { tab: SettingsTab }) {
  const { copy } = useWebLocale();

  return (
    <section className="py-20 text-center">
      <div className="mb-4 inline-flex h-12 w-12 items-center justify-center rounded-full bg-[#F5F5F5] text-[#737373]">
        <SettingsIcon size={24} />
      </div>
      <h3 className="mb-1 text-[16px] font-medium text-[#111111]">{labelForTab(copy, props.tab) || copy.settings.comingSoonSection}</h3>
      <p className="text-[13px] text-[#737373]">{copy.settings.comingSoonDescription}</p>
    </section>
  );
}

function labelForTab(copy: ReturnType<typeof useWebLocale>['copy'], tab: SettingsTab): string {
  if (tab === 'general') {
    return copy.settings.tabGeneral;
  }
  if (tab === 'provider') {
    return copy.settings.tabProvider;
  }
  if (tab === 'relay') {
    return copy.settings.tabRelay;
  }
  if (tab === 'tasks') {
    return copy.settings.tabTasks;
  }
  if (tab === 'orchestration') {
    return copy.settings.tabOrchestration;
  }
  if (tab === 'skills') {
    return copy.settings.tabSkills;
  }
  if (tab === 'tools') {
    return copy.settings.tabTools;
  }
  if (tab === 'presets') {
    return copy.settings.tabPresets;
  }
  if (tab === 'prompts_library') {
    return copy.settings.tabPromptsLibrary;
  }
  if (tab === 'prompts_preview') {
    return copy.settings.tabPromptsPreview;
  }
  return copy.settings.tabGeneral;
}
