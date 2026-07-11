import { useWebLocale } from '@/lib/i18n/provider';
import { useChatStreamRunners } from './useChatStreamRunners';
import type { UseChatStreamControllerOptions } from './chatStreamControllerTypes';

export function useChatStreamController(options: UseChatStreamControllerOptions) {
  const { copy } = useWebLocale();

  return useChatStreamRunners({
    ...options,
    requestFailedText: copy.system.genericRequestFailed,
  });
}
