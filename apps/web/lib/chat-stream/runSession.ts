import type { ActiveAgentRun } from './types';
import { resolveEventSessionId } from './sessionEvent';
import type { AgentStreamEvent } from '@/lib/types';

export interface RuntimeSessionResolution {
  nextActiveRun?: ActiveAgentRun;
  notifySessionResolved: boolean;
  sessionId: string;
}

interface ResolveEventSessionOptions {
  activeRun: ActiveAgentRun | null;
  currentSessionId: string;
  event: AgentStreamEvent;
  runtimeSessionId: string;
}

interface ResolveStreamSessionOptions {
  activeRun: ActiveAgentRun | null;
  currentSessionId: string;
  sessionId?: string;
}

export function resolveEventSession(
  options: ResolveEventSessionOptions,
): RuntimeSessionResolution | null {
  const sessionId = resolveEventSessionId(options.event);
  if (!sessionId || sessionId === options.runtimeSessionId) {
    return null;
  }
  return buildSessionResolution({
    activeRun: options.activeRun,
    currentSessionId: options.currentSessionId,
    sessionId,
  });
}

export function resolveStreamSession(
  options: ResolveStreamSessionOptions,
): RuntimeSessionResolution | null {
  const sessionId = options.sessionId?.trim();
  if (!sessionId) {
    return null;
  }
  return buildSessionResolution({
    activeRun: options.activeRun,
    currentSessionId: options.currentSessionId,
    sessionId,
  });
}

function buildSessionResolution(options: {
  activeRun: ActiveAgentRun | null;
  currentSessionId: string;
  sessionId: string;
}): RuntimeSessionResolution {
  const nextActiveRun = options.activeRun && options.activeRun.sessionId !== options.sessionId
    ? { ...options.activeRun, sessionId: options.sessionId }
    : undefined;

  return {
    nextActiveRun,
    notifySessionResolved: options.sessionId !== options.currentSessionId,
    sessionId: options.sessionId,
  };
}
