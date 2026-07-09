import type { ChatRuntimeAction } from './actions';

export const CHAT_STREAM_ACTION_FLUSH_INTERVAL_MS = 50;

type TimerHandle = ReturnType<typeof setTimeout>;

export interface ChatRuntimeActionBuffer {
  cancel: () => void;
  enqueue: (sessionId: string, actions: ChatRuntimeAction[]) => void;
  flush: () => void;
}

interface ChatRuntimeActionBufferOptions {
  clearTimer?: (handle: TimerHandle) => void;
  scheduleFlush?: (flush: () => void, delayMs: number) => TimerHandle;
}

export function createChatRuntimeActionBuffer(
  applyRuntimeActions: (sessionId: string, actions: ChatRuntimeAction[]) => void,
  options: ChatRuntimeActionBufferOptions = {},
): ChatRuntimeActionBuffer {
  const scheduleFlush = options.scheduleFlush ?? setTimeout;
  const clearTimer = options.clearTimer ?? clearTimeout;
  let pendingSessionId = '';
  let pendingActions: ChatRuntimeAction[] = [];
  let timer: TimerHandle | null = null;

  const clearPendingTimer = () => {
    if (timer === null) {
      return;
    }
    clearTimer(timer);
    timer = null;
  };

  const flush = () => {
    clearPendingTimer();
    if (pendingActions.length === 0) {
      return;
    }
    const actions = pendingActions;
    const sessionId = pendingSessionId;
    pendingActions = [];
    pendingSessionId = '';
    applyRuntimeActions(sessionId, actions);
  };

  const schedulePendingFlush = () => {
    if (timer !== null) {
      return;
    }
    timer = scheduleFlush(flush, CHAT_STREAM_ACTION_FLUSH_INTERVAL_MS);
  };

  const enqueueBufferedAction = (sessionId: string, action: ChatRuntimeAction) => {
    if (pendingActions.length > 0 && pendingSessionId !== sessionId) {
      flush();
    }

    const lastAction = pendingActions.at(-1);
    const mergedAction = lastAction ? mergeBufferedAction(lastAction, action) : null;
    if (mergedAction) {
      pendingActions = [...pendingActions.slice(0, -1), mergedAction];
      return;
    }

    if (pendingActions.length > 0) {
      flush();
    }

    pendingSessionId = sessionId;
    pendingActions = [action];
    schedulePendingFlush();
  };

  return {
    cancel: () => {
      clearPendingTimer();
      pendingActions = [];
      pendingSessionId = '';
    },
    enqueue: (sessionId, actions) => {
      if (actions.length === 0) {
        return;
      }

      const normalizedSessionId = sessionId.trim();
      const immediateActions: ChatRuntimeAction[] = [];
      const applyImmediateActions = () => {
        if (immediateActions.length === 0) {
          return;
        }
        applyRuntimeActions(normalizedSessionId, [...immediateActions]);
        immediateActions.length = 0;
      };

      for (const action of actions) {
        if (isBufferedTextAction(action)) {
          applyImmediateActions();
          enqueueBufferedAction(normalizedSessionId, action);
          continue;
        }

        flush();
        immediateActions.push(action);
      }

      applyImmediateActions();
    },
    flush,
  };
}

function isBufferedTextAction(action: ChatRuntimeAction): boolean {
  return action.type === 'append_streaming_assistant_text'
    || action.type === 'append_streaming_thinking_text';
}

function mergeBufferedAction(
  current: ChatRuntimeAction,
  next: ChatRuntimeAction,
): ChatRuntimeAction | null {
  if (current.type === 'append_streaming_assistant_text' && next.type === 'append_streaming_assistant_text') {
    return {
      type: 'append_streaming_assistant_text',
      text: `${current.text}${next.text}`,
    };
  }

  if (current.type === 'append_streaming_thinking_text' && next.type === 'append_streaming_thinking_text') {
    return {
      type: 'append_streaming_thinking_text',
      text: `${current.text}${next.text}`,
    };
  }

  return null;
}
