'use client';
import type { ReactNode } from 'react';
import { CloseButton } from '@/components/CloseButton';
import type { WebLocale } from '@/lib/i18n/locale';
import {
  CLICK_COORDINATE_SOURCE_MANUAL,
  type ClickEditorState,
} from '@/components/workflow/workflowClickStepEditorHelpers';
import {
  ClickEditorCoordinateFields,
  ClickEditorPositionTypeField,
} from '@/components/workflow/workflowClickStepEditorFields';
interface WorkflowClickStepEditorModalViewProps {
  locale: WebLocale;
  closeAria: string;
  titleID: string;
  title: string;
  stepTag: string;
  errorText: string;
  state: ClickEditorState;
  canUseFindIconReference: boolean;
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
  manualCoordinates: string;
  findIconReference: string;
  findIconUnavailable: string;
  findIconReferenceHint: string;
  pickMouse: string;
  captureReady: string;
  captureHint: string;
  coordinateHint: string;
}

interface ClickEditorFormFieldsProps {
  canUseFindIconReference: boolean;
  captureActive: boolean;
  captureLoading: boolean;
  errorText: string;
  state: ClickEditorState;
  text: ClickEditorText;
  onStartCapture: () => void;
  onStateChange: (next: ClickEditorState) => void;
}

interface ClickEditorStatePatchProps {
  state: ClickEditorState;
  onStatePatch: (patch: Partial<ClickEditorState>) => void;
}

export function WorkflowClickStepEditorModalView(props: WorkflowClickStepEditorModalViewProps) {
  const text = viewText(props.locale);

  return (
    <ClickEditorShell closeAria={props.closeAria} onClose={props.onClose} titleID={props.titleID}>
      <section className="relative z-[1] flex w-full max-w-[420px] flex-col bg-white shadow-[0_24px_64px_rgba(0,0,0,0.24)]">
        <ClickEditorHeader
          closeAria={props.closeAria}
          onClose={props.onClose}
          titleID={props.titleID}
          title={props.title}
          stepTag={props.stepTag}
        />
        <ClickEditorFormFields
          canUseFindIconReference={props.canUseFindIconReference}
          captureActive={props.captureActive}
          captureLoading={props.captureLoading}
          errorText={props.errorText}
          state={props.state}
          text={text}
          onStartCapture={props.onStartCapture}
          onStateChange={props.onStateChange}
        />
        <ClickEditorFooter
          text={text}
          saving={props.saving}
          captureActive={props.captureActive}
          onClose={props.onClose}
          onSave={props.onSave}
        />
      </section>
    </ClickEditorShell>
  );
}

function ClickEditorFormFields(props: ClickEditorFormFieldsProps) {
  const onStatePatch = (patch: Partial<ClickEditorState>) => props.onStateChange({ ...props.state, ...patch });

  return (
    <div className="space-y-6 p-6">
      <ClickEditorPositionTypeSection state={props.state} onStatePatch={onStatePatch} />
      <ClickEditorCoordinateFields
        coordinateSource={props.state.coordinateSource}
        canUseFindIconReference={props.canUseFindIconReference}
        x={props.state.x}
        y={props.state.y}
        captureActive={props.captureActive}
        captureLoading={props.captureLoading}
        text={props.text}
        onCoordinateSourceChange={(coordinateSource) => onStatePatch({ coordinateSource })}
        onStartCapture={props.onStartCapture}
        onXChange={(x) => onStatePatch({ x })}
        onYChange={(y) => onStatePatch({ y })}
      />
      <ClickEditorErrorText errorText={props.errorText} />
    </div>
  );
}

function ClickEditorPositionTypeSection(props: ClickEditorStatePatchProps) {
  if (props.state.coordinateSource !== CLICK_COORDINATE_SOURCE_MANUAL) {
    return null;
  }
  return (
    <ClickEditorPositionTypeField
      value={props.state.positionType}
      onChange={(positionType) => props.onStatePatch({ positionType })}
    />
  );
}

function ClickEditorErrorText(props: { errorText: string }) {
  if (!props.errorText) {
    return null;
  }
  return <p className="text-[11px] text-red-600">{props.errorText}</p>;
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
      <CloseButton
        onClick={onClose}
        className="shrink-0"
        aria-label={closeAria}
      />
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
      <CloseButton
        onClick={onClose}
        aria-label={text.cancel}
      />
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

function viewText(locale: WebLocale): ClickEditorText {
  if (locale === 'zh-CN') {
    return {
      cancel: '取消',
      save: '保存设置',
      saving: '保存中…',
      manualCoordinates: '手动填写坐标',
      findIconReference: '使用最近一次 find_icon 坐标',
      findIconUnavailable: '前面还没有可引用的 find_icon 步骤。',
      findIconReferenceHint: '会在运行时读取当前工具内最近一次成功 find_icon 的首个匹配中心点。',
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
    manualCoordinates: 'Enter coordinates manually',
    findIconReference: 'Use the latest find_icon coordinates',
    findIconUnavailable: 'No earlier find_icon step is available in this composer.',
    findIconReferenceHint: 'At runtime this reads the first match center from the latest successful find_icon step in the same tool.',
    pickMouse: 'Pick Mouse Position',
    captureReady: 'Press Enter to Confirm',
    captureHint: 'Tracking the cursor. Press Enter to confirm or Esc to cancel.',
    coordinateHint: 'Required · number · pixels (px). Relative mode offsets from the live cursor position at runtime. Picking the cursor switches to absolute coordinates.',
  };
}
