import { useWebLocale } from '@/lib/i18n/provider';
import { useChatRunActions, type UseChatRunActionsOptions } from './useChatRunActions';

type UseChatRunControlOptions = Omit<
  UseChatRunActionsOptions,
  'codexImagesUnsupportedText' | 'codexUnavailableText' | 'requestFailedText'
>;

export function useChatRunControl(options: UseChatRunControlOptions) {
  const { copy } = useWebLocale();

  return useChatRunActions({
    ...options,
    codexImagesUnsupportedText: copy.chat.composerCodexImagesUnsupported,
    codexUnavailableText: copy.chat.composerCodexUnavailable,
    requestFailedText: copy.system.genericRequestFailed,
  });
}
