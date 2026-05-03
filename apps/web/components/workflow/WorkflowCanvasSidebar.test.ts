import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { copyForWorkflow } from '@/lib/i18n/messages/workflow';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import { createEmptyWorkflowDraft } from '@/lib/workflow-editor/draft';
import { buildOrchestrationWorkflowCopy } from '@/components/orchestration/orchestrationEditorCopy';
import { WorkflowCanvasSidebar } from './WorkflowCanvasSidebar';

describe('components/workflow/WorkflowCanvasSidebar', () => {
  it('does not expose start or end in the default node library', () => {
    const renderer = renderSidebar();
    const content = textContent(renderer.root);

    expect(content).toContain('角色');
    expect(content).toContain('LLM 模型');
    expect(content).toContain('工具调用');
    expect(content).toContain('条件分支');
    expect(content).toContain('循环');
    expect(content).not.toContain('开始');
    expect(content).not.toContain('结束');
  });

  it('filters the node library to the provided node types', () => {
    const renderer = renderSidebar(['agent']);
    const libraryLabels = renderer.root
      .findAll((node) => node.props.className === 'workflow-arch-library-item')
      .map((node) => textContent(node.findByType('strong')));

    expect(libraryLabels).toEqual(['角色']);
    expect(textContent(renderer.root)).not.toContain('LLM 模型');
    expect(textContent(renderer.root)).not.toContain('工具调用');
    expect(textContent(renderer.root)).not.toContain('条件分支');
    expect(textContent(renderer.root)).not.toContain('循环');
    expect(textContent(renderer.root)).not.toContain('开始');
    expect(textContent(renderer.root)).not.toContain('结束');
  });
});

function renderSidebar(nodeLibraryTypes?: readonly ('agent')[]) {
  let renderer!: TestRenderer.ReactTestRenderer;

  act(() => {
    renderer = TestRenderer.create(
      React.createElement(WebLocaleProvider, {
        initialLocale: 'zh-CN',
        children: React.createElement(WorkflowCanvasSidebar, {
          isOpen: true,
          draft: createEmptyWorkflowDraft(),
          autosaveState: { phase: 'idle', message: 'Autosave idle', updatedAt: 0 },
          importSessionID: '',
          importLoading: false,
          workflowCopy: buildOrchestrationWorkflowCopy(copyForWorkflow('zh-CN'), 'zh-CN'),
          nodeLibraryTypes,
          onToggle: jest.fn(),
          onAddNode: jest.fn(),
          onChangeImportSessionID: jest.fn(),
          onImportFromSession: jest.fn(),
          onScheduleChange: jest.fn(),
          onOpenSettings: jest.fn(),
          onBack: jest.fn(),
          onSave: jest.fn(),
        }),
      }),
    );
  });

  return renderer;
}

function textContent(node: TestRenderer.ReactTestInstance): string {
  return node.children.map((child: string | number | TestRenderer.ReactTestInstance) => {
    if (typeof child === 'string' || typeof child === 'number') {
      return String(child);
    }
    return textContent(child);
  }).join('');
}
