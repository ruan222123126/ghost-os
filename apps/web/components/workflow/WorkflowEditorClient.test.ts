import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import { WorkflowEditorClient } from './WorkflowEditorClient';

const renderWorkbench = jest.fn();
const useWorkflowEditorController = jest.fn();
const TestLocaleProvider = WebLocaleProvider as React.ComponentType<{
  initialLocale: 'zh-CN';
  children?: React.ReactNode;
}>;

jest.mock('@/components/workflow/WorkflowCanvasWorkbench', () => ({
  WorkflowCanvasWorkbench: (props: unknown) => {
    renderWorkbench(props);
    return React.createElement('div', { 'data-testid': 'workflow-canvas-workbench' });
  },
}));

jest.mock('@/hooks/workflow/useWorkflowEditorController', () => ({
  useWorkflowEditorController: (...args: unknown[]) => useWorkflowEditorController(...args),
}));

describe('components/workflow/WorkflowEditorClient', () => {
  beforeEach(() => {
    renderWorkbench.mockReset();
    useWorkflowEditorController.mockReset();
    useWorkflowEditorController.mockReturnValue(buildController());
  });

  it('passes workflow editor kind into the shared workbench', () => {
    renderClient();

    expect(renderWorkbench.mock.calls.at(-1)?.[0]).toMatchObject({
      editorKind: 'workflow',
      importControls: {
        sessionID: '',
        loading: false,
      },
    });
  });
});

function renderClient() {
  act(() => {
    TestRenderer.create(
      React.createElement(TestLocaleProvider, {
        initialLocale: 'zh-CN',
      }, React.createElement(WorkflowEditorClient, {
        mode: 'edit',
        taskID: 'task-1',
      })),
    );
  });
}

function buildController() {
  return {
    phase: 'ready',
    draft: { mode: 'edit', schedule: { mode: 'interval', intervalSeconds: '', cronExpr: '' }, nodes: [], edges: [] },
    autosaveState: { phase: 'idle', message: 'Autosave idle', updatedAt: 0 },
    actionError: '',
    validationErrors: [],
    importSessionID: '',
    importLoading: false,
    onChangeImportSessionID: jest.fn(),
    onImportFromSession: jest.fn(),
    onScheduleChange: jest.fn(),
    onAddNode: jest.fn(),
    onSelectNode: jest.fn(),
    onMoveNode: jest.fn(),
    onConnectNodes: jest.fn(),
    onDeleteEdge: jest.fn(),
    onDuplicateNode: jest.fn(),
    onUpdateNode: jest.fn(),
    onDeleteNode: jest.fn(),
    onSave: jest.fn(),
    onBack: jest.fn(),
  };
}
