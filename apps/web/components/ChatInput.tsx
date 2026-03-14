// ChatInput component used by the web console chat/session interface.

'use client';

import type { FC } from 'react';
import { useCallback, useState } from 'react';
import { ChatComposer } from '@/components/ChatComposer';
import { ModelSelector } from '@/components/ModelSelector';
import type { ProviderModelOption } from '@/lib/types';

interface ChatInputProps {
  loading: boolean;
  canStop?: boolean;
  disabled: boolean;
  awaitingQuestion?: boolean;
  modelLoading?: boolean;
  activeModel?: ProviderModelOption | null;
  availableModels?: ProviderModelOption[];
  onSend: (message: string) => Promise<void>;
  onStop?: () => Promise<void>;
  onSelectModel?: (option: ProviderModelOption) => Promise<boolean>;
}

export const ChatInput: FC<ChatInputProps> = ({
  loading,
  canStop = false,
  disabled,
  awaitingQuestion = false,
  modelLoading = false,
  activeModel = null,
  availableModels = [],
  onSend,
  onStop,
  onSelectModel,
}) => {
  const [draft, setDraft] = useState('');
  const handleSubmit = useCallback(async (message: string) => {
    setDraft('');
    await onSend(message);
  }, [onSend]);

  return (
    <ChatComposer
      value={draft}
      onChange={setDraft}
      onSubmit={handleSubmit}
      onStop={onStop}
      sending={loading}
      canStop={canStop}
      disabled={disabled}
      ariaLabel="Message input"
      placeholder={loading ? '正在思考中...' : '输入消息...'}
      rows={4}
      toolbar={onSelectModel ? (
        <ModelSelector
          value={activeModel}
          options={availableModels}
          loading={modelLoading}
          disabled={disabled || loading}
          onChange={onSelectModel}
        />
      ) : undefined}
      status={awaitingQuestion ? '等待问题回答中' : undefined}
    />
  );
};
