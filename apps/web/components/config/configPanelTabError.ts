import type { SettingsTab } from '@/components/config/ConfigPanelNavigation';

interface ResolveTabErrorOptions {
  activeTab: SettingsTab;
  generalError: string;
  providerError: string;
  taskError: string;
  skillError: string;
  toolError: string;
}

export function resolveConfigPanelTabError(options: ResolveTabErrorOptions): string {
  const { activeTab, generalError, providerError, taskError, skillError, toolError } = options;

  if (activeTab === 'provider') {
    return providerError;
  }
  if (activeTab === 'tasks') {
    return taskError;
  }
  if (activeTab === 'skills') {
    return skillError;
  }
  if (activeTab === 'tools') {
    return toolError;
  }
  if (activeTab === 'general') {
    return generalError;
  }
  return '';
}
