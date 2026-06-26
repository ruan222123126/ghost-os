import { useWebLocale } from '@/lib/i18n/provider';
import { useChatRunActions, type UseChatRunActionsOptions } from './useChatRunActions';

type UseChatRunControlOptions = Omit<UseChatRunActionsOptions, 'requestFailedText'>;

export function useChatRunControl(options: UseChatRunControlOptions) {
  const { copy } = useWebLocale();

  return useChatRunActions({
    ...options,
    requestFailedText: copy.system.genericRequestFailed,
  });
}
