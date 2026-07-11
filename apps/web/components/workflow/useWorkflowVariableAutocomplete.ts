'use client';

import type { Dispatch, KeyboardEvent, MouseEvent, RefObject, SetStateAction } from 'react';
import { useEffect, useMemo, useRef, useState } from 'react';
import {
  WORKFLOW_AUTOCOMPLETE_OPTIONS,
  applyVariableOption,
  filterWorkflowVariableOptions,
  resolveVariableAutocompleteKeyAction,
  resolveVariableTriggerRange,
  type VariableAutocompleteKeyAction,
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

interface AutocompleteControlState {
  activeIndex: number;
  caretPosition: number;
  dismissedRangeKey?: string;
  fieldRef: RefObject<FieldElement>;
  isFocused: boolean;
  pendingCaretPosition?: number;
  setActiveIndex: Dispatch<SetStateAction<number>>;
  setCaretPosition: Dispatch<SetStateAction<number>>;
  setDismissedRangeKey: Dispatch<SetStateAction<string | undefined>>;
  setIsFocused: Dispatch<SetStateAction<boolean>>;
  setPendingCaretPosition: Dispatch<SetStateAction<number | undefined>>;
}

interface AutocompleteDerivedState {
  dropdownOpen: boolean;
  filteredOptions: WorkflowAutocompleteOption[];
  rangeKey?: string;
  triggerRange?: VariableTriggerRange;
}

interface AutocompleteHandlers {
  handleBlur: () => void;
  handleChange: (target: FieldElement) => void;
  handleFocus: () => void;
  handleKeyDown: (event: KeyboardEvent<FieldElement>) => void;
  handleOptionClick: (option: WorkflowAutocompleteOption) => void;
  handleOptionMouseDown: (event: MouseEvent<HTMLButtonElement>) => void;
  syncCaretPosition: (target: FieldElement) => void;
}

export function useWorkflowVariableAutocomplete(
  props: WorkflowVariableAutocompleteProps,
): WorkflowVariableAutocompleteState {
  const { value, disabled = false, readOnly = false, onChange } = props;
  const controls = useAutocompleteControlState(value);
  const derived = useAutocompleteDerivedState({
    caretPosition: controls.caretPosition,
    disabled,
    dismissedRangeKey: controls.dismissedRangeKey,
    isFocused: controls.isFocused,
    readOnly,
    value,
  });
  useAutocompleteLifecycle(controls, derived, value);
  return createAutocompleteState({ controls, derived, onChange, value });
}

function useAutocompleteControlState(value: string): AutocompleteControlState {
  const fieldRef = useRef<FieldElement | null>(null);
  const [activeIndex, setActiveIndex] = useState(0);
  const [caretPosition, setCaretPosition] = useState(value.length);
  const [dismissedRangeKey, setDismissedRangeKey] = useState<string>();
  const [isFocused, setIsFocused] = useState(false);
  const [pendingCaretPosition, setPendingCaretPosition] = useState<number>();

  return {
    activeIndex,
    caretPosition,
    dismissedRangeKey,
    fieldRef,
    isFocused,
    pendingCaretPosition,
    setActiveIndex,
    setCaretPosition,
    setDismissedRangeKey,
    setIsFocused,
    setPendingCaretPosition,
  };
}

function useAutocompleteDerivedState(options: {
  caretPosition: number;
  disabled: boolean;
  dismissedRangeKey?: string;
  isFocused: boolean;
  readOnly: boolean;
  value: string;
}): AutocompleteDerivedState {
  const triggerRange = useMemo(
    () => resolveVariableTriggerRange(options.value, options.caretPosition),
    [options.caretPosition, options.value],
  );
  const filteredOptions = useMemo(() => resolveFilteredOptions(triggerRange), [triggerRange]);
  const rangeKey = formatRangeKey(triggerRange);

  return {
    dropdownOpen: isAutocompleteDropdownOpen({ ...options, rangeKey, triggerRange }),
    filteredOptions,
    rangeKey,
    triggerRange,
  };
}

function useAutocompleteLifecycle(
  controls: AutocompleteControlState,
  derived: AutocompleteDerivedState,
  value: string,
): void {
  useResetDismissedRange({
    dismissedRangeKey: controls.dismissedRangeKey,
    rangeKey: derived.rangeKey,
    setDismissedRangeKey: controls.setDismissedRangeKey,
  });
  useClampActiveIndex(derived.filteredOptions.length, controls.setActiveIndex);
  useRestoreCaret({
    fieldRef: controls.fieldRef,
    pendingCaretPosition: controls.pendingCaretPosition,
    setCaretPosition: controls.setCaretPosition,
    setPendingCaretPosition: controls.setPendingCaretPosition,
    value,
  });
}

function createAutocompleteState(options: {
  controls: AutocompleteControlState;
  derived: AutocompleteDerivedState;
  onChange: (value: string) => void;
  value: string;
}): WorkflowVariableAutocompleteState {
  const { controls, derived, onChange, value } = options;
  return {
    activeIndex: controls.activeIndex,
    filteredOptions: derived.filteredOptions,
    dropdownOpen: derived.dropdownOpen,
    fieldRef: controls.fieldRef,
    ...createAutocompleteHandlers({ controls, derived, onChange, value }),
  };
}

function resolveFilteredOptions(triggerRange?: VariableTriggerRange): WorkflowAutocompleteOption[] {
  if (!triggerRange) {
    return EMPTY_OPTIONS;
  }
  return filterWorkflowVariableOptions(WORKFLOW_AUTOCOMPLETE_OPTIONS, triggerRange.query);
}

function formatRangeKey(triggerRange?: VariableTriggerRange): string | undefined {
  if (!triggerRange) {
    return undefined;
  }
  return `${triggerRange.start}:${triggerRange.end}:${triggerRange.query}`;
}

function isAutocompleteDropdownOpen(options: {
  disabled: boolean;
  dismissedRangeKey?: string;
  isFocused: boolean;
  rangeKey?: string;
  readOnly: boolean;
  triggerRange?: VariableTriggerRange;
}): boolean {
  return options.isFocused
    && !options.disabled
    && !options.readOnly
    && options.triggerRange !== undefined
    && options.dismissedRangeKey !== options.rangeKey;
}

function createAutocompleteHandlers(options: {
  controls: AutocompleteControlState;
  derived: AutocompleteDerivedState;
  onChange: (value: string) => void;
  value: string;
}): AutocompleteHandlers {
  const { controls, derived, onChange, value } = options;
  return {
    handleBlur: createFocusStateHandler(controls.setIsFocused, false),
    handleChange: createHandleChange({ onChange, setCaretPosition: controls.setCaretPosition, setDismissedRangeKey: controls.setDismissedRangeKey }),
    handleFocus: createFocusStateHandler(controls.setIsFocused, true),
    handleKeyDown: createHandleKeyDown({ controls, derived, onChange, value }),
    handleOptionClick: createHandleOptionClick({ controls, derived, onChange, value }),
    handleOptionMouseDown: preventOptionMouseDown,
    syncCaretPosition: createSyncCaretPosition(controls.setCaretPosition),
  };
}

function createSyncCaretPosition(
  setCaretPosition: Dispatch<SetStateAction<number>>,
): (target: FieldElement) => void {
  return function syncCaretPosition(target) {
    setCaretPosition(getFieldCaretPosition(target));
  };
}

function createHandleChange(options: {
  onChange: (value: string) => void;
  setCaretPosition: Dispatch<SetStateAction<number>>;
  setDismissedRangeKey: Dispatch<SetStateAction<string | undefined>>;
}): (target: FieldElement) => void {
  return function handleChange(target) {
    handleFieldChange({ ...options, target });
  };
}

function createHandleKeyDown(options: {
  controls: AutocompleteControlState;
  derived: AutocompleteDerivedState;
  onChange: (value: string) => void;
  value: string;
}): (event: KeyboardEvent<FieldElement>) => void {
  const { controls, derived, onChange, value } = options;
  return function handleKeyDown(event) {
    handleAutocompleteKeyDown({
      event,
      dropdownOpen: derived.dropdownOpen,
      filteredOptions: derived.filteredOptions,
      activeIndex: controls.activeIndex,
      rangeKey: derived.rangeKey,
      value,
      triggerRange: derived.triggerRange,
      onChange,
      setActiveIndex: controls.setActiveIndex,
      setDismissedRangeKey: controls.setDismissedRangeKey,
      setPendingCaretPosition: controls.setPendingCaretPosition,
    });
  };
}

function createHandleOptionClick(options: {
  controls: AutocompleteControlState;
  derived: AutocompleteDerivedState;
  onChange: (value: string) => void;
  value: string;
}): (option: WorkflowAutocompleteOption) => void {
  const { controls, derived, onChange, value } = options;
  return function handleOptionClick(option) {
    commitVariableOption({
      option,
      triggerRange: derived.triggerRange,
      value,
      onChange,
      setActiveIndex: controls.setActiveIndex,
      setDismissedRangeKey: controls.setDismissedRangeKey,
      setPendingCaretPosition: controls.setPendingCaretPosition,
    });
  };
}

function createFocusStateHandler(
  setIsFocused: Dispatch<SetStateAction<boolean>>,
  value: boolean,
): () => void {
  return function handleFocusState() {
    setIsFocused(value);
  };
}

function preventOptionMouseDown(event: MouseEvent<HTMLButtonElement>): void {
  event.preventDefault();
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
  const action = resolveVariableAutocompleteKeyAction({
    dropdownOpen: input.dropdownOpen,
    key: input.event.key,
    optionCount: input.filteredOptions.length,
  });
  if (!action) {
    return;
  }
  runAutocompleteKeyAction(input, action);
}

function runAutocompleteKeyAction(
  input: Parameters<typeof handleAutocompleteKeyDown>[0],
  action: VariableAutocompleteKeyAction,
) {
  if (action === 'clear-dismissed') {
    input.setDismissedRangeKey(undefined);
    return;
  }
  input.event.preventDefault();
  if (action === 'move-next') {
    moveActiveIndex(input.setActiveIndex, input.filteredOptions.length, 1);
    return;
  }
  if (action === 'move-previous') {
    moveActiveIndex(input.setActiveIndex, input.filteredOptions.length, -1);
    return;
  }
  if (action === 'commit') {
    commitActiveVariableOption(input);
    return;
  }
  if (input.rangeKey) {
    input.setDismissedRangeKey(input.rangeKey);
  }
}

function commitActiveVariableOption(input: Parameters<typeof handleAutocompleteKeyDown>[0]): void {
  const option = input.filteredOptions[input.activeIndex] ?? input.filteredOptions[0];
  if (!option) {
    return;
  }
  commitVariableOption({
    option,
    triggerRange: input.triggerRange,
    value: input.value,
    onChange: input.onChange,
    setActiveIndex: input.setActiveIndex,
    setDismissedRangeKey: input.setDismissedRangeKey,
    setPendingCaretPosition: input.setPendingCaretPosition,
  });
}

function moveActiveIndex(
  setActiveIndex: Dispatch<SetStateAction<number>>,
  optionCount: number,
  offset: number,
) {
  setActiveIndex((current) => (current + offset + optionCount) % optionCount);
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

function handleFieldChange(options: {
  target: FieldElement;
  onChange: (value: string) => void;
  setCaretPosition: Dispatch<SetStateAction<number>>;
  setDismissedRangeKey: Dispatch<SetStateAction<string | undefined>>;
}) {
  const { target, onChange, setCaretPosition, setDismissedRangeKey } = options;
  onChange(target.value);
  setCaretPosition(getFieldCaretPosition(target));
  setDismissedRangeKey(undefined);
}

function useResetDismissedRange(options: {
  rangeKey: string | undefined,
  dismissedRangeKey: string | undefined,
  setDismissedRangeKey: Dispatch<SetStateAction<string | undefined>>,
}) {
  const { rangeKey, dismissedRangeKey, setDismissedRangeKey } = options;
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

function useRestoreCaret(options: {
  fieldRef: RefObject<FieldElement>,
  pendingCaretPosition: number | undefined,
  value: string,
  setCaretPosition: Dispatch<SetStateAction<number>>,
  setPendingCaretPosition: Dispatch<SetStateAction<number | undefined>>,
}) {
  const {
    fieldRef,
    pendingCaretPosition,
    setCaretPosition,
    setPendingCaretPosition,
    value,
  } = options;
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

function getFieldCaretPosition(target: FieldElement): number {
  return target.selectionStart ?? target.value.length;
}
