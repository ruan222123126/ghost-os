'use client';

import { useMemo, useState } from 'react';
import { ignorePromise } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type {
  BridgeConfig,
  PresetCreateRequest,
  PresetPayload,
  PresetUpdateRequest,
  SystemPromptPayload,
  ToolPayload,
} from '@/lib/types';
import {
  confirmPresetDeletion,
  findActivePresetID,
  newPresetDraft,
  normalizePresetDraft,
  presetToDraft,
  type PresetEditorState,
} from './presetHelpers';
import { PresetEditor } from './presetEditor';
import {
  LoadingPresetsNotice,
  PresetCard,
  PresetEmptyNotice,
} from './presetParts';

interface PresetSettingsSectionProps {
  config: BridgeConfig | null;
  presets: PresetPayload[];
  loading: boolean;
  saving: boolean;
  tools: ToolPayload[];
  toolsLoading: boolean;
  prompts: SystemPromptPayload | null;
  promptsLoading: boolean;
  onRefresh: () => Promise<void>;
  onCreatePreset: (input: PresetCreateRequest) => Promise<void>;
  onUpdatePreset: (id: string, input: PresetUpdateRequest) => Promise<void>;
  onDeletePreset: (id: string) => Promise<void>;
  onActivatePreset: (id: string) => Promise<void>;
}

export function PresetSettingsSection(props: PresetSettingsSectionProps) {
  const { copy } = useWebLocale();
  const {
    presets,
    loading,
    saving,
    tools,
    toolsLoading,
    prompts,
    promptsLoading,
    onRefresh,
    onCreatePreset,
    onUpdatePreset,
    onDeletePreset,
    onActivatePreset,
  } = props;
  const controlsDisabled = loading || saving;
  const editorDisabled = controlsDisabled || toolsLoading || promptsLoading;
  const promptLibrary = useMemo(() => prompts?.prompt_library ?? [], [prompts?.prompt_library]);
  const promptNameByID = useMemo(() => {
    return Object.fromEntries(promptLibrary.map((item) => [item.id, item.name]));
  }, [promptLibrary]);
  const [editor, setEditor] = useState<PresetEditorState | null>(null);
  const [activatingPresetID, setActivatingPresetID] = useState<string | null>(null);
  const activePresetID = useMemo(() => {
    return findActivePresetID(
      presets,
      tools,
      promptLibrary,
    );
  }, [presets, promptLibrary, tools]);

  const saveEditor = async () => {
    if (editor === null) {
      return;
    }

    const normalizedDraft = normalizePresetDraft(editor.draft);
    if (normalizedDraft.name === '') {
      return;
    }

    if (editor.mode === 'create') {
      await onCreatePreset(normalizedDraft);
    } else if (editor.presetID) {
      await onUpdatePreset(editor.presetID, normalizedDraft);
    }
    setEditor(null);
  };

  const deletePreset = async (preset: PresetPayload) => {
    if (!confirmPresetDeletion(copy.settings.presetsDeleteConfirm(preset.name))) {
      return;
    }
    await onDeletePreset(preset.id);
    if (editor?.presetID === preset.id) {
      setEditor(null);
    }
  };

  const activatePreset = async (presetID: string) => {
    setActivatingPresetID(presetID);
    try {
      await onActivatePreset(presetID);
    } finally {
      setActivatingPresetID(null);
    }
  };

  return (
    <section>
      <header className="mb-8 flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="mb-2 text-[28px] font-semibold tracking-tight text-[#111111]">{copy.settings.presetsTitle}</h1>
          <p className="text-[14px] text-[#737373]">{copy.settings.presetsDescription}</p>
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button"
            disabled={controlsDisabled}
            onClick={() => {
              ignorePromise(onRefresh());
            }}
            className="rounded-full border border-[#E5E5E5] px-4 py-2 text-[13px] font-medium text-[#111111] transition-colors hover:bg-[#F5F5F5] disabled:cursor-not-allowed disabled:opacity-50"
          >
            {copy.settings.refresh}
          </button>
          <button
            type="button"
            data-testid="preset-new-card"
            disabled={editorDisabled}
            onClick={() => {
              const draft = newPresetDraft();
              setEditor({ mode: 'create', original: draft, draft });
            }}
            className="rounded-full bg-[#111111] px-4 py-2 text-[13px] font-medium text-white transition-colors hover:bg-[#333333] disabled:cursor-not-allowed disabled:opacity-50"
          >
            {copy.settings.presetsNewPreset}
          </button>
        </div>
      </header>

      {loading && presets.length === 0 ? <LoadingPresetsNotice text={copy.settings.presetsLoading} /> : null}
      {toolsLoading || promptsLoading ? <LoadingPresetsNotice text={copy.settings.presetsDependenciesLoading} /> : null}

      {presets.length === 0 ? <PresetEmptyNotice /> : null}
      <div className="settings-card-list">
        {presets.map((preset, index) => (
          <PresetCard
            key={preset.id}
            preset={preset}
            index={index}
            promptNameByID={promptNameByID}
            active={activePresetID === preset.id}
            activating={activatingPresetID === preset.id}
            controlsDisabled={editorDisabled}
            onActivate={() => {
              ignorePromise(activatePreset(preset.id));
            }}
            onEdit={() => {
              const draft = presetToDraft(preset);
              setEditor({ mode: 'edit', presetID: preset.id, original: draft, draft });
            }}
            onDelete={() => {
              ignorePromise(deletePreset(preset));
            }}
          />
        ))}
      </div>

      {editor === null ? null : (
        <PresetEditor
          editor={editor}
          tools={tools}
          promptLibrary={promptLibrary}
          controlsDisabled={editorDisabled}
          onClose={() => setEditor(null)}
          onSave={() => {
            ignorePromise(saveEditor());
          }}
          onChange={(patch) => {
            setEditor((current) => {
              if (current === null) {
                return current;
              }
              return {
                ...current,
                draft: {
                  ...current.draft,
                  ...patch,
                },
              };
            });
          }}
        />
      )}
    </section>
  );
}
