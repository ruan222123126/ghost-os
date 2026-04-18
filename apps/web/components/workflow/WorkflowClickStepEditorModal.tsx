'use client';

import { useEffect, useMemo, useState, type ReactNode } from 'react';
import { createPortal } from 'react-dom';
import { toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import {
  normalizeClickComposerParams,
  withClickComposerParams,
  type ScreenControlComposerStep,
} from '@/lib/workflow-editor';

interface WorkflowClickStepEditorModalProps {
  open: boolean;
  stepIndex: number;
  step: ScreenControlComposerStep;
  onClose: () => void;
  onSave: (stepIndex: number, step: ScreenControlComposerStep) => void;
}

const CLICK_POSITION_TYPE_KEY = 'position_type';
const POSITION_TYPE_RELATIVE = 'relative';
const POSITION_TYPE_ABSOLUTE = 'absolute';

type ClickPositionType = typeof POSITION_TYPE_RELATIVE | typeof POSITION_TYPE_ABSOLUTE;

interface ClickEditorState {
  x: string;
  y: string;
  positionType: ClickPositionType;
}

interface ClickEditorShellProps {
  closeAria: string;
  onClose: () => void;
  titleID: string;
  children: ReactNode;
}

interface ClickEditorPanelProps {
  closeAria: string;
  onClose: () => void;
  titleID: string;
  title: string;
  stepTag: string;
  errorText: string;
  state: ClickEditorState;
  saving: boolean;
  onStateChange: (next: ClickEditorState) => void;
  onSave: () => void;
}

interface ClickEditorHeaderProps {
  closeAria: string;
  onClose: () => void;
  titleID: string;
  title: string;
  stepTag: string;
}

interface ClickEditorFooterProps {
  saving: boolean;
  onClose: () => void;
  onSave: () => void;
}

export function WorkflowClickStepEditorModal(props: WorkflowClickStepEditorModalProps) {
  const { open } = props;
  const [mounted, setMounted] = useState(false);

  useEffect(() => setMounted(true), []);
  if (!mounted || !open) {
    return null;
  }

  return createPortal(<WorkflowClickStepEditorModalContent {...props} />, document.body);
}

function WorkflowClickStepEditorModalContent(props: WorkflowClickStepEditorModalProps) {
  const { copy } = useWebLocale();
  const { step, stepIndex, onClose, onSave } = props;
  const initial = useMemo(() => buildInitialEditorState(step), [step]);
  const [state, setState] = useState<ClickEditorState>(initial);
  const [saving, setSaving] = useState(false);
  const [errorText, setErrorText] = useState('');
  const titleID = 'workflow-screen-click-editor-title';

  useEffect(() => {
    setState(initial);
    setErrorText('');
  }, [initial]);

  const saveClick = () =>
    handleSaveClick({
      step,
      stepIndex,
      state,
      onSave,
      setSaving,
      setErrorText,
      fallbackError: copy.system.genericRequestFailed,
    });

  return (
    <ClickEditorShell
      closeAria={copy.workflow.clickEditorCloseAria}
      onClose={onClose}
      titleID={titleID}
    >
      <ClickEditorPanel
        closeAria={copy.workflow.clickEditorCloseAria}
        onClose={onClose}
        titleID={titleID}
        title={copy.workflow.clickEditorTitle}
        stepTag={buildStepTag(stepIndex)}
        errorText={errorText}
        state={state}
        saving={saving}
        onStateChange={setState}
        onSave={saveClick}
      />
    </ClickEditorShell>
  );
}

function ClickEditorShell(props: ClickEditorShellProps) {
  const { closeAria, onClose, titleID, children } = props;
  return (
    <div className="fixed inset-0 z-[76] flex items-center justify-center p-4" role="dialog" aria-modal="true" aria-labelledby={titleID}>
      <button
        type="button"
        className="absolute inset-0 border-0 bg-black/20 backdrop-blur-[2px]"
        onClick={onClose}
        aria-label={closeAria}
      />
      {children}
    </div>
  );
}

function ClickEditorPanel(props: ClickEditorPanelProps) {
  const {
    closeAria,
    onClose,
    titleID,
    title,
    stepTag,
    errorText,
    state,
    saving,
    onStateChange,
    onSave,
  } = props;

  return (
    <section className="relative z-[1] flex w-full max-w-[420px] flex-col bg-white shadow-[0_24px_64px_rgba(0,0,0,0.24)]">
      <ClickEditorHeader
        closeAria={closeAria}
        onClose={onClose}
        titleID={titleID}
        title={title}
        stepTag={stepTag}
      />
      <div className="space-y-6 p-6">
        <PositionTypeField
          value={state.positionType}
          onChange={(value) => onStateChange({ ...state, positionType: value })}
        />
        <CoordinateFields
          x={state.x}
          y={state.y}
          onXChange={(value) => onStateChange({ ...state, x: value })}
          onYChange={(value) => onStateChange({ ...state, y: value })}
        />
        {errorText ? <p className="text-[11px] text-red-600">{errorText}</p> : null}
      </div>
      <ClickEditorFooter saving={saving} onClose={onClose} onSave={onSave} />
    </section>
  );
}

function ClickEditorHeader(props: ClickEditorHeaderProps) {
  const { closeAria, onClose, titleID, title, stepTag } = props;
  return (
    <header className="flex items-start justify-between border-b border-gray-100 px-6 py-5">
      <div>
        <h4 id={titleID} className="mb-1 text-2xl font-black italic tracking-wide text-black">
          {title}
        </h4>
        <p className="font-mono text-[10px] uppercase tracking-[0.16em] text-gray-400">{stepTag}</p>
      </div>
      <button
        type="button"
        onClick={onClose}
        className="p-1 text-gray-400 transition-colors hover:text-black"
        aria-label={closeAria}
      >
        <CloseIcon />
      </button>
    </header>
  );
}

function ClickEditorFooter(props: ClickEditorFooterProps) {
  const { saving, onClose, onSave } = props;
  return (
    <footer className="flex justify-end gap-3 bg-white px-6 pb-6 pt-2">
      <button type="button" onClick={onClose} className="px-5 py-2.5 text-sm text-gray-600 transition-colors hover:bg-gray-100">
        取消
      </button>
      <button
        type="button"
        onClick={onSave}
        disabled={saving}
        className="border-2 border-black bg-white px-6 py-2.5 text-sm font-medium text-black shadow-sm transition-all hover:bg-black hover:text-white disabled:cursor-not-allowed disabled:opacity-45"
      >
        {saving ? '保存中…' : '保存设置'}
      </button>
    </footer>
  );
}

function PositionTypeField(props: {
  value: ClickPositionType;
  onChange: (value: ClickPositionType) => void;
}) {
  const { value, onChange } = props;
  return (
    <div className="space-y-2">
      <label className="block text-xs font-medium text-gray-700">位置基准</label>
      <div className="relative">
        <select
          value={value}
          onChange={(event) => onChange(parsePositionType(event.target.value))}
          className="h-10 w-full cursor-pointer appearance-none border border-gray-200 bg-white pl-3 pr-10 text-sm outline-none transition-colors hover:border-gray-300 focus:border-black focus:ring-1 focus:ring-black"
        >
          <option value={POSITION_TYPE_RELATIVE}>相对于鼠标原本位置 (Relative)</option>
          <option value={POSITION_TYPE_ABSOLUTE}>屏幕绝对位置 (Absolute)</option>
        </select>
        <span className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-gray-500">
          <ChevronDownIcon />
        </span>
      </div>
      <p className="text-[11px] text-gray-500">
        <span className="text-gray-800">必填</span> · enum · 决定坐标系原点.
      </p>
    </div>
  );
}

function CoordinateFields(props: {
  x: string;
  y: string;
  onXChange: (value: string) => void;
  onYChange: (value: string) => void;
}) {
  const { x, y, onXChange, onYChange } = props;
  return (
    <div className="space-y-4">
      <label className="block text-xs font-medium text-gray-700">坐标参数</label>
      <div className="grid grid-cols-2 gap-3">
        <CoordinateInput axis="X" value={x} onChange={onXChange} />
        <CoordinateInput axis="Y" value={y} onChange={onYChange} />
      </div>
      <p className="text-[11px] text-gray-500">
        <span className="text-gray-800">必填</span> · number · 单位为像素(px).
      </p>
    </div>
  );
}

function CoordinateInput(props: {
  axis: 'X' | 'Y';
  value: string;
  onChange: (value: string) => void;
}) {
  const { axis, value, onChange } = props;
  return (
    <div className="flex bg-gray-50/50">
      <span className="flex w-12 flex-none items-center justify-center border border-r-0 border-gray-200 bg-gray-50 px-3 py-2 text-sm text-gray-600">
        {axis}
      </span>
      <input
        type="number"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder="0"
        className="w-full border border-gray-200 bg-white px-3 py-2 text-sm outline-none transition-colors focus:border-black focus:ring-1 focus:ring-black"
      />
    </div>
  );
}

function CloseIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <path d="M18 6 6 18M6 6l12 12" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

function ChevronDownIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <path d="m6 9 6 6 6-6" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

function buildInitialEditorState(step: ScreenControlComposerStep): ClickEditorState {
  const params = normalizeClickComposerParams(step.params);
  return {
    x: params === undefined ? '' : String(params.x),
    y: params === undefined ? '' : String(params.y),
    positionType: parsePositionType(asRecord(step.params)[CLICK_POSITION_TYPE_KEY]),
  };
}

function buildStepTag(stepIndex: number): string {
  const id = String(stepIndex + 1).padStart(4, '0');
  return `STEP ID: CLICK-${id}`;
}

function handleSaveClick(options: {
  step: ScreenControlComposerStep;
  stepIndex: number;
  state: ClickEditorState;
  onSave: (stepIndex: number, step: ScreenControlComposerStep) => void;
  setSaving: (value: boolean) => void;
  setErrorText: (value: string) => void;
  fallbackError: string;
}) {
  const { step, stepIndex, state, onSave, setSaving, setErrorText, fallbackError } = options;
  setSaving(true);
  setErrorText('');

  try {
    const baseStep = withClickComposerParams(step, {
      x: parseCoordinate(state.x, 'x'),
      y: parseCoordinate(state.y, 'y'),
    });
    const baseParams = asRecord(baseStep.params);
    onSave(stepIndex, {
      ...baseStep,
      params: {
        ...baseParams,
        [CLICK_POSITION_TYPE_KEY]: state.positionType,
      },
    });
  } catch (error) {
    setErrorText(toErrorMessage(error, fallbackError));
  } finally {
    setSaving(false);
  }
}

function parseCoordinate(raw: string, field: 'x' | 'y'): number {
  const trimmed = raw.trim();
  if (!trimmed) {
    throw new Error(`${field} is required`);
  }
  const value = Number(trimmed);
  if (!Number.isFinite(value)) {
    throw new Error(`${field} must be a finite number`);
  }
  return value;
}

function parsePositionType(raw: unknown): ClickPositionType {
  if (raw === POSITION_TYPE_ABSOLUTE) {
    return POSITION_TYPE_ABSOLUTE;
  }
  return POSITION_TYPE_RELATIVE;
}

function asRecord(input: unknown): Record<string, unknown> {
  if (typeof input !== 'object' || input === null || Array.isArray(input)) {
    return {};
  }
  return input as Record<string, unknown>;
}
