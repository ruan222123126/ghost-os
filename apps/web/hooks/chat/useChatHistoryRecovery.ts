import { useWebLocale } from '@/lib/i18n/provider';
import {
  useChatHistoryRecoveryActions,
  type UseChatHistoryRecoveryActionsOptions,
} from './useChatHistoryRecoveryActions';

type UseChatHistoryRecoveryOptions = Omit<UseChatHistoryRecoveryActionsOptions, 'requestFailedText'>;

export function useChatHistoryRecovery(options: UseChatHistoryRecoveryOptions) {
  const { copy } = useWebLocale();

  return useChatHistoryRecoveryActions({
    ...options,
    requestFailedText: copy.system.genericRequestFailed,
  });
}
