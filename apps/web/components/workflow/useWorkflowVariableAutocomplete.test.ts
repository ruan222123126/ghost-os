import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { WORKFLOW_FIND_ICON_VARIABLE } from '@/components/workflow/workflowVariableAutocomplete';
import {
  useWorkflowVariableAutocomplete,
  type WorkflowVariableAutocompleteState,
} from '@/components/workflow/useWorkflowVariableAutocomplete';

describe('components/workflow/useWorkflowVariableAutocomplete', () => {
  it('commits the active variable option with Enter', () => {
    const changes: string[] = [];
    let latestState: WorkflowVariableAutocompleteState | undefined;

    act(() => {
      TestRenderer.create(
        React.createElement(HookProbe, {
          value: 'prefix {fi',
          onChange: (value) => changes.push(value),
          onRender: (state) => {
            latestState = state;
          },
        }),
      );
    });
    act(() => {
      latestState!.handleFocus();
    });

    const event = createKeyDownEvent('Enter');
    act(() => {
      latestState!.handleKeyDown(event);
    });

    expect(event.preventDefault).toHaveBeenCalledTimes(1);
    expect(changes).toEqual([`prefix ${WORKFLOW_FIND_ICON_VARIABLE}`]);
  });

  it('dismisses the current trigger range with Escape', () => {
    let latestState: WorkflowVariableAutocompleteState | undefined;

    act(() => {
      TestRenderer.create(
        React.createElement(HookProbe, {
          value: 'prefix {fi',
          onChange: () => undefined,
          onRender: (state) => {
            latestState = state;
          },
        }),
      );
    });
    act(() => {
      latestState!.handleFocus();
    });

    expect(latestState!.dropdownOpen).toBe(true);
    act(() => {
      latestState!.handleKeyDown(createKeyDownEvent('Escape'));
    });

    expect(latestState!.dropdownOpen).toBe(false);
  });
});

interface HookProbeProps {
  value: string;
  onChange: (value: string) => void;
  onRender: (state: WorkflowVariableAutocompleteState) => void;
}

function HookProbe(props: HookProbeProps) {
  const state = useWorkflowVariableAutocomplete({
    value: props.value,
    onChange: props.onChange,
  });
  props.onRender(state);
  return null;
}

function createKeyDownEvent(key: string) {
  return {
    key,
    preventDefault: jest.fn(),
  } as unknown as React.KeyboardEvent<HTMLInputElement>;
}
