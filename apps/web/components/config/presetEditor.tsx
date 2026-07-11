'use client';

import { BodyPortal } from '@/components/BodyPortal';
import { CloseButton } from '@/components/CloseButton';
import { useWebLocale } from '@/lib/i18n/provider';
import type { PromptLibraryItem, ToolPayload } from '@/lib/types';
import type {
  PresetEditorDraft,
  PresetEditorState,
} from './presetHelpers';
import {
  presetDraftEqual,
  togglePresetToolSelection,
} from './presetHelpers';
import { PresetPromptRefsField } from './presetPromptRefsField';

export function PresetEditor(props: {
  editor: PresetEditorState;
  tools: ToolPayload[];
  promptLibrary: PromptLibraryItem[];
  controlsDisabled: boolean;
  onClose: () => void;
  onSave: () => void;
  onChange: (patch: Partial<PresetEditorDraft>) => void;
}) {
  const { copy } = useWebLocale();
  const { editor, tools, promptLibrary, controlsDisabled, onClose, onSave, onChange } = props;
  const createMode = editor.mode === 'create';
  const title = createMode ? copy.settings.presetsCreateTitle : copy.settings.presetsEditTitle;
  const saveLabel = createMode ? copy.settings.create : copy.settings.save;
  const dirty = !presetDraftEqual(editor.original, editor.draft);
  const nameInvalid = editor.draft.name.trim() === '';

  return (
    <BodyPortal>
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 sm:p-6">
      <button
        type="button"
        aria-label={copy.settings.cancel}
        className="absolute inset-0 cursor-default bg-gray-900/20 backdrop-blur-[2px] transition-opacity"
        onClick={onClose}
      />
      <div className="relative flex max-h-[90vh] w-full max-w-3xl flex-col overflow-hidden rounded-[24px] bg-white shadow-2xl shadow-gray-900/10">
        <PresetEditorHeader title={title} onClose={onClose} />
        <PresetEditorFields
          draft={editor.draft}
          tools={tools}
          promptLibrary={promptLibrary}
          controlsDisabled={controlsDisabled}
          onChange={onChange}
        />
        <PresetEditorActions
          controlsDisabled={controlsDisabled}
          saveDisabled={nameInvalid || (!createMode && !dirty)}
          saveLabel={saveLabel}
          onClose={onClose}
          onSave={onSave}
        />
      </div>
    </div>
    </BodyPortal>
  );
}

function PresetEditorHeader(props: { title: string; onClose: () => void }) {
  const { copy } = useWebLocale();
  const { title, onClose } = props;

  return (
    <div className="flex items-center justify-between border-b border-gray-100/80 px-8 py-5">
      <h2 data-testid="preset-editor-title" className="text-lg font-semibold tracking-tight text-gray-900">
        {title}
      </h2>
      <CloseButton
        aria-label={copy.settings.cancel}
        onClick={onClose}
        className="-mr-2"
      />
    </div>
  );
}

function PresetEditorFields(props: {
  draft: PresetEditorDraft;
  tools: ToolPayload[];
  promptLibrary: PromptLibraryItem[];
  controlsDisabled: boolean;
  onChange: (patch: Partial<PresetEditorDraft>) => void;
}) {
  return (
    <div className="preset-editor-scrollbar flex-1 overflow-y-auto px-8 py-6">
      <div className="space-y-9">
        <PresetNameField {...props} />
        <PresetToolField {...props} />
        <PresetPromptRefsField {...props} />
      </div>
    </div>
  );
}

function PresetNameField(props: {
  draft: PresetEditorDraft;
  controlsDisabled: boolean;
  onChange: (patch: Partial<PresetEditorDraft>) => void;
}) {
  const { copy } = useWebLocale();
  const { draft, controlsDisabled, onChange } = props;

  return (
    <div className="space-y-2">
      <label htmlFor="preset-name-input" className="block text-sm font-medium text-gray-700">
        {copy.settings.presetsNameLabel}
      </label>
      <input
        type="text"
        id="preset-name-input"
        data-testid="preset-name-input"
        value={draft.name}
        disabled={controlsDisabled}
        onChange={(event) => {
          onChange({ name: event.target.value });
        }}
        className="w-full rounded-xl border border-gray-200 bg-gray-50/50 px-4 py-3 text-gray-900 outline-none transition-all placeholder:text-gray-400 focus:border-gray-400 focus:bg-white focus:ring-2 focus:ring-gray-900/5 disabled:cursor-not-allowed disabled:opacity-60"
      />
    </div>
  );
}

function PresetToolField(props: {
  draft: PresetEditorDraft;
  tools: ToolPayload[];
  controlsDisabled: boolean;
  onChange: (patch: Partial<PresetEditorDraft>) => void;
}) {
  const { copy } = useWebLocale();
  const { draft, tools, controlsDisabled, onChange } = props;

  return (
    <div className="space-y-4">
      <div>
        <h3 className="text-sm font-medium text-gray-700">{copy.settings.presetsToolAllowlistLabel}</h3>
        <p className="mt-1 text-[13px] text-gray-400">{copy.settings.presetsToolAllowlistDescription}</p>
      </div>
      <div className="flex flex-wrap gap-2.5">
        {tools.map((tool) => {
          const selected = draft.tool_allowlist.includes(tool.name);
          return (
            <button
              key={tool.name}
              type="button"
              data-testid={`preset-tool-${tool.name}`}
              aria-pressed={selected}
              disabled={controlsDisabled}
              onClick={() => {
                onChange({
                  tool_allowlist: togglePresetToolSelection(draft.tool_allowlist, tool.name, !selected),
                });
              }}
              className={toolButtonClassName(selected)}
            >
              {tool.name}
            </button>
          );
        })}
      </div>
    </div>
  );
}

function toolButtonClassName(selected: boolean): string {
  const base = 'select-none rounded-full px-4 py-2 text-[13px] font-medium transition-all duration-200 disabled:cursor-not-allowed disabled:opacity-50';
  if (selected) {
    return `${base} bg-gray-900 text-white shadow-md shadow-gray-900/10`;
  }
  return `${base} border border-gray-200 bg-white text-gray-600 hover:border-gray-300 hover:bg-gray-50 hover:text-gray-900`;
}

function PresetEditorActions(props: {
  controlsDisabled: boolean;
  saveDisabled: boolean;
  saveLabel: string;
  onClose: () => void;
  onSave: () => void;
}) {
  const { copy } = useWebLocale();
  const { controlsDisabled, saveDisabled, saveLabel, onClose, onSave } = props;

  return (
    <div className="flex items-center justify-end border-t border-gray-100/80 bg-white px-8 py-5">
      <div className="flex items-center gap-3">
        <CloseButton
          disabled={controlsDisabled}
          onClick={onClose}
          className="disabled:cursor-not-allowed disabled:opacity-50"
          aria-label={copy.settings.cancel}
        />
        <button
          type="button"
          data-testid="preset-save"
          disabled={controlsDisabled || saveDisabled}
          onClick={onSave}
          className="rounded-full bg-gray-900 px-8 py-2.5 text-sm font-medium text-white shadow-md shadow-gray-900/10 transition-all hover:bg-black disabled:cursor-not-allowed disabled:opacity-50"
        >
          {saveLabel}
        </button>
      </div>
    </div>
  );
}
