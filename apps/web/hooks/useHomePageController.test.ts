import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { getSession } from '@/lib/api/sessions/api';
import { getCodexModelCatalog } from '@/lib/api/agent/models';
import type { ProviderModelOption } from '@/lib/types';
import { useHomePageController } from './useHomePageController';

let mockRouter = buildRouter();
let mockSessions = buildSessionsController();
let mockChat = buildChatController();
let mockConfig = buildConfigController();

jest.mock('next/navigation', () => ({
  useRouter: () => mockRouter,
}));

jest.mock('@/hooks/useSessions', () => ({
  useSessions: () => mockSessions,
}));

jest.mock('@/hooks/chat/useBridgeChat', () => ({
  useBridgeChat: () => mockChat,
}));

jest.mock('@/hooks/useBridgeConfig', () => ({
  useBridgeConfig: () => mockConfig,
}));

jest.mock('@/lib/api/sessions/api', () => ({
  getSession: jest.fn(),
}));

jest.mock('@/lib/api/agent/models', () => ({
  getCodexModelCatalog: jest.fn(),
}));

const CODEX_CATALOG = {
  models: ['codex-default', 'codex-fast', 'gpt-5.4'],
  default_model: 'codex-default',
};

describe('hooks/useHomePageController', () => {
  beforeEach(() => {
    mockRouter = buildRouter();
    mockSessions = buildSessionsController();
    mockChat = buildChatController();
    mockConfig = buildConfigController();
    jest.mocked(getSession).mockReset();
    jest.mocked(getCodexModelCatalog).mockReturnValue(new Promise(() => undefined));
    installWindowSearch('');
  });

  afterEach(() => {
    delete (globalThis as { window?: unknown }).window;
  });

  it('sends chat messages and refreshes sessions silently', async () => {
    const latest = renderController();

    await act(async () => {
      await latest.current.sendMessage({ images: [], message: 'hello' });
    });

    expect(mockChat.sendChatMessage).toHaveBeenCalledWith({ images: [], message: 'hello' });
    expect(mockSessions.loadSessions).toHaveBeenCalledWith({ silent: true });
  });

  it('selects sessions and only loads history when needed', () => {
    mockChat.shouldLoadSessionHistory.mockReturnValue(true);
    const latest = renderController();

    act(() => {
      latest.current.selectSession('session-2');
    });

    expect(mockSessions.setCurrentSessionId).toHaveBeenCalledWith('session-2');
    expect(mockChat.clearBackgroundCompletion).toHaveBeenCalledWith('session-2');
    expect(mockChat.loadSessionHistory).toHaveBeenCalledWith('session-2');
  });

  it('restores runtime selection from loaded session history detail', async () => {
    jest.mocked(getCodexModelCatalog).mockResolvedValue(CODEX_CATALOG);
    mockChat.shouldLoadSessionHistory.mockReturnValue(true);
    mockChat.loadSessionHistory.mockResolvedValue({
      last_runtime_selection: {
        runtime: 'codex',
        provider: 'codex',
        provider_type: 'codex',
        model: 'gpt-5.4',
        mode: 'plan',
      },
    } as Awaited<ReturnType<typeof getSession>>);
    const latest = renderController();

    await act(async () => {
      latest.current.selectSession('session-2');
      await Promise.resolve();
    });

    expect(latest.current.agentMode).toBe('plan');
    expect(latest.current.activeModelOption).toEqual({
      providerName: 'codex',
      providerType: 'codex',
      model: 'gpt-5.4',
    });
  });

  it('loads lightweight session detail to restore runtime selection when history is cached', async () => {
    mockChat.shouldLoadSessionHistory.mockReturnValue(false);
    jest.mocked(getSession).mockResolvedValue({
      last_runtime_selection: {
        runtime: 'ghost',
        provider: 'openai-main',
        provider_type: 'openai',
        model: 'gpt-5.4',
        mode: 'plan',
      },
    } as Awaited<ReturnType<typeof getSession>>);
    const latest = renderController();

    await act(async () => {
      latest.current.selectSession('session-2');
      await Promise.resolve();
    });

    expect(getSession).toHaveBeenCalledWith('session-2', { limit: 1 });
    expect(latest.current.agentMode).toBeNull();
    expect(mockConfig.selectActiveModel).toHaveBeenCalledWith({
      providerName: 'openai-main',
      providerType: 'openai',
      model: 'gpt-5.4',
    });
  });

  it('uses backend Codex model options while codex mode is active', async () => {
    jest.mocked(getCodexModelCatalog).mockResolvedValue(CODEX_CATALOG);
    mockConfig.activeModelOption = {
      providerName: 'openai-main',
      providerType: 'openai',
      model: 'gpt-ghost',
    };
    mockConfig.modelOptions = [mockConfig.activeModelOption];
    const latest = renderController();

    await act(async () => {
      latest.current.setAgentMode('normal');
      await Promise.resolve();
    });

    expect(latest.current.activeModelOption).toEqual({
      providerName: 'codex',
      providerType: 'codex',
      model: CODEX_CATALOG.default_model,
    });
    expect(latest.current.modelOptions).toEqual(CODEX_CATALOG.models.map((model) => ({
      providerName: 'codex',
      providerType: 'codex',
      model,
    })));
  });

  it('drops chat state and clears current messages after deleting the active session', async () => {
    mockSessions.currentSessionId = 'session-1';
    const latest = renderController();

    await act(async () => {
      await latest.current.deleteSession('session-1');
    });

    expect(mockSessions.deleteSession).toHaveBeenCalledWith('session-1');
    expect(mockChat.dropSessionState).toHaveBeenCalledWith('session-1');
    expect(mockChat.clearMessages).toHaveBeenCalledWith('');
  });

  it('opens settings from query and strips settings query on close', () => {
    installWindowSearch('?settings=tasks&source=launcher');
    const latest = renderController();

    expect(latest.current.showConfig).toBe(true);
    expect(latest.current.settingsTabFromQuery).toBe('tasks');

    act(() => {
      latest.current.closeConfig();
    });

    expect(mockRouter.replace).toHaveBeenCalledWith('/?source=launcher');
  });
});

function renderController() {
  const latest: { current: ReturnType<typeof useHomePageController> } = {
    current: null as unknown as ReturnType<typeof useHomePageController>,
  };

  act(() => {
    TestRenderer.create(
      React.createElement(HomePageControllerProbe, {
        onRender: (state) => {
          latest.current = state;
        },
      }),
    );
  });

  return latest;
}

function HomePageControllerProbe(props: {
  onRender: (state: ReturnType<typeof useHomePageController>) => void;
}) {
  const state = useHomePageController();
  props.onRender(state);
  return null;
}

function installWindowSearch(search: string): void {
  (globalThis as { window?: unknown }).window = {
    location: { search },
    addEventListener: () => undefined,
    removeEventListener: () => undefined,
  };
}

function buildRouter() {
  return {
    push: jest.fn(),
    replace: jest.fn(),
  };
}

function buildSessionsController() {
  return {
    sessions: [],
    currentSessionId: 'session-1',
    loading: false,
    error: '',
    setCurrentSessionId: jest.fn(),
    loadSessions: jest.fn().mockResolvedValue(undefined),
    deleteSession: jest.fn().mockResolvedValue(undefined),
    createNewSession: jest.fn(),
  };
}

function buildChatController() {
  return {
    backgroundCompletedSessionIds: new Set<string>(),
    committedMessages: [],
    streamingAssistantSegments: [],
    streamingThinkingSegments: [],
    activeStreamingThinkingId: null,
    streamingItemOrder: [],
    streamingTools: [],
    pendingQuestions: [],
    loading: false,
    historyLoading: false,
    loadingOlderHistory: false,
    chatError: '',
    hasPendingQuestion: false,
    hasOlderHistory: false,
    postSendFocusRequest: null,
    canStop: false,
    answerQuestion: jest.fn(),
    cancelQuestion: jest.fn(),
    loadOlderHistory: jest.fn(),
    stopCurrentRun: jest.fn(),
    sendChatMessage: jest.fn().mockResolvedValue(undefined),
    clearBackgroundCompletion: jest.fn(),
    shouldLoadSessionHistory: jest.fn().mockReturnValue(false),
    loadSessionHistory: jest.fn().mockResolvedValue(null),
    dropSessionState: jest.fn(),
    clearMessages: jest.fn(),
  };
}

function buildConfigController(): {
  activeModelOption: ProviderModelOption | null;
  config: {
    session_system_prompt_visible_enabled: boolean;
  };
  configError: string;
  configLoading: boolean;
  modelOptions: ProviderModelOption[];
  modelOptionsLoading: boolean;
  refreshConfig: jest.Mock;
  saveConfig: jest.Mock;
  savingConfig: boolean;
  selectActiveModel: jest.Mock;
} {
  return {
    config: {
      session_system_prompt_visible_enabled: true,
    },
    configLoading: false,
    savingConfig: false,
    configError: '',
    modelOptionsLoading: false,
    activeModelOption: null,
    modelOptions: [],
    saveConfig: jest.fn(),
    selectActiveModel: jest.fn(),
    refreshConfig: jest.fn(),
  };
}
