import {
  CLICK_COORDINATE_REF_KEY,
  CLICK_COORDINATE_SOURCE_FIND_ICON,
  CLICK_COORDINATE_SOURCE_MANUAL,
  SCREEN_CONTROL_FIND_ICON_COORDINATE_REF,
  buildInitialEditorState,
  buildSavedClickStep,
} from '@/components/workflow/workflowClickStepEditorHelpers';
import type { ScreenControlComposerStep } from '@/lib/workflow-editor';

describe('components/workflow/workflowClickStepEditorHelpers', () => {
  it('builds find_icon reference click step without manual coordinates', () => {
    const step: ScreenControlComposerStep = {
      action: 'click',
      params: { x: 12, y: 34, position_type: 'absolute', display_id: 2 },
    };

    expect(buildSavedClickStep(step, {
      x: '12',
      y: '34',
      coordinateSource: CLICK_COORDINATE_SOURCE_FIND_ICON,
      positionType: 'absolute',
      displayID: 2,
    })).toEqual({
      action: 'click',
      params: {
        [CLICK_COORDINATE_REF_KEY]: SCREEN_CONTROL_FIND_ICON_COORDINATE_REF,
      },
    });
  });

  it('reads stored find_icon coordinate reference into editor state', () => {
    const step: ScreenControlComposerStep = {
      action: 'click',
      params: { [CLICK_COORDINATE_REF_KEY]: SCREEN_CONTROL_FIND_ICON_COORDINATE_REF },
    };

    expect(buildInitialEditorState(step)).toMatchObject({
      x: '',
      y: '',
      coordinateSource: CLICK_COORDINATE_SOURCE_FIND_ICON,
    });
  });

  it('keeps manual coordinate source for numeric click steps', () => {
    const step: ScreenControlComposerStep = {
      action: 'click',
      params: { x: 12, y: 34 },
    };

    expect(buildInitialEditorState(step).coordinateSource).toBe(CLICK_COORDINATE_SOURCE_MANUAL);
  });
});
