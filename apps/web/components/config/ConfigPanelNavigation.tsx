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

interface TabDefinition {
  id: SettingsTab;
  group: 'system' | 'prompts';
}

const SettingsIcon = ({ size = 16 }: { size?: number }) => (
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

const tabs: TabDefinition[] = [
  { id: 'general', group: 'system' },
  { id: 'provider', group: 'system' },
  { id: 'relay', group: 'system' },
  { id: 'tasks', group: 'system' },
  { id: 'orchestration', group: 'system' },
  { id: 'skills', group: 'system' },
  { id: 'tools', group: 'system' },
  { id: 'presets', group: 'system' },
  { id: 'prompts_library', group: 'prompts' },
  { id: 'prompts_preview', group: 'prompts' },
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
