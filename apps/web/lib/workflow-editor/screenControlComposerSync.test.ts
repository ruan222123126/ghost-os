import { syncScreenControlComposerStepsToToolArguments } from '@/lib/workflow-editor/screenControlComposerSync';
import type {
  ScreenControlComposerStep,
  WorkflowCanvasNodeDraft,
} from '@/lib/workflow-editor/types';

describe('lib/workflow-editor/screenControlComposerSync', () => {
  it('writes workflow_steps when composer has multiple steps', () => {
    const node: WorkflowCanvasNodeDraft = {
      id: 'tool-node',
      type: 'tool',
      position: { x: 0, y: 0 },
      ui: { toolArgumentsMode: 'kv' },
      tool: {
        tool_name: 'screen_control',
        arguments: {
          mode: 'atomic',
          action: 'click_icon',
          params: {
            x: 120,
            y: 300,
          },
          workflow_steps: [{ action: 'stale', params: { keep: false } }],
          keep_top_level: true,
        },
      },
    };
    const steps: ScreenControlComposerStep[] = [
      { action: 'screenshot' },
      { action: 'find_text' },
      {
        action: 'click',
        params: {
          x: 10,
          y: 20,
        },
      },
    ];

    expect(syncScreenControlComposerStepsToToolArguments(node, steps).tool?.arguments).toEqual({
      mode: 'atomic',
      keep_top_level: true,
      workflow_steps: [
        { action: 'screenshot', params: {} },
        { action: 'find_text', params: {} },
        { action: 'click', params: { x: 10, y: 20 } },
      ],
    });
  });

  it('keeps single step with legacy action/params fields', () => {
    const node: WorkflowCanvasNodeDraft = {
      id: 'tool-node',
      type: 'tool',
      position: { x: 0, y: 0 },
      ui: { toolArgumentsMode: 'kv' },
      tool: {
        tool_name: 'screen_control',
        arguments: {
          mode: 'atomic',
          action: 'click_icon',
          params: {
            x: 120,
            y: 300,
          },
          workflow_steps: [{ action: 'screenshot', params: {} }],
          keep_top_level: true,
        },
      },
    };
    const steps: ScreenControlComposerStep[] = [{
      action: 'click',
      params: {
        x: 88,
        y: 99,
      },
    }];

    expect(syncScreenControlComposerStepsToToolArguments(node, steps).tool?.arguments).toEqual({
      mode: 'atomic',
      action: 'click_icon',
      keep_top_level: true,
      params: {
        x: 88,
        y: 99,
      },
    });
  });

  it('clears stale action, params and workflow_steps when steps become empty', () => {
    const node: WorkflowCanvasNodeDraft = {
      id: 'tool-node',
      type: 'tool',
      position: { x: 0, y: 0 },
      ui: { toolArgumentsMode: 'kv' },
      tool: {
        tool_name: 'screen_control',
        arguments: {
          mode: 'atomic',
          action: 'click_icon',
          params: {
            x: 120,
            y: 300,
          },
          workflow_steps: [{ action: 'screenshot', params: {} }],
          keep_top_level: true,
        },
      },
    };

    expect(syncScreenControlComposerStepsToToolArguments(node, []).tool?.arguments).toEqual({
      mode: 'atomic',
      keep_top_level: true,
    });
  });

  it('clears stale action when single click step has invalid params', () => {
    const node: WorkflowCanvasNodeDraft = {
      id: 'tool-node',
      type: 'tool',
      position: { x: 0, y: 0 },
      ui: { toolArgumentsMode: 'kv' },
      tool: {
        tool_name: 'screen_control',
        arguments: {
          mode: 'atomic',
          action: 'find_icon',
          params: {
            display_id: 7,
          },
          workflow_steps: [{ action: 'find_text', params: {} }],
          keep_top_level: true,
        },
      },
    };
    const steps: ScreenControlComposerStep[] = [{ action: 'click', params: { x: 10 } }];

    expect(syncScreenControlComposerStepsToToolArguments(node, steps).tool?.arguments).toEqual({
      mode: 'atomic',
      keep_top_level: true,
    });
  });

  it('keeps coordinate_ref inside multi-step click workflow steps', () => {
    const node: WorkflowCanvasNodeDraft = {
      id: 'tool-node',
      type: 'tool',
      position: { x: 0, y: 0 },
      ui: { toolArgumentsMode: 'kv' },
      tool: {
        tool_name: 'screen_control',
        arguments: { keep_top_level: true },
      },
    };
    const steps: ScreenControlComposerStep[] = [
      { action: 'find_icon', params: { template_path: '/tmp/icon.png' } },
      { action: 'click', params: { coordinate_ref: '${find_icon}' } },
    ];

    expect(syncScreenControlComposerStepsToToolArguments(node, steps).tool?.arguments).toEqual({
      keep_top_level: true,
      workflow_steps: [
        { action: 'find_icon', params: { template_path: '/tmp/icon.png' } },
        { action: 'click', params: { coordinate_ref: '${find_icon}' } },
      ],
    });
  });
});
