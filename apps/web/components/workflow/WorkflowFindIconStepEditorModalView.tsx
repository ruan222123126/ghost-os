'use client';
import { type ChangeEvent, type ReactNode, type RefObject } from 'react';
export type FindIconTestResult = 'idle' | 'success' | 'failure';
export interface FindIconEditorPanelState {
  templatePath: string;
  templateName: string;
  hoverAfterMatch: boolean;
}
export interface FindIconEditorPreview {
  url: string;
  revocable: boolean;
}
export interface FindIconEditorShellProps {
  closeAria: string;
  titleID: string;
  onClose: () => void;
  children: ReactNode;
}
export interface FindIconEditorPanelProps {
  closeAria: string;
  titleID: string;
  stepTag: string;
  state: FindIconEditorPanelState;
  uploading: boolean;
  saving: boolean;
  testing: boolean;
  testResult: FindIconTestResult;
  errorText: string;
  preview: FindIconEditorPreview | null;
  fileInputRef: RefObject<HTMLInputElement>;
  onClose: () => void;
  onPickAction: (hover: boolean) => void;
  onSave: () => void;
  onUpload: (event: ChangeEvent<HTMLInputElement>) => void;
  onRemoveImage: () => void;
  onTest: () => void;
}
interface FindIconImageFieldProps {
  state: FindIconEditorPanelState;
  uploading: boolean;
  testing: boolean;
  testResult: FindIconTestResult;
  preview: FindIconEditorPreview | null;
  fileInputRef: RefObject<HTMLInputElement>;
  onUpload: (event: ChangeEvent<HTMLInputElement>) => void;
  onRemoveImage: () => void;
  onTest: () => void;
}
interface FindIconActionFieldProps {
  hoverAfterMatch: boolean;
  onPickAction: (hover: boolean) => void;
}
export function FindIconEditorShell(props: FindIconEditorShellProps) {
  const { closeAria, titleID, onClose, children } = props;
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
export function FindIconEditorPanel(props: FindIconEditorPanelProps) {
  const {
    closeAria,
    titleID,
    stepTag,
    state,
    uploading,
    saving,
    testing,
    testResult,
    errorText,
    preview,
    fileInputRef,
    onClose,
    onPickAction,
    onSave,
    onUpload,
    onRemoveImage,
    onTest,
  } = props;
  return (
    <section className="relative z-[1] flex w-full max-w-[400px] flex-col bg-white shadow-[0_24px_64px_rgba(0,0,0,0.24)]">
      <FindIconEditorHeader
        closeAria={closeAria}
        titleID={titleID}
        stepTag={stepTag}
        onClose={onClose}
      />
      <div className="space-y-7 p-6">
        <FindIconImageField
          state={state}
          uploading={uploading}
          testing={testing}
          testResult={testResult}
          preview={preview}
          fileInputRef={fileInputRef}
          onUpload={onUpload}
          onRemoveImage={onRemoveImage}
          onTest={onTest}
        />
        <FindIconActionField
          hoverAfterMatch={state.hoverAfterMatch}
          onPickAction={onPickAction}
        />
        {errorText ? <p className="text-[11px] text-red-600">{errorText}</p> : null}
      </div>
      <FindIconEditorFooter saving={saving} onClose={onClose} onSave={onSave} />
    </section>
  );
}
function FindIconEditorHeader(props: {
  closeAria: string;
  titleID: string;
  stepTag: string;
  onClose: () => void;
}) {
  const { closeAria, titleID, stepTag, onClose } = props;
  return (
    <header className="flex items-start justify-between border-b border-gray-100 px-6 py-6">
      <div>
        <h4 id={titleID} className="mb-1 text-2xl font-black italic tracking-wide text-black">
          图像查找配置
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
function FindIconEditorFooter(props: {
  saving: boolean;
  onClose: () => void;
  onSave: () => void;
}) {
  const { saving, onClose, onSave } = props;
  return (
    <footer className="flex justify-end gap-3 bg-white px-6 pb-6 pt-2">
      <button
        type="button"
        onClick={onClose}
        className="px-5 py-2.5 text-xs font-bold text-gray-400 transition-colors hover:text-black"
      >
        取消
      </button>
      <button
        type="button"
        onClick={onSave}
        disabled={saving}
        className="border-2 border-black bg-white px-8 py-2.5 text-xs font-black text-black shadow-sm transition-all hover:bg-black hover:text-white disabled:cursor-not-allowed disabled:opacity-45"
      >
        {saving ? '保存中…' : '保存配置'}
      </button>
    </footer>
  );
}
function FindIconImageField(props: FindIconImageFieldProps) {
  const { state, uploading, testing, testResult, preview, fileInputRef, onUpload, onRemoveImage, onTest } = props;
  const hasTemplate = state.templatePath.trim().length > 0;
  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <label className="block text-xs font-medium text-gray-700">目标图像</label>
        <div className="flex items-center justify-end gap-2">
          <span className={`min-w-8 text-[11px] font-bold ${testResultClassName(testResult)}`}>
            {testResultText(testResult)}
          </span>
          <button
            type="button"
            onClick={onTest}
            disabled={testing}
            className="border border-gray-300 bg-white px-3 py-1.5 text-[11px] font-bold text-gray-700 transition-colors hover:border-black hover:text-black disabled:cursor-not-allowed disabled:opacity-45"
          >
            {testing ? '识别中…' : '测试'}
          </button>
        </div>
      </div>
      <input
        type="file"
        ref={fileInputRef}
        onChange={onUpload}
        accept="image/*"
        disabled={uploading}
        className="hidden"
      />
      <FindIconImageCard
        hasTemplate={hasTemplate}
        previewURL={preview?.url}
        templateName={state.templateName}
        templatePath={state.templatePath}
        uploading={uploading}
        onOpenFile={() => fileInputRef.current?.click()}
        onRemoveImage={onRemoveImage}
      />
      <p className="text-[11px] text-gray-500">
        <span className="text-gray-800">必填</span> · 图像 · 屏幕中需要进行匹配的参考图.
      </p>
    </div>
  );
}
function FindIconImageCard(props: {
  hasTemplate: boolean;
  previewURL: string | undefined;
  templateName: string;
  templatePath: string;
  uploading: boolean;
  onOpenFile: () => void;
  onRemoveImage: () => void;
}) {
  const { hasTemplate, previewURL, templateName, templatePath, uploading, onOpenFile, onRemoveImage } = props;
  const className = hasTemplate
    ? 'relative h-36 w-full border border-gray-200 bg-gray-50'
    : 'relative h-36 w-full cursor-pointer border border-dashed border-gray-300 bg-gray-50/50 transition-all hover:border-black hover:bg-gray-100';
  return (
    <div onClick={() => (!hasTemplate && !uploading ? onOpenFile() : undefined)} className={className}>
      {hasTemplate ? (
        <>
          {previewURL ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={previewURL} alt="预览图" className="absolute inset-0 h-full w-full object-contain p-4" />
          ) : (
            <div className="flex h-full flex-col items-center justify-center px-4 text-center">
              <span className="text-sm font-bold text-gray-700">{templateName || '已上传模板'}</span>
              <span className="mt-1 break-all text-[10px] text-gray-400">{templatePath}</span>
            </div>
          )}
          <button
            type="button"
            onClick={onRemoveImage}
            className="absolute right-2 top-2 z-10 border border-gray-200 bg-white px-2 py-1 text-[10px] font-bold text-gray-400 shadow-sm transition-all hover:border-black hover:text-black"
          >
            移除图片
          </button>
        </>
      ) : (
        <div className="pointer-events-none flex h-full flex-col items-center justify-center text-gray-500">
          <span className="text-sm font-bold tracking-tight">{uploading ? '上传中…' : '选择图像'}</span>
          <span className="mt-1 text-[10px] text-gray-400">仅支持 PNG, JPG</span>
        </div>
      )}
    </div>
  );
}
function FindIconActionField(props: FindIconActionFieldProps) {
  const { hoverAfterMatch, onPickAction } = props;
  return (
    <div className="space-y-2">
      <label className="block text-xs font-medium text-gray-700">匹配成功后动作</label>
      <div className="flex gap-2">
        <FindIconActionButton active={!hoverAfterMatch} onClick={() => onPickAction(false)}>仅返回坐标</FindIconActionButton>
        <FindIconActionButton active={hoverAfterMatch} onClick={() => onPickAction(true)}>悬停鼠标</FindIconActionButton>
      </div>
      <p className="text-[11px] text-gray-500">
        <span className="text-gray-800">必填</span> · 布尔值 · 决定查找到图像后是否进行物理鼠标位移.
      </p>
    </div>
  );
}
function FindIconActionButton(props: { active: boolean; onClick: () => void; children: ReactNode }) {
  const { active, onClick, children } = props;
  const className = active
    ? 'flex-1 border-2 border-black bg-white py-3 text-xs font-black tracking-tight text-black transition-all'
    : 'flex-1 border-2 border-transparent bg-gray-50 py-3 text-xs tracking-tight text-gray-400 transition-all hover:bg-gray-100';
  return <button type="button" onClick={onClick} className={className}>{children}</button>;
}
function testResultText(result: FindIconTestResult): string {
  if (result === 'success') {
    return '成功';
  }
  if (result === 'failure') {
    return '失败';
  }
  return '';
}
function testResultClassName(result: FindIconTestResult): string {
  if (result === 'success') {
    return 'text-emerald-600';
  }
  if (result === 'failure') {
    return 'text-rose-600';
  }
  return 'text-transparent';
}
function CloseIcon() {
  return <svg width="20" height="20" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="M18 6 6 18M6 6l12 12" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" /></svg>;
}
