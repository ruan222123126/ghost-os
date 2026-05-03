'use client';

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
  onDelete: () => void;
  onSave: () => void;
  onChange: (patch: Partial<PresetEditorDraft>) => void;
}) {
  const { copy } = useWebLocale();
  const { editor, tools, promptLibrary, controlsDisabled, onClose, onDelete, onSave, onChange } = props;
  const createMode = editor.mode === 'create';
  const title = createMode ? copy.settings.presetsCreateTitle : copy.settings.presetsEditTitle;
  const saveLabel = createMode ? copy.settings.create : copy.settings.save;
  const dirty = !presetDraftEqual(editor.original, editor.draft);
  const nameInvalid = editor.draft.name.trim() === '';

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/45 px-4 py-6" onClick={onClose}>
      <div
        className="w-full max-w-3xl rounded-[20px] border border-[#E5E5E5] bg-white p-6 shadow-2xl"
        onClick={(event) => event.stopPropagation()}
      >
        <PresetEditorHeader title={title} onClose={onClose} />
        <PresetEditorFields
          draft={editor.draft}
          tools={tools}
          promptLibrary={promptLibrary}
          controlsDisabled={controlsDisabled}
          onChange={onChange}
        />
        <PresetEditorActions
          createMode={createMode}
          controlsDisabled={controlsDisabled}
          saveDisabled={nameInvalid || (!createMode && !dirty)}
          saveLabel={saveLabel}
          onSave={onSave}
          onDelete={onDelete}
        />
      </div>
    </div>
  );
}

function PresetEditorHeader(props: { title: string; onClose: () => void }) {
  const { copy } = useWebLocale();
  const { title, onClose } = props;

  return (
    <div className="mb-4 flex items-center justify-between gap-3">
      <h3 data-testid="preset-editor-title" className="text-[18px] font-semibold text-[#111111]">{title}</h3>
      <button
        type="button"
        onClick={onClose}
        className="rounded-full border border-[#E5E5E5] px-3 py-1.5 text-[12px] font-medium text-[#111111] transition-colors hover:bg-[#F5F5F5]"
      >
        {copy.settings.cancel}
      </button>
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
    <>
      <PresetNameField {...props} />
      <PresetToolField {...props} />
      <PresetPromptRefsField {...props} />
    </>
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
    <div className="mb-3">
      <label className="mb-1 block text-[12px] font-medium text-[#525252]">{copy.settings.presetsNameLabel}</label>
      <input
        type="text"
        data-testid="preset-name-input"
        value={draft.name}
        disabled={controlsDisabled}
        onChange={(event) => {
          onChange({ name: event.target.value });
        }}
        className="w-full rounded-[10px] border border-[#E5E5E5] bg-white px-3 py-2 text-[13px] text-[#111111] transition-colors focus:border-[#111111] focus:outline-none disabled:opacity-60"
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
    <div className="mb-4">
      <div className="mb-1 text-[12px] font-medium text-[#525252]">{copy.settings.presetsToolAllowlistLabel}</div>
      <p className="mb-2 text-[12px] text-[#737373]">{copy.settings.presetsToolAllowlistDescription}</p>
      <div className="grid gap-2 sm:grid-cols-2">
        {tools.map((tool) => (
          <label
            key={tool.name}
            className="flex items-center gap-2 rounded-[10px] border border-[#E5E5E5] bg-[#FAFAFA] px-3 py-2 text-[13px] text-[#111111]"
          >
            <input
              type="checkbox"
              data-testid={`preset-tool-${tool.name}`}
              checked={draft.tool_allowlist.includes(tool.name)}
              disabled={controlsDisabled}
              onChange={(event) => {
                onChange({
                  tool_allowlist: togglePresetToolSelection(
                    draft.tool_allowlist,
                    tool.name,
                    event.target.checked,
                  ),
                });
              }}
            />
            <span className="truncate">{tool.name}</span>
          </label>
        ))}
      </div>
    </div>
  );
}

function PresetEditorActions(props: {
  createMode: boolean;
  controlsDisabled: boolean;
  saveDisabled: boolean;
  saveLabel: string;
  onSave: () => void;
  onDelete: () => void;
}) {
  const { copy } = useWebLocale();
  const { createMode, controlsDisabled, saveDisabled, saveLabel, onSave, onDelete } = props;

  return (
    <div className="flex items-center gap-2">
      <button
        type="button"
        data-testid="preset-save"
        disabled={controlsDisabled || saveDisabled}
        onClick={onSave}
        className="rounded-full bg-[#111111] px-4 py-1.5 text-[12px] font-medium text-white transition-colors hover:bg-[#333333] disabled:cursor-not-allowed disabled:opacity-50"
      >
        {saveLabel}
      </button>
      {createMode ? null : (
        <button
          type="button"
          data-testid="preset-delete"
          disabled={controlsDisabled}
          onClick={onDelete}
          className="rounded-full border border-[#E5E5E5] px-4 py-1.5 text-[12px] font-medium text-[#111111] transition-colors hover:bg-[#F5F5F5] disabled:cursor-not-allowed disabled:opacity-50"
        >
          {copy.settings.presetsDelete}
        </button>
      )}
    </div>
  );
}
