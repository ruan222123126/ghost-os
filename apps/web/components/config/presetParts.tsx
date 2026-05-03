'use client';

import { useWebLocale } from '@/lib/i18n/provider';
import type {
  PresetPayload,
  PresetPromptRefs,
} from '@/lib/types';
import {
  PRESET_PROMPT_REF_FIELDS,
} from './presetHelpers';

export function PresetCard(props: {
  preset: PresetPayload;
  index: number;
  promptNameByID: Record<string, string>;
  active: boolean;
  activating: boolean;
  controlsDisabled: boolean;
  onActivate: () => void;
  onEdit: () => void;
  onDelete: () => void;
}) {
  const { copy } = useWebLocale();
  const {
    preset,
    index,
    promptNameByID,
    active,
    activating,
    controlsDisabled,
    onActivate,
    onEdit,
    onDelete,
  } = props;

  return (
    <article
      data-testid={`preset-card-${index}`}
      className="rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] p-4"
    >
      <div className="mb-3 flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="mb-2 flex flex-wrap items-center gap-2">
            <span className="inline-flex rounded-full bg-white px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider text-[#111111]">
              {copy.settings.presetsToolCount(preset.tool_allowlist.length)}
            </span>
            {active ? (
              <span className="inline-flex rounded-full bg-[#111111] px-2 py-0.5 text-[10px] font-bold tracking-wider text-white">
                {copy.settings.presetsActive}
              </span>
            ) : null}
          </div>
          <h3 className="truncate text-[15px] font-semibold text-[#111111]">{preset.name}</h3>
          <p className="mt-1 text-[12px] text-[#737373]">
            {renderPromptSummary(
              preset.prompt_refs,
              promptNameByID,
              copy.settings.presetsPromptUnbound,
            )}
          </p>
        </div>
        <div className="flex shrink-0 items-center gap-2 whitespace-nowrap">
          <button
            type="button"
            data-testid={`preset-card-activate-${index}`}
            disabled={controlsDisabled || active}
            onClick={onActivate}
            className="whitespace-nowrap rounded-full bg-[#111111] px-3 py-1.5 text-[12px] font-medium text-white transition-colors hover:bg-[#333333] disabled:cursor-not-allowed disabled:bg-[#D4D4D4]"
          >
            {active ? copy.settings.presetsActive : (activating ? copy.settings.presetsActivating : copy.settings.presetsActivate)}
          </button>
          <button
            type="button"
            data-testid={`preset-card-edit-${index}`}
            disabled={controlsDisabled}
            onClick={onEdit}
            className="whitespace-nowrap rounded-full border border-[#E5E5E5] px-3 py-1.5 text-[12px] font-medium text-[#111111] transition-colors hover:bg-white disabled:cursor-not-allowed disabled:opacity-50"
          >
            {copy.settings.edit}
          </button>
          <button
            type="button"
            data-testid={`preset-card-delete-${index}`}
            disabled={controlsDisabled}
            onClick={onDelete}
            className="whitespace-nowrap rounded-full border border-[#E5E5E5] px-3 py-1.5 text-[12px] font-medium text-[#111111] transition-colors hover:bg-white disabled:cursor-not-allowed disabled:opacity-50"
          >
            {copy.settings.presetsDelete}
          </button>
        </div>
      </div>
    </article>
  );
}

export function LoadingPresetsNotice(props: { text: string }) {
  return (
    <div className="mb-4 rounded-[12px] border border-[#E5E5E5] bg-[#FAFAFA] px-4 py-3 text-[13px] text-[#737373]">
      {props.text}
    </div>
  );
}

export function PresetEmptyNotice() {
  const { copy } = useWebLocale();

  return (
    <p className="mb-3 text-[13px] text-[#737373]">
      {copy.settings.presetsEmpty}
    </p>
  );
}

function renderPromptSummary(
  promptRefs: PresetPromptRefs,
  promptNameByID: Record<string, string>,
  unboundLabel: string,
): string {
  const summary = PRESET_PROMPT_REF_FIELDS.map((field) => {
    const refID = promptRefs[field.slot];
    const promptName = refID ? (promptNameByID[refID] ?? refID) : unboundLabel;
    return `${field.slot}: ${promptName}`;
  });
  summary.push(`context: ${renderContextPromptSummary(promptRefs.context, promptNameByID, unboundLabel)}`);
  return summary.join(' · ');
}

function renderContextPromptSummary(
  contextRefs: string[] | undefined,
  promptNameByID: Record<string, string>,
  unboundLabel: string,
): string {
  if (!contextRefs || contextRefs.length === 0) {
    return unboundLabel;
  }
  return contextRefs.map((refID) => promptNameByID[refID] ?? refID).join(', ');
}
