import { Fragment } from 'react';
import { useWebLocale } from '@/lib/i18n/provider';

export type SettingsTab =
  | 'general'
  | 'provider'
  | 'tasks'
  | 'orchestration'
  | 'skills'
  | 'tools'
  | 'presets'
  | 'prompts_library'
  | 'prompts_preview';

interface TabDefinition {
  id: SettingsTab;
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
  { id: 'general' },
  { id: 'provider' },
  { id: 'tasks' },
  { id: 'orchestration' },
  { id: 'skills' },
  { id: 'tools' },
  { id: 'presets' },
  { id: 'prompts_library' },
  { id: 'prompts_preview' },
];

function Kicker(props: { children: string }) {
  return (
    <h3 className="mb-3 px-2 text-xs font-semibold text-gray-400">
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

  return (
    <aside className="flex w-64 shrink-0 flex-col overflow-y-auto border-r border-gray-100 bg-gray-50 p-6">
      <h2 id="settings-title" className="mb-6 text-xl font-bold text-gray-900">
        {copy.settings.panelTitle}
      </h2>

      <Kicker>{copy.settings.groupSystem}</Kicker>
      <nav className="flex-1 space-y-1" data-testid="settings-nav-group-system">
        {tabs.map((tab) => (
          <Fragment key={tab.id}>
            {tab.id === 'prompts_library' ? <PromptNavLabel label={copy.settings.groupPrompts} /> : null}
            <SettingsNavItem
              tab={tab}
              active={activeTab === tab.id}
              onSelectTab={onSelectTab}
            />
          </Fragment>
        ))}
      </nav>
    </aside>
  );
}

function PromptNavLabel(props: { label: string }) {
  return <Kicker>{props.label}</Kicker>;
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
      className={`block w-full rounded-lg border px-4 py-2.5 text-left text-sm font-medium transition-colors ${
        active
          ? 'border-gray-200 bg-white text-gray-900 shadow-sm'
          : 'border-transparent text-gray-600 hover:bg-gray-100 hover:text-gray-900'
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
