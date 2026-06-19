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
      className="settings-config-card"
    >
      <div className="settings-config-card-main">
        <div className="settings-card-badges">
          <span className="settings-card-badge">
            {copy.settings.presetsToolCount(preset.tool_allowlist.length)}
          </span>
          {active ? (
            <span className="settings-card-badge">
              {copy.settings.presetsActive}
            </span>
          ) : null}
        </div>
        <h3 className="settings-card-title is-strong truncate">{preset.name}</h3>
        <p className="settings-card-meta line-clamp-2">
          {renderPromptSummary(
            preset.prompt_refs,
            promptNameByID,
            copy.settings.presetsPromptUnbound,
          )}
        </p>
      </div>
      <div className="settings-config-card-actions">
        <button
          type="button"
          data-testid={`preset-card-activate-${index}`}
          disabled={controlsDisabled || active}
          onClick={onActivate}
          className="settings-card-action whitespace-nowrap"
        >
          {active ? copy.settings.presetsActive : (activating ? copy.settings.presetsActivating : copy.settings.presetsActivate)}
        </button>
        <button
          type="button"
          data-testid={`preset-card-edit-${index}`}
          disabled={controlsDisabled}
          onClick={onEdit}
          className="settings-card-action whitespace-nowrap"
        >
          {copy.settings.edit}
        </button>
        <button
          type="button"
          data-testid={`preset-card-delete-${index}`}
          disabled={controlsDisabled}
          onClick={onDelete}
          className="settings-card-action is-danger whitespace-nowrap"
        >
          {copy.settings.presetsDelete}
        </button>
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
