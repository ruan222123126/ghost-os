'use client';

import { useWebLocale } from '@/lib/i18n/provider';
import type { PresetPromptRefs, PromptLibraryItem } from '@/lib/types';
import type { PresetEditorDraft, PresetPromptRefSlot } from './presetHelpers';
import {
  PRESET_PROMPT_REF_FIELDS,
  presetPromptSlotLabel,
  togglePresetContextSelection,
} from './presetHelpers';

interface PresetPromptRefsFieldProps {
  draft: PresetEditorDraft;
  promptLibrary: PromptLibraryItem[];
  controlsDisabled: boolean;
  onChange: (patch: Partial<PresetEditorDraft>) => void;
}

export function PresetPromptRefsField(props: PresetPromptRefsFieldProps) {
  const { copy } = useWebLocale();
  const { draft, promptLibrary, controlsDisabled, onChange } = props;
  const contextOptions = promptLibrary.filter((item) => item.insert_point === 'context');

  return (
    <div className="mb-4">
      <div className="mb-2 text-[12px] font-medium text-[#525252]">{copy.settings.presetsPromptRefsLabel}</div>
      <div className="mb-3 grid gap-3 sm:grid-cols-3">
        {PRESET_PROMPT_REF_FIELDS.map((field) => (
          <PresetPromptRefSelect
            key={field.slot}
            slot={field.slot}
            draft={draft}
            promptLibrary={promptLibrary}
            controlsDisabled={controlsDisabled}
            onChange={onChange}
          />
        ))}
      </div>
      <PresetContextPromptRefs
        draft={draft}
        options={contextOptions}
        controlsDisabled={controlsDisabled}
        onChange={onChange}
      />
    </div>
  );
}

function PresetPromptRefSelect(props: {
  slot: PresetPromptRefSlot;
  draft: PresetEditorDraft;
  promptLibrary: PromptLibraryItem[];
  controlsDisabled: boolean;
  onChange: (patch: Partial<PresetEditorDraft>) => void;
}) {
  const { copy } = useWebLocale();
  const { slot, draft, promptLibrary, controlsDisabled, onChange } = props;
  const options = promptLibrary.filter((item) => item.insert_point === slot);
  const currentValue = draft.prompt_refs[slot] ?? '';

  return (
    <div>
      <label className="mb-1 block text-[12px] font-medium text-[#525252]">
        {presetPromptSlotLabel(copy.settings, slot)}
      </label>
      <select
        data-testid={`preset-prompt-ref-${slot}-select`}
        value={currentValue}
        disabled={controlsDisabled}
        onChange={(event) => {
          onChange({
            prompt_refs: updateSinglePromptRef(draft.prompt_refs, slot, event.target.value),
          });
        }}
        className="w-full rounded-[10px] border border-[#E5E5E5] bg-white px-3 py-2 text-[13px] text-[#111111] transition-colors focus:border-[#111111] focus:outline-none disabled:opacity-60"
      >
        <option value="">{copy.settings.presetsPromptRefEmpty}</option>
        {options.map((item) => (
          <option key={item.id} value={item.id}>
            {item.name}
          </option>
        ))}
      </select>
    </div>
  );
}

function PresetContextPromptRefs(props: {
  draft: PresetEditorDraft;
  options: PromptLibraryItem[];
  controlsDisabled: boolean;
  onChange: (patch: Partial<PresetEditorDraft>) => void;
}) {
  const { copy } = useWebLocale();
  const { draft, options, controlsDisabled, onChange } = props;

  return (
    <div>
      <div className="mb-1 text-[12px] font-medium text-[#525252]">
        {copy.settings.presetsContextPromptRefsLabel}
      </div>
      <p className="mb-2 text-[12px] text-[#737373]">{copy.settings.presetsContextPromptRefsDescription}</p>
      <div className="grid gap-2 sm:grid-cols-2">
        {options.map((item) => (
          <label
            key={item.id}
            className="flex items-center gap-2 rounded-[10px] border border-[#E5E5E5] bg-[#FAFAFA] px-3 py-2 text-[13px] text-[#111111]"
          >
            <input
              type="checkbox"
              data-testid={`preset-prompt-ref-context-${item.id}`}
              checked={draft.prompt_refs.context?.includes(item.id) ?? false}
              disabled={controlsDisabled}
              onChange={(event) => {
                onChange({
                  prompt_refs: updateContextPromptRefs(draft.prompt_refs, item.id, event.target.checked),
                });
              }}
            />
            <span className="truncate">{item.name}</span>
          </label>
        ))}
      </div>
    </div>
  );
}

function updateSinglePromptRef(
  promptRefs: PresetPromptRefs,
  slot: PresetPromptRefSlot,
  nextValue: string,
): PresetPromptRefs {
  return {
    ...promptRefs,
    [slot]: nextValue || undefined,
  };
}

function updateContextPromptRefs(
  promptRefs: PresetPromptRefs,
  promptID: string,
  selected: boolean,
): PresetPromptRefs {
  return {
    ...promptRefs,
    context: togglePresetContextSelection(promptRefs.context, promptID, selected),
  };
}
