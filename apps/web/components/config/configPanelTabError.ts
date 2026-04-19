import type { SettingsTab } from '@/components/config/ConfigPanelNavigation';

interface ResolveTabErrorOptions {
  activeTab: SettingsTab;
  generalError: string;
  providerError: string;
  promptError: string;
  taskError: string;
  skillError: string;
  toolError: string;
}

export function resolveConfigPanelTabError(options: ResolveTabErrorOptions): string {
  const { activeTab, generalError, providerError, promptError, taskError, skillError, toolError } = options;

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
  if (activeTab === 'prompts') {
    return promptError;
  }
  if (activeTab === 'general') {
    return generalError;
  }
  return '';
}
