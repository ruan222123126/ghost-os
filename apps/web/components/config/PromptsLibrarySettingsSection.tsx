'use client';

import { useState } from 'react';
import { ignorePromise } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { PromptLibraryItem, SystemPromptPayload } from '@/lib/types';
import {
  applyPromptLibraryActivationRules,
  confirmPromptDeletion,
  newPromptLibraryCard,
  normalizePromptLibraryCard,
  type PromptLibraryEditorState,
} from './promptLibraryHelpers';
import {
  LoadingPromptsNotice,
  PromptLibraryCard,
  PromptLibraryEditor,
  PromptLibraryEmptyNotice,
} from './promptLibraryParts';

interface PromptsLibrarySettingsSectionProps {
  prompts: SystemPromptPayload | null;
  loading: boolean;
  saving: boolean;
  onRefresh: () => Promise<void>;
  onSavePromptLibrary: (promptLibrary: PromptLibraryItem[]) => Promise<void>;
}

export function PromptsLibrarySettingsSection(props: PromptsLibrarySettingsSectionProps) {
  const { copy } = useWebLocale();
  const { prompts, loading, saving, onRefresh, onSavePromptLibrary } = props;
  const controlsDisabled = loading || saving;
  const promptLibrary = prompts?.prompt_library ?? [];
  const [editor, setEditor] = useState<PromptLibraryEditorState | null>(null);

  const savePromptLibrary = async (nextPromptLibrary: PromptLibraryItem[]) => {
    await onSavePromptLibrary(nextPromptLibrary);
  };

  const updateCard = (cardID: string, updater: (item: PromptLibraryItem) => PromptLibraryItem) => {
    return promptLibrary.map((item) => {
      if (item.id === cardID) {
        return updater(item);
      }
      return item;
    });
  };

  const saveEditor = async () => {
    if (editor === null) {
      return;
    }
    const normalizedDraft = normalizePromptLibraryCard(editor.draft);
    if (normalizedDraft.name === '') {
      return;
    }

    const nextPromptLibrary = editor.mode === 'create'
      ? [...promptLibrary, normalizedDraft]
      : updateCard(normalizedDraft.id, () => normalizedDraft);
    const syncedPromptLibrary = applyPromptLibraryActivationRules(
      promptLibrary,
      nextPromptLibrary,
      normalizedDraft.id,
    );

    await savePromptLibrary(syncedPromptLibrary);
    setEditor(null);
  };

  const deleteCard = async (card: PromptLibraryItem) => {
    if (!confirmPromptDeletion(copy.settings.promptsLibraryDeleteConfirm(card.name))) {
      return;
    }
    const nextPromptLibrary = promptLibrary.filter((item) => item.id !== card.id);
    await savePromptLibrary(nextPromptLibrary);
    if (editor?.draft.id === card.id) {
      setEditor(null);
    }
  };

  const toggleCard = async (card: PromptLibraryItem) => {
    const toggled = updateCard(card.id, (item) => ({ ...item, active: !item.active }));
    const synced = applyPromptLibraryActivationRules(promptLibrary, toggled, card.id);
    await savePromptLibrary(synced);
  };

  const listDisabled = controlsDisabled || prompts === null;

  return (
    <section>
      <header className="mb-8 flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="mb-2 text-[28px] font-semibold tracking-tight text-[#111111]">{copy.settings.tabPromptsLibrary}</h1>
          <p className="text-[14px] text-[#737373]">{copy.settings.promptsLibraryDescription}</p>
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
            data-testid="prompt-library-new-card"
            disabled={listDisabled}
            onClick={() => {
              const card = newPromptLibraryCard(promptLibrary);
              setEditor({ mode: 'create', original: card, draft: card });
            }}
            className="rounded-full bg-[#111111] px-4 py-2 text-[13px] font-medium text-white transition-colors hover:bg-[#333333] disabled:cursor-not-allowed disabled:opacity-50"
          >
            {copy.settings.promptsLibraryNewCard}
          </button>
        </div>
      </header>

      {loading && prompts === null ? <LoadingPromptsNotice text={copy.settings.promptsLoading} /> : null}

      {promptLibrary.length === 0 ? <PromptLibraryEmptyNotice /> : null}
      <div className="space-y-3">
        {promptLibrary.map((item, index) => (
          <PromptLibraryCard
            key={item.id}
            item={item}
            index={index}
            controlsDisabled={listDisabled}
            onToggle={() => {
              ignorePromise(toggleCard(item));
            }}
            onEdit={() => {
              setEditor({ mode: 'edit', original: item, draft: item });
            }}
            onDelete={() => {
              ignorePromise(deleteCard(item));
            }}
          />
        ))}
      </div>

      {editor === null ? null : (
        <PromptLibraryEditor
          editor={editor}
          controlsDisabled={controlsDisabled}
          onClose={() => setEditor(null)}
          onDelete={() => {
            if (editor.mode === 'create') {
              setEditor(null);
              return;
            }
            ignorePromise(deleteCard(editor.draft));
          }}
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
