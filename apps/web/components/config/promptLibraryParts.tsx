'use client';

import { useWebLocale } from '@/lib/i18n/provider';
import type { PromptLibraryItem } from '@/lib/types';
import {
  labelForInsertPoint,
  promptLibraryCardEqual,
  promptInsertPointOptionsForEditor,
  type PromptLibraryEditorState,
} from './promptLibraryHelpers';

export function PromptLibraryCard(props: {
  item: PromptLibraryItem;
  index: number;
  controlsDisabled: boolean;
  onToggle: () => void;
  onEdit: () => void;
  onDelete: () => void;
}) {
  const { copy } = useWebLocale();
  const { item, index, controlsDisabled, onToggle, onEdit, onDelete } = props;

  return (
    <article
      data-testid={`prompt-library-card-${index}`}
      className="rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] p-4"
    >
      <div className="mb-3 flex items-center justify-between gap-3">
        <div className="min-w-0">
          <div className="mb-1 flex items-center gap-2">
            <span className="inline-flex rounded-full bg-white px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider text-[#111111]">
              {item.active ? copy.settings.promptsLibraryActive : copy.settings.promptsLibraryInactive}
            </span>
            <span className="inline-flex rounded-full bg-white px-2 py-0.5 font-mono text-[10px] text-[#525252]">
              {labelForInsertPoint(item.insert_point, copy.settings)}
            </span>
          </div>
          <h3 className="truncate text-[15px] font-semibold text-[#111111]">{item.name}</h3>
          <p className="mt-1 line-clamp-2 text-[12px] text-[#737373]">{item.content || copy.settings.promptsLibraryEmptyContent}</p>
        </div>
        <div className="flex shrink-0 items-center gap-2 whitespace-nowrap">
          <button
            type="button"
            data-testid={`prompt-library-toggle-${index}`}
            disabled={controlsDisabled}
            onClick={onToggle}
            className="whitespace-nowrap rounded-full border border-[#E5E5E5] px-3 py-1.5 text-[12px] font-medium text-[#111111] transition-colors hover:bg-white disabled:cursor-not-allowed disabled:opacity-50"
          >
            {item.active ? copy.settings.promptsLibraryDeactivate : copy.settings.promptsLibraryActivate}
          </button>
          <button
            type="button"
            data-testid={`prompt-library-edit-${index}`}
            disabled={controlsDisabled}
            onClick={onEdit}
            className="whitespace-nowrap rounded-full border border-[#E5E5E5] px-3 py-1.5 text-[12px] font-medium text-[#111111] transition-colors hover:bg-white disabled:cursor-not-allowed disabled:opacity-50"
          >
            {copy.settings.edit}
          </button>
          <button
            type="button"
            data-testid={`prompt-library-delete-${index}`}
            disabled={controlsDisabled}
            onClick={onDelete}
            className="whitespace-nowrap rounded-full border border-[#E5E5E5] px-3 py-1.5 text-[12px] font-medium text-[#111111] transition-colors hover:bg-white disabled:cursor-not-allowed disabled:opacity-50"
          >
            {copy.settings.promptsLibraryDelete}
          </button>
        </div>
      </div>
    </article>
  );
}

export function PromptLibraryEditor(props: {
  editor: PromptLibraryEditorState;
  controlsDisabled: boolean;
  onClose: () => void;
  onDelete: () => void;
  onSave: () => void;
  onChange: (patch: Partial<PromptLibraryItem>) => void;
}) {
  const { copy } = useWebLocale();
  const { editor, controlsDisabled, onClose, onDelete, onSave, onChange } = props;
  const createMode = editor.mode === 'create';
  const editorTitle = createMode ? copy.settings.promptsLibraryCreateTitle : copy.settings.promptsLibraryEditTitle;
  const saveLabel = createMode ? copy.settings.create : copy.settings.save;
  const dirty = !promptLibraryCardEqual(editor.original, editor.draft);
  const nameInvalid = editor.draft.name.trim() === '';

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/45 px-4 py-6" onClick={onClose}>
      <div
        className="w-full max-w-3xl rounded-[20px] border border-[#E5E5E5] bg-white p-6 shadow-2xl"
        onClick={(event) => event.stopPropagation()}
      >
        <PromptLibraryEditorHeader title={editorTitle} onClose={onClose} />
        <PromptLibraryEditorFields
          draft={editor.draft}
          controlsDisabled={controlsDisabled}
          onChange={onChange}
        />
        <PromptLibraryEditorActions
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

function PromptLibraryEditorHeader(props: { title: string; onClose: () => void }) {
  const { copy } = useWebLocale();
  const { title, onClose } = props;

  return (
    <div className="mb-4 flex items-center justify-between gap-3">
      <h3 data-testid="prompt-card-editor-title" className="text-[18px] font-semibold text-[#111111]">{title}</h3>
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

function PromptLibraryEditorFields(props: {
  draft: PromptLibraryItem;
  controlsDisabled: boolean;
  onChange: (patch: Partial<PromptLibraryItem>) => void;
}) {
  return (
    <>
      <PromptLibraryNameField {...props} />
      <PromptLibraryInsertPointField {...props} />
      <PromptLibraryContentField {...props} />
    </>
  );
}

function PromptLibraryNameField(props: {
  draft: PromptLibraryItem;
  controlsDisabled: boolean;
  onChange: (patch: Partial<PromptLibraryItem>) => void;
}) {
  const { copy } = useWebLocale();
  const { draft, controlsDisabled, onChange } = props;

  return (
    <div className="mb-3">
      <label className="mb-1 block text-[12px] font-medium text-[#525252]">{copy.settings.promptsLibraryNameLabel}</label>
      <input
        type="text"
        data-testid="prompt-card-name-input"
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

function PromptLibraryInsertPointField(props: {
  draft: PromptLibraryItem;
  controlsDisabled: boolean;
  onChange: (patch: Partial<PromptLibraryItem>) => void;
}) {
  const { copy } = useWebLocale();
  const { draft, controlsDisabled, onChange } = props;
  const options = promptInsertPointOptionsForEditor(draft.insert_point);

  return (
    <div className="mb-3">
      <label className="mb-1 block text-[12px] font-medium text-[#525252]">{copy.settings.promptsLibraryInsertPointLabel}</label>
      <select
        data-testid="prompt-card-insert-point-select"
        value={draft.insert_point}
        disabled={controlsDisabled}
        onChange={(event) => {
          onChange({ insert_point: event.target.value as PromptLibraryItem['insert_point'] });
        }}
        className="w-full rounded-[10px] border border-[#E5E5E5] bg-white px-3 py-2 text-[13px] text-[#111111] transition-colors focus:border-[#111111] focus:outline-none disabled:opacity-60"
      >
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {copy.settings[option.labelKey]}
          </option>
        ))}
      </select>
    </div>
  );
}

function PromptLibraryContentField(props: {
  draft: PromptLibraryItem;
  controlsDisabled: boolean;
  onChange: (patch: Partial<PromptLibraryItem>) => void;
}) {
  const { copy } = useWebLocale();
  const { draft, controlsDisabled, onChange } = props;

  return (
    <div className="mb-3">
      <label className="mb-1 block text-[12px] font-medium text-[#525252]">{copy.settings.promptsLibraryContentLabel}</label>
      <textarea
        data-testid="prompt-card-content-input"
        value={draft.content}
        rows={12}
        disabled={controlsDisabled}
        onChange={(event) => {
          onChange({ content: event.target.value });
        }}
        className="w-full rounded-[10px] border border-[#E5E5E5] bg-white px-3 py-2 font-mono text-[13px] leading-6 text-[#111111] transition-colors focus:border-[#111111] focus:outline-none disabled:opacity-60"
      />
    </div>
  );
}

function PromptLibraryEditorActions(props: {
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
        data-testid="prompt-card-save"
        disabled={controlsDisabled || saveDisabled}
        onClick={onSave}
        className="rounded-full bg-[#111111] px-4 py-1.5 text-[12px] font-medium text-white transition-colors hover:bg-[#333333] disabled:cursor-not-allowed disabled:opacity-50"
      >
        {saveLabel}
      </button>
      {createMode ? null : (
        <button
          type="button"
          data-testid="prompt-card-delete"
          disabled={controlsDisabled}
          onClick={onDelete}
          className="rounded-full border border-[#E5E5E5] px-4 py-1.5 text-[12px] font-medium text-[#111111] transition-colors hover:bg-[#F5F5F5] disabled:cursor-not-allowed disabled:opacity-50"
        >
          {copy.settings.promptsLibraryDelete}
        </button>
      )}
    </div>
  );
}

export function LoadingPromptsNotice(props: { text: string }) {
  return (
    <div className="mb-4 rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-3 text-[13px] text-[#737373]">
      {props.text}
    </div>
  );
}

export function PromptLibraryEmptyNotice() {
  const { copy } = useWebLocale();
  return (
    <p className="mb-3 text-[13px] text-[#737373]">
      {copy.settings.promptsLibraryEmpty}
    </p>
  );
}
