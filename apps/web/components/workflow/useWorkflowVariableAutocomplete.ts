'use client';

import type { Dispatch, KeyboardEvent, MouseEvent, RefObject, SetStateAction } from 'react';
import { useEffect, useMemo, useRef, useState } from 'react';
import {
  WORKFLOW_AUTOCOMPLETE_OPTIONS,
  applyVariableOption,
  filterWorkflowVariableOptions,
  resolveVariableTriggerRange,
  type VariableTriggerRange,
  type WorkflowAutocompleteOption,
} from '@/components/workflow/workflowVariableAutocomplete';

interface WorkflowVariableAutocompleteProps {
  value: string;
  disabled?: boolean;
  readOnly?: boolean;
  onChange: (value: string) => void;
}

export interface WorkflowVariableAutocompleteState {
  activeIndex: number;
  filteredOptions: WorkflowAutocompleteOption[];
  dropdownOpen: boolean;
  fieldRef: RefObject<FieldElement>;
  syncCaretPosition: (target: FieldElement) => void;
  handleChange: (target: FieldElement) => void;
  handleKeyDown: (event: KeyboardEvent<FieldElement>) => void;
  handleOptionMouseDown: (event: MouseEvent<HTMLButtonElement>) => void;
  handleOptionClick: (option: WorkflowAutocompleteOption) => void;
  handleFocus: () => void;
  handleBlur: () => void;
}

type FieldElement = HTMLInputElement | HTMLTextAreaElement;

const EMPTY_OPTIONS: WorkflowAutocompleteOption[] = [];

export function useWorkflowVariableAutocomplete(
  props: WorkflowVariableAutocompleteProps,
): WorkflowVariableAutocompleteState {
  const { value, disabled = false, readOnly = false, onChange } = props;
  const fieldRef = useRef<FieldElement | null>(null);
  const [activeIndex, setActiveIndex] = useState(0);
  const [caretPosition, setCaretPosition] = useState(value.length);
  const [dismissedRangeKey, setDismissedRangeKey] = useState<string>();
  const [isFocused, setIsFocused] = useState(false);
  const [pendingCaretPosition, setPendingCaretPosition] = useState<number>();
  const triggerRange = useMemo(() => resolveVariableTriggerRange(value, caretPosition), [caretPosition, value]);
  const filteredOptions = useMemo(
    () => (triggerRange ? filterWorkflowVariableOptions(WORKFLOW_AUTOCOMPLETE_OPTIONS, triggerRange.query) : EMPTY_OPTIONS),
    [triggerRange],
  );
  const rangeKey = triggerRange ? `${triggerRange.start}:${triggerRange.end}:${triggerRange.query}` : undefined;
  const dropdownOpen = isFocused && !disabled && !readOnly && Boolean(triggerRange) && dismissedRangeKey !== rangeKey;

  useResetDismissedRange(rangeKey, dismissedRangeKey, setDismissedRangeKey);
  useClampActiveIndex(filteredOptions.length, setActiveIndex);
  useRestoreCaret(fieldRef, pendingCaretPosition, value, setCaretPosition, setPendingCaretPosition);

  return {
    activeIndex,
    filteredOptions,
    dropdownOpen,
    fieldRef,
    syncCaretPosition: (target) => setCaretPosition(target.selectionStart ?? target.value.length),
    handleChange: (target) => handleFieldChange(target, onChange, setCaretPosition, setDismissedRangeKey),
    handleKeyDown: (event) => handleAutocompleteKeyDown({
      event,
      dropdownOpen,
      filteredOptions,
      activeIndex,
      rangeKey,
      value,
      triggerRange,
      onChange,
      setActiveIndex,
      setDismissedRangeKey,
      setPendingCaretPosition,
    }),
    handleOptionMouseDown: (event) => event.preventDefault(),
    handleOptionClick: (option) => commitVariableOption({
      option,
      triggerRange,
      value,
      onChange,
      setActiveIndex,
      setDismissedRangeKey,
      setPendingCaretPosition,
    }),
    handleFocus: () => setIsFocused(true),
    handleBlur: () => setIsFocused(false),
  };
}

function handleFieldChange(
  target: FieldElement,
  onChange: (value: string) => void,
  setCaretPosition: Dispatch<SetStateAction<number>>,
  setDismissedRangeKey: Dispatch<SetStateAction<string | undefined>>,
) {
  onChange(target.value);
  setCaretPosition(target.selectionStart ?? target.value.length);
  setDismissedRangeKey(undefined);
}

function handleAutocompleteKeyDown(input: {
  event: KeyboardEvent<FieldElement>;
  dropdownOpen: boolean;
  filteredOptions: WorkflowAutocompleteOption[];
  activeIndex: number;
  rangeKey?: string;
  value: string;
  triggerRange: VariableTriggerRange | undefined;
  onChange: (value: string) => void;
  setActiveIndex: Dispatch<SetStateAction<number>>;
  setDismissedRangeKey: Dispatch<SetStateAction<string | undefined>>;
  setPendingCaretPosition: Dispatch<SetStateAction<number | undefined>>;
}) {
  const { event, dropdownOpen, filteredOptions, activeIndex, rangeKey, value, triggerRange, onChange, setActiveIndex, setDismissedRangeKey, setPendingCaretPosition } = input;
  if (!dropdownOpen || filteredOptions.length === 0) {
    if (event.key === 'Escape') {
      setDismissedRangeKey(undefined);
    }
    return;
  }
  if (event.key === 'ArrowDown') {
    event.preventDefault();
    setActiveIndex((current) => (current + 1) % filteredOptions.length);
    return;
  }
  if (event.key === 'ArrowUp') {
    event.preventDefault();
    setActiveIndex((current) => (current - 1 + filteredOptions.length) % filteredOptions.length);
    return;
  }
  if (event.key === 'Enter' || event.key === 'Tab') {
    event.preventDefault();
    commitVariableOption({
      option: filteredOptions[activeIndex] ?? filteredOptions[0],
      triggerRange,
      value,
      onChange,
      setActiveIndex,
      setDismissedRangeKey,
      setPendingCaretPosition,
    });
    return;
  }
  if (event.key === 'Escape') {
    event.preventDefault();
    if (rangeKey) {
      setDismissedRangeKey(rangeKey);
    }
  }
}

function commitVariableOption(input: {
  option: WorkflowAutocompleteOption;
  triggerRange: VariableTriggerRange | undefined;
  value: string;
  onChange: (value: string) => void;
  setActiveIndex: Dispatch<SetStateAction<number>>;
  setDismissedRangeKey: Dispatch<SetStateAction<string | undefined>>;
  setPendingCaretPosition: Dispatch<SetStateAction<number | undefined>>;
}) {
  const { option, triggerRange, value, onChange, setActiveIndex, setDismissedRangeKey, setPendingCaretPosition } = input;
  if (!triggerRange) {
    return;
  }
  const result = applyVariableOption(value, triggerRange, option);
  onChange(result.value);
  setActiveIndex(0);
  setDismissedRangeKey(undefined);
  setPendingCaretPosition(result.caretPosition);
}

function useResetDismissedRange(
  rangeKey: string | undefined,
  dismissedRangeKey: string | undefined,
  setDismissedRangeKey: Dispatch<SetStateAction<string | undefined>>,
) {
  useEffect(() => {
    if (!rangeKey || !dismissedRangeKey || dismissedRangeKey === rangeKey) {
      return;
    }
    setDismissedRangeKey(undefined);
  }, [dismissedRangeKey, rangeKey, setDismissedRangeKey]);
}

function useClampActiveIndex(optionCount: number, setActiveIndex: Dispatch<SetStateAction<number>>) {
  useEffect(() => {
    setActiveIndex((current) => {
      if (optionCount === 0) {
        return 0;
      }
      return current >= optionCount ? 0 : current;
    });
  }, [optionCount, setActiveIndex]);
}

function useRestoreCaret(
  fieldRef: RefObject<FieldElement>,
  pendingCaretPosition: number | undefined,
  value: string,
  setCaretPosition: Dispatch<SetStateAction<number>>,
  setPendingCaretPosition: Dispatch<SetStateAction<number | undefined>>,
) {
  useEffect(() => {
    if (pendingCaretPosition === undefined) {
      return;
    }
    const field = fieldRef.current;
    if (!field) {
      return;
    }
    field.focus();
    field.setSelectionRange(pendingCaretPosition, pendingCaretPosition);
    setCaretPosition(pendingCaretPosition);
    setPendingCaretPosition(undefined);
  }, [fieldRef, pendingCaretPosition, setCaretPosition, setPendingCaretPosition, value]);
}
