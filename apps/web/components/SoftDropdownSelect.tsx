'use client';

import { useEffect, useId, useMemo, useRef, useState, type RefObject } from 'react';

export interface DropdownSelectOption {
  value: string;
  label: string;
  disabled?: boolean;
}

interface SoftDropdownSelectProps {
  value: string;
  options: readonly DropdownSelectOption[];
  onChange: (value: string) => void;
  placeholder?: string;
  disabled?: boolean;
  testId?: string;
}

const TRIGGER_CLASS = 'flex w-full items-center justify-between rounded-lg border border-gray-300 bg-white px-4 py-2.5 text-left text-sm text-gray-900 shadow-sm transition-colors duration-200 hover:border-gray-400 focus:outline-none focus:ring-2 focus:ring-gray-900 focus:border-transparent disabled:cursor-not-allowed disabled:opacity-60';
const PANEL_BASE_CLASS = 'absolute left-0 top-full z-10 mt-1 max-h-72 w-full overflow-y-auto rounded-lg border border-gray-200 bg-white py-1 text-sm shadow-lg transition-opacity duration-150';
const PANEL_OPEN_CLASS = 'visible opacity-100';
const PANEL_CLOSED_CLASS = 'pointer-events-none invisible opacity-0';
const OPTION_BASE_CLASS = 'flex w-full items-center px-4 py-2 text-left transition-colors duration-150';
const OPTION_ENABLED_CLASS = 'cursor-pointer text-gray-700 hover:bg-gray-50 hover:text-gray-900';
const OPTION_SELECTED_CLASS = 'bg-gray-50 font-medium text-gray-900';
const OPTION_DISABLED_CLASS = 'cursor-not-allowed text-gray-400';

export function SoftDropdownSelect(props: SoftDropdownSelectProps) {
  const { value, options, onChange, placeholder = '', disabled = false, testId } = props;
  const listboxID = useId();
  const rootRef = useRef<HTMLDivElement>(null);
  const [open, setOpen] = useState(false);
  const selectedOption = useMemo(
    () => options.find((option) => option.value === value),
    [options, value],
  );
  const selectedLabel = selectedOption?.label ?? value;
  const triggerText = selectedOption?.label ?? (placeholder || value);

  useDismissDropdown(open, rootRef, () => setOpen(false));

  return (
    <div ref={rootRef} className="relative w-full select-none">
      <HiddenNativeSelect
        testId={testId}
        value={value}
        selectedLabel={selectedLabel}
        options={options}
        disabled={disabled}
        onChange={onChange}
      />
      <button
        type="button"
        disabled={disabled}
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-controls={listboxID}
        className={TRIGGER_CLASS}
        onClick={() => setOpen((current) => !current)}
      >
        <span className="truncate pr-4">{triggerText}</span>
        <span className={`transform transition-transform duration-200 ${open ? 'rotate-180' : ''}`} aria-hidden="true">
          <ChevronDownIcon />
        </span>
      </button>
      <ul
        id={listboxID}
        role="listbox"
        aria-hidden={!open}
        className={`${PANEL_BASE_CLASS} ${open ? PANEL_OPEN_CLASS : PANEL_CLOSED_CLASS}`}
      >
        {options.map((option) => (
          <DropdownOption
            key={option.value}
            option={option}
            selected={option.value === value}
            onSelect={(nextValue) => {
              onChange(nextValue);
              setOpen(false);
            }}
          />
        ))}
      </ul>
    </div>
  );
}

function HiddenNativeSelect(props: {
  testId?: string;
  value: string;
  selectedLabel: string;
  options: readonly DropdownSelectOption[];
  disabled: boolean;
  onChange: (value: string) => void;
}) {
  const { testId, value, selectedLabel, options, disabled, onChange } = props;
  const needsCurrentOption = !options.some((option) => option.value === value);

  return (
    <select
      data-testid={testId}
      value={value}
      disabled={disabled}
      aria-hidden="true"
      tabIndex={-1}
      onChange={(event) => onChange(event.target.value)}
      className="sr-only"
    >
      {needsCurrentOption ? <option value={value}>{selectedLabel}</option> : null}
      {options.map((option) => (
        <option key={option.value} value={option.value} disabled={option.disabled}>
          {option.label}
        </option>
      ))}
    </select>
  );
}

function DropdownOption(props: {
  option: DropdownSelectOption;
  selected: boolean;
  onSelect: (value: string) => void;
}) {
  const { option, selected, onSelect } = props;
  const disabled = option.disabled === true;

  return (
    <li role="presentation">
      <button
        type="button"
        role="option"
        aria-selected={selected}
        disabled={disabled}
        className={optionClassName(selected, disabled)}
        onClick={() => {
          if (!disabled) {
            onSelect(option.value);
          }
        }}
      >
        <span className="truncate">{option.label}</span>
      </button>
    </li>
  );
}

function useDismissDropdown(
  open: boolean,
  rootRef: RefObject<HTMLDivElement>,
  onDismiss: () => void,
) {
  useEffect(() => {
    if (!open || typeof document === 'undefined') {
      return undefined;
    }

    const handlePointerDown = (event: MouseEvent | TouchEvent) => {
      if (!rootRef.current?.contains(event.target as Node)) {
        onDismiss();
      }
    };
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onDismiss();
      }
    };

    document.addEventListener('mousedown', handlePointerDown);
    document.addEventListener('touchstart', handlePointerDown);
    document.addEventListener('keydown', handleKeyDown);
    return () => {
      document.removeEventListener('mousedown', handlePointerDown);
      document.removeEventListener('touchstart', handlePointerDown);
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [onDismiss, open, rootRef]);
}

function ChevronDownIcon() {
  return (
    <svg className="h-4 w-4 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 9l-7 7-7-7" />
    </svg>
  );
}

function optionClassName(selected: boolean, disabled: boolean): string {
  if (disabled) {
    return `${OPTION_BASE_CLASS} ${OPTION_DISABLED_CLASS}`;
  }
  if (selected) {
    return `${OPTION_BASE_CLASS} ${OPTION_ENABLED_CLASS} ${OPTION_SELECTED_CLASS}`;
  }
  return `${OPTION_BASE_CLASS} ${OPTION_ENABLED_CLASS}`;
}
