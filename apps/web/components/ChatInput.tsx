// ChatInput component used by the web console chat/session interface.

'use client';

import type { FC, ReactNode } from 'react';
import { useCallback, useState } from 'react';
import { ChatComposer } from '@/components/ChatComposer';
import { ComposerImageStrip } from '@/components/ComposerImageStrip';
import { ModelSelector } from '@/components/ModelSelector';
import { useComposerSkills } from '@/hooks/useComposerSkills';
import { createChatImageDrafts } from '@/lib/chatImageDrafts';
import { toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { AgentRuntimeType, ChatImageDraft, ChatSelectedSkill, ChatSendInput, ProviderModelOption } from '@/lib/types';

interface ChatInputProps {
  loading: boolean;
  canStop?: boolean;
  canEnableCodexMode?: boolean;
  disabled: boolean;
  awaitingQuestion?: boolean;
  modelLoading?: boolean;
  activeModel?: ProviderModelOption | null;
  availableModels?: ProviderModelOption[];
  onSend: (input: ChatSendInput) => Promise<void>;
  onStop?: () => Promise<void>;
  onSelectModel?: (option: ProviderModelOption) => Promise<boolean>;
}

export const ChatInput: FC<ChatInputProps> = ({
  loading,
  canStop = false,
  canEnableCodexMode = false,
  disabled,
  awaitingQuestion = false,
  modelLoading = false,
  activeModel = null,
  availableModels = [],
  onSend,
  onStop,
  onSelectModel,
}) => {
  const { copy } = useWebLocale();
  const [draft, setDraft] = useState('');
  const [agentRuntime, setAgentRuntime] = useState<AgentRuntimeType>('ghost');
  const [pendingImages, setPendingImages] = useState<ChatImageDraft[]>([]);
  const [imageError, setImageError] = useState('');
  const [selectedSkill, setSelectedSkill] = useState<ChatSelectedSkill | null>(null);
  const { skillError, skills, skillsLoading, refreshSkills } = useComposerSkills();
  const canSubmit = draft.trim().length > 0 || pendingImages.length > 0 || selectedSkill !== null;
  const codexModeEnabled = agentRuntime === 'codex';
  const codexModeBlockedByImages = pendingImages.length > 0;
  const codexModeDisabledMessage = codexModeBlockedByImages
    ? copy.chat.composerCodexImagesUnsupported
    : copy.chat.composerCodexUnavailable;

  const handleSubmit = useCallback(async () => {
    const input = buildChatSendInput(draft, pendingImages, selectedSkill, agentRuntime);
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
  }, [agentRuntime, draft, onSend, pendingImages, selectedSkill]);

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
      agentRuntime={agentRuntime}
      canEnableCodexMode={canEnableCodexMode && !codexModeBlockedByImages}
      codexModeDisabledMessage={codexModeDisabledMessage}
      value={draft}
      onChange={setDraft}
      onSubmit={handleSubmit}
      onStop={onStop}
      onSwitchAgentRuntime={setAgentRuntime}
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
      toolbar={onSelectModel && !codexModeEnabled ? (
        <ModelSelector
          value={activeModel}
          options={availableModels}
          loading={modelLoading}
          disabled={disabled || loading}
          onChange={onSelectModel}
        />
      ) : undefined}
      status={awaitingQuestion ? copy.chat.composerAwaitingQuestion : undefined}
    />
  );
};

function buildChatSendInput(
  message: string,
  images: ChatImageDraft[],
  selectedSkill: ChatSelectedSkill | null,
  agentRuntime: AgentRuntimeType,
): ChatSendInput {
  const input: ChatSendInput = {
    agentRuntime,
    message: message.trim(),
    images,
  };
  return selectedSkill ? { ...input, selectedSkill } : input;
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
