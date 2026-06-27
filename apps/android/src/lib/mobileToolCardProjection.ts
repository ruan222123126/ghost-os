import type { MobileToolCard, MobileToolCardStatus } from "../mobileTypes";
import { parseAgentToolCallPayload } from "./agentStream";
import type {
  AgentAwaitingHumanStreamPayload,
  AgentCompletionDeltaPayload,
  AgentStreamEvent,
  AgentToolCallPayload,
} from "./agentStream";
import type { MobileAgentStreamRuntime } from "./mobileAgentStreamRuntime";
import type { ToolTagStreamEvent } from "./mobileToolTags";

export interface ProjectedToolStatus {
  toolCallId?: string;
  toolName?: string;
}

export function projectToolStarted(
  runtime: MobileAgentStreamRuntime,
  event: AgentStreamEvent,
): ProjectedToolStatus {
  const tool = parseAgentToolCallPayload(event.payload);
  const cardId = resolveToolCardId(runtime, event, tool.tool_call_id);
  const input = tool.arguments_json?.trim() || runtime.previewToolArgs.get(cardId);
  if (input) {
    runtime.previewToolArgs.set(cardId, input);
  }
  upsertToolCard(runtime, {
    id: cardId,
    input,
    status: "running",
    toolCallId: tool.tool_call_id,
    toolName: tool.tool || resolvePreviewToolName(runtime, cardId),
    traceId: event.trace_id,
  });
  clearStructuredPreviewLookups(runtime);
  return { toolCallId: tool.tool_call_id, toolName: tool.tool };
}

export function projectToolFinished(
  runtime: MobileAgentStreamRuntime,
  event: AgentStreamEvent,
): ProjectedToolStatus {
  const tool = parseAgentToolCallPayload(event.payload);
  const cardId = resolveToolCardId(runtime, event, tool.tool_call_id);
  upsertToolCard(runtime, {
    id: cardId,
    error: tool.error,
    input: runtime.previewToolArgs.get(cardId),
    output: tool.output,
    status: resolveFinishedToolStatus(tool),
    toolCallId: tool.tool_call_id,
    toolName: tool.tool || resolvePreviewToolName(runtime, cardId),
    traceId: event.trace_id,
  });
  clearStructuredPreviewLookups(runtime);
  return { toolCallId: tool.tool_call_id, toolName: tool.tool };
}

export function projectAwaitingHumanApproval(
  runtime: MobileAgentStreamRuntime,
  event: AgentStreamEvent,
  payload: AgentAwaitingHumanStreamPayload,
): void {
  const approvalId = payload.approval?.id?.trim() || payload.question_id.trim();
  const cardId = resolveToolCardId(runtime, event, payload.tool_call_id || approvalId);
  upsertToolCard(runtime, {
    approvalId,
    approvalKind: payload.approval?.kind,
    approvalPayload: payload.approval?.payload,
    approvalPrompt: payload.prompt,
    id: cardId,
    input: payload.prompt,
    status: "pending",
    toolCallId: payload.tool_call_id,
    toolName: payload.tool || payload.approval?.tool || "codex_approval",
    traceId: event.trace_id,
  });
}

export function projectToolCallStartDelta(
  runtime: MobileAgentStreamRuntime,
  traceId: string,
  payload: AgentCompletionDeltaPayload,
): void {
  const cardId = resolvePreviewToolCardId(runtime, payload.tool_call_index, payload.tool_call_id, traceId);
  if (payload.tool_name?.trim()) {
    runtime.previewToolNamesByCardId.set(cardId, payload.tool_name.trim());
  }
  upsertToolCard(runtime, {
    id: cardId,
    input: runtime.previewToolArgs.get(cardId),
    status: "pending",
    toolCallId: resolvePreviewToolCallId(runtime, cardId),
    toolName: resolvePreviewToolName(runtime, cardId),
    traceId,
  });
}

export function projectToolCallDelta(
  runtime: MobileAgentStreamRuntime,
  traceId: string,
  payload: AgentCompletionDeltaPayload,
): void {
  const cardId = resolvePreviewToolCardId(runtime, payload.tool_call_index, payload.tool_call_id, traceId);
  if (payload.arguments_fragment) {
    const nextInput = `${runtime.previewToolArgs.get(cardId) || ""}${payload.arguments_fragment}`;
    runtime.previewToolArgs.set(cardId, nextInput);
  }
  upsertToolCard(runtime, {
    id: cardId,
    input: runtime.previewToolArgs.get(cardId),
    status: "pending",
    toolCallId: resolvePreviewToolCallId(runtime, cardId),
    toolName: resolvePreviewToolName(runtime, cardId),
    traceId,
  });
}

export function projectToolCallEndDelta(
  runtime: MobileAgentStreamRuntime,
  traceId: string,
  payload: AgentCompletionDeltaPayload,
): void {
  const cardId = resolvePreviewToolCardId(runtime, payload.tool_call_index, payload.tool_call_id, traceId);
  if (payload.tool_call_index !== undefined && !resolveStructuredToolCallId(runtime, payload.tool_call_index)) {
    enqueuePendingPreview(runtime, cardId);
  }
  upsertToolCard(runtime, {
    id: cardId,
    input: runtime.previewToolArgs.get(cardId),
    status: "pending",
    toolCallId: resolvePreviewToolCallId(runtime, cardId),
    toolName: resolvePreviewToolName(runtime, cardId),
    traceId,
  });
}

export function projectToolTagEvents(
  runtime: MobileAgentStreamRuntime,
  traceId: string,
  events: ToolTagStreamEvent[],
): void {
  for (const event of events) {
    if (event.type === "tool_open") {
      projectToolTagOpen(runtime, traceId, event);
      continue;
    }
    if (event.type === "tool_args") {
      projectToolTagArgs(runtime, traceId, event);
      continue;
    }
    projectToolTagClose(runtime, traceId, event);
  }
}

export function cloneToolCard(tool: MobileToolCard): MobileToolCard {
  return { ...tool };
}

function projectToolTagOpen(
  runtime: MobileAgentStreamRuntime,
  traceId: string,
  event: Extract<ToolTagStreamEvent, { type: "tool_open" }>,
): void {
  const cardId = ensureToolTagCardId(runtime, event.callSeq, event.toolId, traceId);
  upsertToolCard(runtime, {
    id: cardId,
    input: runtime.previewToolArgs.get(cardId),
    status: "pending",
    toolName: formatToolIdName(event.toolId),
    traceId,
  });
}

function projectToolTagArgs(
  runtime: MobileAgentStreamRuntime,
  traceId: string,
  event: Extract<ToolTagStreamEvent, { type: "tool_args" }>,
): void {
  const cardId = runtime.previewToolCallSeqToId.get(event.callSeq);
  if (!cardId) {
    return;
  }
  const input = `${runtime.previewToolArgs.get(cardId) || ""}${event.argsDelta}`;
  runtime.previewToolArgs.set(cardId, input);
  upsertToolCard(runtime, {
    id: cardId,
    input,
    status: "pending",
    toolName: resolvePreviewToolName(runtime, cardId),
    traceId,
  });
}

function projectToolTagClose(
  runtime: MobileAgentStreamRuntime,
  traceId: string,
  event: Extract<ToolTagStreamEvent, { type: "tool_close" }>,
): void {
  const cardId = ensureToolTagCardId(runtime, event.callSeq, event.toolId, traceId);
  runtime.previewToolArgs.set(cardId, event.argsText);
  enqueuePendingPreview(runtime, cardId);
  upsertToolCard(runtime, {
    id: cardId,
    input: event.argsText,
    status: "pending",
    toolName: formatToolIdName(event.toolId),
    traceId,
  });
}

function resolvePreviewToolCardId(
  runtime: MobileAgentStreamRuntime,
  toolCallIndex: number | undefined,
  toolCallId: string | undefined,
  traceId: string,
): string {
  if (toolCallIndex === undefined) {
    return ensureToolCardIdWithCallId(runtime, toolCallId, traceId);
  }

  const existing = runtime.previewStructuredToolCardIdsByIndex.get(toolCallIndex);
  if (existing) {
    rememberStructuredToolCallId(runtime, toolCallIndex, toolCallId, existing);
    return existing;
  }

  const generated = ensureToolCardIdWithCallId(runtime, toolCallId, traceId)
    || createStructuredPreviewCardId(runtime, toolCallIndex, traceId);
  runtime.previewStructuredToolCardIdsByIndex.set(toolCallIndex, generated);
  rememberStructuredToolCallId(runtime, toolCallIndex, toolCallId, generated);
  return generated;
}

function resolveToolCardId(
  runtime: MobileAgentStreamRuntime,
  event: AgentStreamEvent,
  toolCallId: string | undefined,
): string {
  const trimmedToolCallId = toolCallId?.trim();
  if (trimmedToolCallId) {
    return resolveToolCardIdWithCallId(runtime, event.trace_id, trimmedToolCallId);
  }

  const eventKey = event.step_id.trim() || event.id.trim();
  const existing = runtime.toolCardIds.get(eventKey);
  if (existing) {
    return existing;
  }
  const generated = `stream-tool:${event.trace_id}:${eventKey}`;
  runtime.toolCardIds.set(eventKey, generated);
  return generated;
}

function resolveToolCardIdWithCallId(
  runtime: MobileAgentStreamRuntime,
  traceId: string,
  toolCallId: string,
): string {
  const existing = runtime.toolCardIds.get(toolCallId);
  if (existing) {
    return existing;
  }
  const pending = runtime.pendingPreviewQueue.shift();
  if (pending) {
    runtime.toolCardIds.set(toolCallId, pending);
    return pending;
  }
  const generated = `stream-tool:${traceId}:${toolCallId}`;
  runtime.toolCardIds.set(toolCallId, generated);
  return generated;
}

function ensureToolCardIdWithCallId(
  runtime: MobileAgentStreamRuntime,
  toolCallId: string | undefined,
  traceId: string,
): string {
  const trimmedToolCallId = toolCallId?.trim();
  if (!trimmedToolCallId) {
    return "";
  }

  const existing = runtime.toolCardIds.get(trimmedToolCallId);
  if (existing) {
    return existing;
  }
  const generated = `stream-tool:${traceId}:${trimmedToolCallId}`;
  runtime.toolCardIds.set(trimmedToolCallId, generated);
  return generated;
}

function rememberStructuredToolCallId(
  runtime: MobileAgentStreamRuntime,
  toolCallIndex: number,
  toolCallId: string | undefined,
  cardId: string,
): void {
  const trimmedToolCallId = toolCallId?.trim();
  if (!trimmedToolCallId) {
    return;
  }

  runtime.toolCardIds.set(trimmedToolCallId, cardId);
  runtime.previewStructuredToolCallIdsByIndex.set(toolCallIndex, trimmedToolCallId);
}

function resolveStructuredToolCallId(
  runtime: MobileAgentStreamRuntime,
  toolCallIndex: number,
): string | undefined {
  return runtime.previewStructuredToolCallIdsByIndex.get(toolCallIndex)?.trim() || undefined;
}

function resolvePreviewToolCallId(runtime: MobileAgentStreamRuntime, cardId: string): string | undefined {
  for (const [toolCallId, currentCardId] of runtime.toolCardIds.entries()) {
    if (currentCardId === cardId) {
      return toolCallId;
    }
  }
  return undefined;
}

function resolvePreviewToolName(runtime: MobileAgentStreamRuntime, cardId: string): string | undefined {
  return runtime.previewToolNamesByCardId.get(cardId) || formatToolIdName(runtime.previewToolIdsByCardId.get(cardId));
}

function createStructuredPreviewCardId(
  runtime: MobileAgentStreamRuntime,
  toolCallIndex: number,
  traceId: string,
): string {
  const previewSeq = runtime.nextStructuredToolPreviewSeq;
  runtime.nextStructuredToolPreviewSeq += 1;
  return `stream-tool:${traceId}:preview:${previewSeq}:index:${toolCallIndex}`;
}

function ensureToolTagCardId(
  runtime: MobileAgentStreamRuntime,
  callSeq: number,
  toolId: string,
  traceId: string,
): string {
  const existing = runtime.previewToolCallSeqToId.get(callSeq);
  if (existing) {
    return existing;
  }

  const cardId = `stream-tag-tool:${traceId}:${callSeq}`;
  runtime.previewToolCallSeqToId.set(callSeq, cardId);
  runtime.previewToolIdsByCardId.set(cardId, toolId);
  if (!runtime.previewToolArgs.has(cardId)) {
    runtime.previewToolArgs.set(cardId, "");
  }
  return cardId;
}

function enqueuePendingPreview(runtime: MobileAgentStreamRuntime, cardId: string): void {
  if (!runtime.pendingPreviewQueue.includes(cardId)) {
    runtime.pendingPreviewQueue.push(cardId);
  }
}

function clearStructuredPreviewLookups(runtime: MobileAgentStreamRuntime): void {
  runtime.previewStructuredToolCallIdsByIndex.clear();
  runtime.previewStructuredToolCardIdsByIndex.clear();
}

function upsertToolCard(runtime: MobileAgentStreamRuntime, patch: MobileToolCard): void {
  const existingIndex = runtime.tools.findIndex((tool) => tool.id === patch.id);
  if (existingIndex < 0) {
    runtime.tools.push(patch);
    return;
  }

  runtime.tools[existingIndex] = {
    ...runtime.tools[existingIndex],
    ...patch,
  };
}

function resolveFinishedToolStatus(tool: AgentToolCallPayload): MobileToolCardStatus {
  if (tool.status === "error" || tool.error) {
    return "error";
  }
  return "success";
}

function formatToolIdName(toolId?: string): string | undefined {
  const trimmed = toolId?.trim();
  return trimmed ? `tool#${trimmed}` : undefined;
}
