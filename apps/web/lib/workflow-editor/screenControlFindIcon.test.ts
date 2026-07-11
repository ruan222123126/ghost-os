import {
  normalizeFindIconComposerParams,
  syncFindIconStepToToolArguments,
  withFindIconComposerParams,
} from '@/lib/workflow-editor/screenControlFindIcon';
import type { ScreenControlComposerStep, WorkflowCanvasNodeDraft } from '@/lib/workflow-editor/types';

describe('lib/workflow-editor/screenControlFindIcon', () => {
  it('normalizes find_icon params only when template_path exists', () => {
    expect(normalizeFindIconComposerParams(undefined)).toBeUndefined();
    expect(normalizeFindIconComposerParams({ threshold: 0.9 })).toBeUndefined();
    expect(normalizeFindIconComposerParams({
      template_path: ' /tmp/icon.png ',
      template_name: ' icon.png ',
      threshold: 0.91,
      max_results: 3,
    })).toEqual({
      template_path: '/tmp/icon.png',
      template_name: 'icon.png',
      threshold: 0.91,
      max_results: 3,
    });
  });

  it('updates find_icon step params while keeping unknown fields', () => {
    const step: ScreenControlComposerStep = {
      action: 'find_icon',
      params: { keep: true },
    };

    expect(withFindIconComposerParams(step, {
      template_path: '/tmp/new.png',
      threshold: 0.88,
    })).toEqual({
      action: 'find_icon',
      params: {
        keep: true,
        template_path: '/tmp/new.png',
        threshold: 0.88,
      },
    });
  });

  it('syncs find_icon params to screen_control tool arguments', () => {
    const node: WorkflowCanvasNodeDraft = {
      id: 'tool-node',
      type: 'tool',
      position: { x: 0, y: 0 },
      ui: { toolArgumentsMode: 'kv' },
      tool: {
        tool_name: 'screen_control',
        arguments: {
          mode: 'agent',
          action: 'click_text',
          params: {
            display_id: 2,
            template_path: '/tmp/old.png',
          },
          keep_top_level: true,
        },
      },
    };
    const step: ScreenControlComposerStep = {
      action: 'find_icon',
      params: {
        template_path: '/tmp/new.png',
        threshold: 0.92,
        max_results: 5,
      },
    };

    expect(syncFindIconStepToToolArguments(node, step).tool?.arguments).toEqual({
      mode: 'atomic',
      action: 'find_icon',
      keep_top_level: true,
      params: {
        display_id: 2,
        template_path: '/tmp/new.png',
        threshold: 0.92,
        max_results: 5,
      },
    });
  });

  it('syncs hover_after_match flag to tool arguments', () => {
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
            template_path: '/tmp/icon.png',
          },
        },
      },
    };
    const step: ScreenControlComposerStep = {
      action: 'find_icon',
      params: {
        template_path: '/tmp/icon.png',
        hover_after_match: true,
      },
    };

    expect(syncFindIconStepToToolArguments(node, step).tool?.arguments).toEqual({
      mode: 'atomic',
      action: 'find_icon',
      params: {
        template_path: '/tmp/icon.png',
        hover_after_match: true,
      },
    });
  });
});
