import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { copyForWorkflow } from '@/lib/i18n/messages/workflow';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import { OrchestrationEditorClient } from './OrchestrationEditorClient';

const renderWorkbench = jest.fn();
const useOrchestrationEditorController = jest.fn();

jest.mock('@/components/workflow/WorkflowCanvasWorkbench', () => ({
  WorkflowCanvasWorkbench: (props: unknown) => {
    renderWorkbench(props);
    return React.createElement('div', { 'data-testid': 'workflow-canvas-workbench' });
  },
}));

jest.mock('./useOrchestrationEditorController', () => ({
  useOrchestrationEditorController: (...args: unknown[]) => useOrchestrationEditorController(...args),
}));

describe('components/orchestration/OrchestrationEditorClient', () => {
  beforeEach(() => {
    renderWorkbench.mockReset();
    useOrchestrationEditorController.mockReset();
    useOrchestrationEditorController.mockReturnValue(buildController());
  });

  it('limits orchestration library nodes to agent only', () => {
    renderClient();

    expect(renderWorkbench).toHaveBeenCalled();
    expect(renderWorkbench.mock.calls.at(-1)?.[0]).toMatchObject({
      editorKind: 'orchestration',
      nodeLibraryTypes: ['agent'],
    });
  });
});

function renderClient() {
  let renderer!: TestRenderer.ReactTestRenderer;

  act(() => {
    renderer = TestRenderer.create(
      React.createElement(WebLocaleProvider, {
        initialLocale: 'zh-CN',
        children: React.createElement(OrchestrationEditorClient, {
          orchestrationID: 'orch_1',
        }),
      }),
    );
  });

  return renderer;
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
    workflowCopy: copyForWorkflow('zh-CN'),
    localizeValidationError: jest.fn(),
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
