import {
  appendScreenControlComposerStep,
  moveScreenControlComposerStep,
  removeScreenControlComposerStep,
  updateScreenControlComposerStep,
} from '@/lib/workflow-editor/screenControlComposer';
import type { ScreenControlComposerStep } from '@/lib/workflow-editor/types';

describe('lib/workflow-editor/screenControlComposer', () => {
  it('appends steps in click order', () => {
    const initial: ScreenControlComposerStep[] = [];
    const afterShot = appendScreenControlComposerStep(initial, 'screenshot');
    const afterFindText = appendScreenControlComposerStep(afterShot, 'find_text');

    expect(afterFindText.map((item) => item.action)).toEqual(['screenshot', 'find_text']);
    expect(initial).toEqual([]);
  });

  it('moves step up and down by index', () => {
    const steps: ScreenControlComposerStep[] = [
      { action: 'screenshot' },
      { action: 'find_text' },
      { action: 'click' },
    ];

    const movedUp = moveScreenControlComposerStep(steps, 2, 'up');
    const movedDown = moveScreenControlComposerStep(movedUp, 1, 'down');

    expect(movedUp.map((item) => item.action)).toEqual(['screenshot', 'click', 'find_text']);
    expect(movedDown.map((item) => item.action)).toEqual(['screenshot', 'find_text', 'click']);
    expect(steps.map((item) => item.action)).toEqual(['screenshot', 'find_text', 'click']);
  });

  it('keeps order when move exceeds boundary', () => {
    const steps: ScreenControlComposerStep[] = [{ action: 'screenshot' }, { action: 'find_text' }];

    const topMove = moveScreenControlComposerStep(steps, 0, 'up');
    const bottomMove = moveScreenControlComposerStep(steps, 1, 'down');

    expect(topMove.map((item) => item.action)).toEqual(['screenshot', 'find_text']);
    expect(bottomMove.map((item) => item.action)).toEqual(['screenshot', 'find_text']);
  });

  it('removes step by index', () => {
    const steps: ScreenControlComposerStep[] = [
      { action: 'screenshot' },
      { action: 'find_text' },
      { action: 'click' },
    ];

    const removed = removeScreenControlComposerStep(steps, 1);

    expect(removed.map((item) => item.action)).toEqual(['screenshot', 'click']);
    expect(steps.map((item) => item.action)).toEqual(['screenshot', 'find_text', 'click']);
  });

  it('updates step by index', () => {
    const steps: ScreenControlComposerStep[] = [
      { action: 'screenshot' },
      { action: 'find_icon', params: { template_path: '/tmp/old.png' } },
    ];

    const updated = updateScreenControlComposerStep(steps, 1, {
      action: 'find_icon',
      params: { template_path: '/tmp/new.png' },
    });

    expect(updated).toEqual([
      { action: 'screenshot' },
      { action: 'find_icon', params: { template_path: '/tmp/new.png' } },
    ]);
    expect(steps[1].params).toEqual({ template_path: '/tmp/old.png' });
  });

  it('normalizes legacy click_text/click_icon actions', () => {
    const appended = appendScreenControlComposerStep([], 'click_icon');
    const normalizedText = updateScreenControlComposerStep([{ action: 'click_text' }], 0, { action: 'click_text' });
    const normalizedIcon = updateScreenControlComposerStep([{ action: 'click_icon' }], 0, { action: 'click_icon' });

    expect(appended).toEqual([{ action: 'click' }]);
    expect(normalizedText).toEqual([{ action: 'find_text' }]);
    expect(normalizedIcon).toEqual([{ action: 'click' }]);
  });
});
