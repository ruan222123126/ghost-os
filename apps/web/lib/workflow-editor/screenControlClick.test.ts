import {
  normalizeClickComposerParams,
  syncClickStepToToolArguments,
  withClickComposerParams,
} from '@/lib/workflow-editor/screenControlClick';
import type { ScreenControlComposerStep, WorkflowCanvasNodeDraft } from '@/lib/workflow-editor/types';

describe('lib/workflow-editor/screenControlClick', () => {
  it('normalizes click params only when x/y exist', () => {
    expect(normalizeClickComposerParams(undefined)).toBeUndefined();
    expect(normalizeClickComposerParams({ x: 10 })).toBeUndefined();
    expect(normalizeClickComposerParams({ y: 20 })).toBeUndefined();
    expect(normalizeClickComposerParams({ x: 11, y: 22 })).toEqual({ x: 11, y: 22 });
  });

  it('updates click step params while keeping unknown fields', () => {
    const step: ScreenControlComposerStep = {
      action: 'click',
      params: { keep: true },
    };

    expect(withClickComposerParams(step, { x: 100, y: 200 })).toEqual({
      action: 'click',
      params: {
        keep: true,
        x: 100,
        y: 200,
      },
    });
  });

  it('syncs click params to screen_control tool arguments', () => {
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
            display_id: 2,
            threshold: 0.9,
          },
          keep_top_level: true,
        },
      },
    };
    const step: ScreenControlComposerStep = {
      action: 'click',
      params: {
        x: 333,
        y: 444,
      },
    };

    expect(syncClickStepToToolArguments(node, step).tool?.arguments).toEqual({
      mode: 'atomic',
      action: 'click_icon',
      keep_top_level: true,
      params: {
        display_id: 2,
        threshold: 0.9,
        x: 333,
        y: 444,
      },
    });
  });
});
