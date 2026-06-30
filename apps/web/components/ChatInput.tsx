// ChatInput component used by the web console chat/session interface.

'use client';

import type { FC, ReactNode } from 'react';
import { useCallback, useState } from 'react';
import { ChatComposer } from '@/components/ChatComposer';
import { ComposerImageStrip } from '@/components/ComposerImageStrip';
import { ModelSelector } from '@/components/ModelSelector';
import { useComposerSkills } from '@/hooks/useComposerSkills';
import { createChatImageDrafts } from '@/lib/chatImageDrafts';
import { DEFAULT_CODEX_MODEL } from '@/lib/codexModels';
import { toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { AgentModeSelection, ChatImageDraft, ChatSelectedSkill, ChatSendInput, ProviderModelOption } from '@/lib/types';

interface ChatInputProps {
  loading: boolean;
  canStop?: boolean;
  canEnableCodexMode?: boolean;
  disabled: boolean;
  awaitingQuestion?: boolean;
  modelLoading?: boolean;
  agentMode?: AgentModeSelection;
  activeModel?: ProviderModelOption | null;
  availableModels?: ProviderModelOption[];
  onSend: (input: ChatSendInput) => Promise<void>;
  onStop?: () => Promise<void>;
  onChangeAgentMode?: (mode: AgentModeSelection) => void;
  onSelectModel?: (option: ProviderModelOption) => Promise<boolean>;
}

export const ChatInput: FC<ChatInputProps> = ({
  loading,
  canStop = false,
  canEnableCodexMode = false,
  disabled,
  awaitingQuestion = false,
  modelLoading = false,
  agentMode: controlledAgentMode,
  activeModel = null,
  availableModels = [],
  onSend,
  onStop,
  onChangeAgentMode,
  onSelectModel,
}) => {
  const { copy } = useWebLocale();
  const [draft, setDraft] = useState('');
  const [uncontrolledAgentMode, setUncontrolledAgentMode] = useState<AgentModeSelection>(null);
  const agentMode = controlledAgentMode ?? uncontrolledAgentMode;
  const setAgentMode = onChangeAgentMode ?? setUncontrolledAgentMode;
  const [pendingImages, setPendingImages] = useState<ChatImageDraft[]>([]);
  const [imageError, setImageError] = useState('');
  const [selectedSkill, setSelectedSkill] = useState<ChatSelectedSkill | null>(null);
  const { skillError, skills, skillsLoading, refreshSkills } = useComposerSkills();
  const canSubmit = draft.trim().length > 0 || pendingImages.length > 0 || selectedSkill !== null;

  const handleSubmit = useCallback(async () => {
    const input = buildChatSendInput(draft, pendingImages, selectedSkill, agentMode, activeModel);
    if (!canSubmitChatInput(input)) {
      return;
    }

    const previousDraft = draft;
    const previousImages = pendingImages;
    const previousSkill = selectedSkill;
    setDraft('');
    setPendingImages([]);
    setSelectedSkill(null);
    setImageError('');

    try {
      await onSend(input);
    } catch (error) {
      setDraft(previousDraft);
      setPendingImages(previousImages);
      setSelectedSkill(previousSkill);
      throw error;
    }
  }, [activeModel, agentMode, draft, onSend, pendingImages, selectedSkill]);

  const handleSelectFiles = useCallback(async (files: FileList) => {
    try {
      const nextImages = await createChatImageDrafts(Array.from(files));
      setPendingImages((current) => [...current, ...nextImages]);
      setImageError('');
    } catch (error) {
      setImageError(toErrorMessage(error, copy.system.genericRequestFailed));
    }
  }, [copy.system.genericRequestFailed]);

  const handleRemoveImage = useCallback((imageId: string) => {
    setPendingImages((current) => current.filter((image) => image.id !== imageId));
    setImageError('');
  }, []);

  return (
    <ChatComposer
      agentMode={agentMode}
      canEnableCodexMode={canEnableCodexMode}
      codexModeDisabledMessage={copy.chat.composerCodexUnavailable}
      value={draft}
      onChange={setDraft}
      onSubmit={handleSubmit}
      onStop={onStop}
      onChangeAgentMode={setAgentMode}
      onSelectFiles={handleSelectFiles}
      sending={loading}
      canStop={canStop}
      canSubmit={canSubmit}
      disabled={disabled}
      ariaLabel={copy.chat.composerMessageInputAria}
      placeholder={loading ? copy.chat.composerThinkingPlaceholder : copy.chat.composerInputPlaceholder}
      rows={1}
      preview={<ComposerImageStrip images={pendingImages} onRemove={handleRemoveImage} />}
      selectedSkill={selectedSkill}
      skillError={skillError}
      skills={skills}
      skillsLoading={skillsLoading}
      onClearSelectedSkill={() => setSelectedSkill(null)}
      onRefreshSkills={refreshSkills}
      onSelectSkill={(skill) => setSelectedSkill({ id: skill.id, name: skill.name })}
      hint={buildHint(copy, imageError, pendingImages)}
      toolbar={buildModelSelector({
        activeModel,
        availableModels,
        disabled: disabled || loading,
        modelLoading,
        onSelectModel,
      })}
      status={awaitingQuestion ? copy.chat.composerAwaitingQuestion : undefined}
    />
  );
};

function buildModelSelector(options: {
  activeModel: ProviderModelOption | null;
  availableModels: ProviderModelOption[];
  disabled: boolean;
  modelLoading: boolean;
  onSelectModel?: (option: ProviderModelOption) => Promise<boolean>;
}): ReactNode {
  if (!options.onSelectModel) {
    return undefined;
  }
  return (
    <ModelSelector
      value={options.activeModel}
      options={options.availableModels}
      loading={options.modelLoading}
      disabled={options.disabled}
      onChange={options.onSelectModel}
    />
  );
}

function buildChatSendInput(
  message: string,
  images: ChatImageDraft[],
  selectedSkill: ChatSelectedSkill | null,
  agentMode: AgentModeSelection,
  activeModel: ProviderModelOption | null,
): ChatSendInput {
  const codexModeEnabled = agentMode !== null;
  const input: ChatSendInput = {
    agentRuntime: codexModeEnabled ? 'codex' : undefined,
    codexMode: resolveCodexMode(agentMode),
    message: message.trim(),
    model: resolveCodexModel(agentMode, activeModel),
    images,
  };
  return selectedSkill ? { ...input, selectedSkill } : input;
}

function resolveCodexMode(agentMode: AgentModeSelection): ChatSendInput['codexMode'] {
  if (agentMode === 'normal') {
    return 'default';
  }
  if (agentMode === 'plan') {
    return 'plan';
  }
  return undefined;
}

function resolveCodexModel(
  agentMode: AgentModeSelection,
  activeModel: ProviderModelOption | null,
): string | undefined {
  if (agentMode === null) {
    return undefined;
  }

  if (activeModel?.providerType === 'codex') {
    return activeModel.model.trim() || DEFAULT_CODEX_MODEL;
  }

  return DEFAULT_CODEX_MODEL;
}

function canSubmitChatInput(input: ChatSendInput): boolean {
  return input.message.length > 0 || input.images.length > 0 || input.selectedSkill !== undefined;
}

function buildHint(
  copy: ReturnType<typeof useWebLocale>['copy'],
  imageError: string,
  images: ChatImageDraft[],
): ReactNode {
  if (imageError) {
    return <span className="composer-inline-error">{imageError}</span>;
  }
  if (images.length === 0) {
    return undefined;
  }

  return copy.chat.composerPendingImages(images.length);
}
