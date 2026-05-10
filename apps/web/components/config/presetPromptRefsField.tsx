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
    <>
      <div className="space-y-4">
        <h3 className="text-sm font-medium text-gray-700">{copy.settings.presetsPromptRefsLabel}</h3>
        <div className="grid grid-cols-1 gap-5 sm:grid-cols-2">
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
      </div>
      <PresetContextPromptRefs
        draft={draft}
        options={contextOptions}
        controlsDisabled={controlsDisabled}
        onChange={onChange}
      />
    </>
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
    <div className="space-y-2">
      <label className="block text-[13px] font-medium text-gray-500">
        {presetPromptSlotLabel(copy.settings, slot)}
      </label>
      <div className="relative">
        <select
          data-testid={`preset-prompt-ref-${slot}-select`}
          value={currentValue}
          disabled={controlsDisabled}
          onChange={(event) => {
            onChange({
              prompt_refs: updateSinglePromptRef(draft.prompt_refs, slot, event.target.value),
            });
          }}
          className="w-full appearance-none rounded-xl border border-gray-200 bg-gray-50/50 px-4 py-3 pr-10 text-sm text-gray-900 outline-none transition-all focus:border-gray-400 focus:bg-white focus:ring-2 focus:ring-gray-900/5 disabled:cursor-not-allowed disabled:opacity-60"
        >
          <option value="">{copy.settings.presetsPromptRefEmpty}</option>
          {options.map((item) => (
            <option key={item.id} value={item.id}>
              {item.name}
            </option>
          ))}
        </select>
        <span className="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-4 text-gray-400" aria-hidden="true">
          v
        </span>
      </div>
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
    <div className="space-y-3">
      <div>
        <h3 className="text-sm font-medium text-gray-700">{copy.settings.presetsContextPromptRefsLabel}</h3>
        <p className="mt-1 text-[13px] text-gray-400">{copy.settings.presetsContextPromptRefsDescription}</p>
      </div>
      {options.length === 0 ? (
        <div className="flex items-center justify-center rounded-xl border border-dashed border-gray-200 bg-gray-50/50 px-4 py-6 text-[13px] text-gray-400">
          {copy.settings.presetsContextPromptRefsEmpty}
        </div>
      ) : (
        <div className="grid gap-2 sm:grid-cols-2">
          {options.map((item) => (
            <label
              key={item.id}
              className="flex items-center gap-2 rounded-xl border border-gray-200 bg-gray-50/50 px-4 py-3 text-[13px] text-gray-700"
            >
              <input
                type="checkbox"
                data-testid={`preset-prompt-ref-context-${item.id}`}
                checked={draft.prompt_refs.context?.includes(item.id) ?? false}
                disabled={controlsDisabled}
                className="h-4 w-4 accent-gray-900 disabled:cursor-not-allowed"
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
      )}
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
