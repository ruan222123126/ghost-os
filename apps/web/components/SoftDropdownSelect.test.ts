import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { SoftDropdownSelect } from './SoftDropdownSelect';

describe('components/SoftDropdownSelect', () => {
  it('shows placeholder text when the current value has no option label', () => {
    const renderer = renderSelect({ value: '', placeholder: 'Choose model' });

    expect(textContent(renderer.root)).toContain('Choose model');
  });

  it('calls onChange for enabled options and ignores disabled options', () => {
    const onChange = jest.fn();
    const renderer = renderSelect({ onChange, value: 'openai' });
    const optionButtons = renderer.root.findAllByProps({ role: 'option' });

    act(() => {
      optionButtons[1].props.onClick();
      optionButtons[2].props.onClick();
    });

    expect(onChange).toHaveBeenCalledTimes(1);
    expect(onChange).toHaveBeenCalledWith('anthropic');
  });
});

function renderSelect(props: {
  value: string;
  placeholder?: string;
  onChange?: (value: string) => void;
}): TestRenderer.ReactTestRenderer {
  let renderer!: TestRenderer.ReactTestRenderer;

  act(() => {
    renderer = TestRenderer.create(
      React.createElement(SoftDropdownSelect, {
        value: props.value,
        placeholder: props.placeholder,
        onChange: props.onChange ?? jest.fn(),
        options: [
          { value: 'openai', label: 'OpenAI' },
          { value: 'anthropic', label: 'Anthropic' },
          { value: 'disabled', label: 'Disabled', disabled: true },
        ],
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
