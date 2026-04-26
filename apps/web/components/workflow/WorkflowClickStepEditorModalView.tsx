'use client';
import type { ReactNode } from 'react';
import type { WebLocale } from '@/lib/i18n/locale';
import {
  POSITION_TYPE_ABSOLUTE,
  POSITION_TYPE_RELATIVE,
  type ClickEditorState,
  type ClickPositionType,
} from '@/components/workflow/workflowClickStepEditorHelpers';
interface WorkflowClickStepEditorModalViewProps {
  locale: WebLocale;
  closeAria: string;
  titleID: string;
  title: string;
  stepTag: string;
  errorText: string;
  state: ClickEditorState;
  saving: boolean;
  captureActive: boolean;
  captureLoading: boolean;
  onClose: () => void;
  onSave: () => void;
  onStartCapture: () => void;
  onStateChange: (next: ClickEditorState) => void;
}
interface ClickEditorShellProps {
  closeAria: string;
  onClose: () => void;
  titleID: string;
  children: ReactNode;
}
interface ClickEditorText {
  cancel: string;
  save: string;
  saving: string;
  pickMouse: string;
  captureReady: string;
  captureHint: string;
  coordinateHint: string;
}
export function WorkflowClickStepEditorModalView(props: WorkflowClickStepEditorModalViewProps) {
  const {
    locale,
    closeAria,
    titleID,
    title,
    stepTag,
    errorText,
    state,
    saving,
    captureActive,
    captureLoading,
    onClose,
    onSave,
    onStartCapture,
    onStateChange,
  } = props;
  const text = viewText(locale);

  return (
    <ClickEditorShell closeAria={closeAria} onClose={onClose} titleID={titleID}>
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
            text={text}
            x={state.x}
            y={state.y}
            captureActive={captureActive}
            captureLoading={captureLoading}
            onStartCapture={onStartCapture}
            onXChange={(value) => onStateChange({ ...state, x: value })}
            onYChange={(value) => onStateChange({ ...state, y: value })}
          />
          {errorText ? <p className="text-[11px] text-red-600">{errorText}</p> : null}
        </div>
        <ClickEditorFooter
          text={text}
          saving={saving}
          captureActive={captureActive}
          onClose={onClose}
          onSave={onSave}
        />
      </section>
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

function ClickEditorHeader(props: {
  closeAria: string;
  onClose: () => void;
  titleID: string;
  title: string;
  stepTag: string;
}) {
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

function ClickEditorFooter(props: {
  text: ClickEditorText;
  saving: boolean;
  captureActive: boolean;
  onClose: () => void;
  onSave: () => void;
}) {
  const { text, saving, captureActive, onClose, onSave } = props;
  return (
    <footer className="flex justify-end gap-3 bg-white px-6 pb-6 pt-2">
      <button type="button" onClick={onClose} className="px-5 py-2.5 text-sm text-gray-600 transition-colors hover:bg-gray-100">
        {text.cancel}
      </button>
      <button
        type="button"
        onClick={onSave}
        disabled={saving || captureActive}
        className="border-2 border-black bg-white px-6 py-2.5 text-sm font-medium text-black shadow-sm transition-all hover:bg-black hover:text-white disabled:cursor-not-allowed disabled:opacity-45"
      >
        {saving ? text.saving : text.save}
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
          onChange={(event) => onChange(event.target.value === POSITION_TYPE_ABSOLUTE ? POSITION_TYPE_ABSOLUTE : POSITION_TYPE_RELATIVE)}
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
  text: ClickEditorText;
  x: string;
  y: string;
  captureActive: boolean;
  captureLoading: boolean;
  onStartCapture: () => void;
  onXChange: (value: string) => void;
  onYChange: (value: string) => void;
}) {
  const {
    text,
    x,
    y,
    captureActive,
    captureLoading,
    onStartCapture,
    onXChange,
    onYChange,
  } = props;
  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-3">
        <label className="block text-xs font-medium text-gray-700">坐标参数</label>
        <button
          type="button"
          onClick={onStartCapture}
          disabled={captureActive || captureLoading}
          className="border border-gray-300 bg-white px-3 py-1.5 text-[11px] font-bold text-gray-700 transition-colors hover:border-black hover:text-black disabled:cursor-not-allowed disabled:opacity-45"
        >
          {captureActive ? text.captureReady : text.pickMouse}
        </button>
      </div>
      <div className="grid grid-cols-2 gap-3">
        <CoordinateInput axis="X" value={x} onChange={onXChange} />
        <CoordinateInput axis="Y" value={y} onChange={onYChange} />
      </div>
      <p className="text-[11px] text-gray-500">
        {captureActive ? text.captureHint : text.coordinateHint}
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

function viewText(locale: WebLocale): ClickEditorText {
  if (locale === 'zh-CN') {
    return {
      cancel: '取消',
      save: '保存设置',
      saving: '保存中…',
      pickMouse: '获取鼠标位置',
      captureReady: '按 Enter 确认',
      captureHint: '正在追踪鼠标位置，按 Enter 确认，按 Esc 取消。',
      coordinateHint: '必填 · number · 单位为像素(px)。相对模式会在运行时基于当前鼠标位置偏移；点击右侧按钮会自动切换为绝对坐标。',
    };
  }
  return {
    cancel: 'Cancel',
    save: 'Save',
    saving: 'Saving…',
    pickMouse: 'Pick Mouse Position',
    captureReady: 'Press Enter to Confirm',
    captureHint: 'Tracking the cursor. Press Enter to confirm or Esc to cancel.',
    coordinateHint: 'Required · number · pixels (px). Relative mode offsets from the live cursor position at runtime. Picking the cursor switches to absolute coordinates.',
  };
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
