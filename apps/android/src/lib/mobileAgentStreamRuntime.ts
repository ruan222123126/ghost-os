import type { AgentPayload, MobileToolCard, StatusMessage } from "../mobileTypes";
import {
  assertAgentStreamTerminal,
  createAgentStreamSummary,
  parseAgentAwaitingHumanStreamPayload,
  parseAgentCompletionDeltaPayload,
  parseAgentDonePayload,
  parseAgentErrorPayload,
  parseAgentStreamMessagePayload,
  resolveSessionId,
  updateAgentStreamSummary,
} from "./agentStream";
import type { AgentStreamEvent, AgentStreamResult } from "./agentStream";
import { MobileWebRTCBridge } from "./mobileWebRTC";
import {
  cloneToolCard,
  projectAwaitingHumanApproval,
  projectToolCallDelta,
  projectToolCallEndDelta,
  projectToolCallStartDelta,
  projectToolFinished,
  projectToolStarted,
  projectToolTagEvents,
} from "./mobileToolCardProjection";
import {
  consumeToolTagStreamChunk,
  createToolTagStreamState,
  stripToolTagCalls,
} from "./mobileToolTags";
import type { ToolTagStreamState } from "./mobileToolTags";

export interface MobileAgentStreamRuntime {
  message: string;
  sessionEnded: boolean;
  sessionId: string;
  thinking: string;
  terminal: boolean;
  nextStructuredToolPreviewSeq: number;
  pendingPreviewQueue: string[];
  previewStructuredToolCallIdsByIndex: Map<number, string>;
  previewStructuredToolCardIdsByIndex: Map<number, string>;
  previewToolArgs: Map<string, string>;
  previewToolCallSeqToId: Map<number, string>;
  previewToolIdsByCardId: Map<string, string>;
  previewToolNamesByCardId: Map<string, string>;
  toolCardIds: Map<string, string>;
  tools: MobileToolCard[];
  toolTags: ToolTagStreamState;
}

export interface MobileAgentStreamProjector {
  commitReply: (runtime: MobileAgentStreamRuntime) => void;
  commitSessionId: (sessionId: string) => void;
  setStatus: (status: StatusMessage) => void;
}

export function createMobileAgentStreamRuntime(sessionId: string): MobileAgentStreamRuntime {
  return {
    message: "",
    nextStructuredToolPreviewSeq: 1,
    pendingPreviewQueue: [],
    previewStructuredToolCallIdsByIndex: new Map(),
    previewStructuredToolCardIdsByIndex: new Map(),
    previewToolArgs: new Map(),
    previewToolCallSeqToId: new Map(),
    previewToolIdsByCardId: new Map(),
    previewToolNamesByCardId: new Map(),
    sessionEnded: false,
    sessionId: sessionId.trim(),
    thinking: "",
    terminal: false,
    toolCardIds: new Map(),
    tools: [],
    toolTags: createToolTagStreamState(),
  };
}

export function createAgentPayloadFromRuntime(runtime: MobileAgentStreamRuntime): AgentPayload {
  return {
    message: runtime.message,
    session_ended: runtime.sessionEnded,
    session_id: runtime.sessionId,
    thinking: runtime.thinking,
    tools: runtime.tools.length > 0 ? runtime.tools.map(cloneToolCard) : undefined,
  };
}

export function projectMobileAgentStreamEvent(
  event: AgentStreamEvent,
  runtime: MobileAgentStreamRuntime,
  projector: MobileAgentStreamProjector,
): void {
  applyEventSessionId(event, runtime, projector);

  switch (event.type) {
    case "run_started":
      projector.setStatus({ tone: "loading", text: "运行中" });
      projector.commitReply(runtime);
      break;
    case "completion_delta":
      projectCompletionDelta(event, runtime, projector);
      break;
    case "tool_call_started":
      projectStartedTool(event, runtime, projector);
      break;
    case "tool_call_finished":
      projectFinishedTool(event, runtime, projector);
      break;
    case "awaiting_human":
      projectAwaitingHuman(event, runtime, projector);
      break;
    case "message":
      projectFinalMessage(event, runtime, projector);
      break;
    case "done":
      projectDone(event, runtime, projector);
      break;
    case "error":
      projectError(event, runtime, projector);
      break;
    default:
      break;
  }
}

export async function streamAgentMessageWebRTC(
  client: MobileWebRTCBridge,
  params: Record<string, unknown>,
  traceId: string,
  onEvent: (event: AgentStreamEvent) => void,
  action?: string,
): Promise<AgentStreamResult> {
  const summary = createAgentStreamSummary();
  const endPayload = await client.streamAgent<Record<string, unknown>>(params, traceId, (event) => {
    onEvent(event);
    updateAgentStreamSummary(summary, event);
  }, action);
  const streamEndSessionId = parseStreamEndSessionId(endPayload);
  if (streamEndSessionId && !summary.result.sessionId) {
    summary.result.sessionId = streamEndSessionId;
  }
  assertAgentStreamTerminal(summary);
  return summary.result;
}

function applyEventSessionId(
  event: AgentStreamEvent,
  runtime: MobileAgentStreamRuntime,
  projector: MobileAgentStreamProjector,
): void {
  const sessionId = resolveSessionId(event);
  if (!sessionId || sessionId === runtime.sessionId) {
    return;
  }
  runtime.sessionId = sessionId;
  projector.commitSessionId(sessionId);
}

function projectCompletionDelta(
  event: AgentStreamEvent,
  runtime: MobileAgentStreamRuntime,
  projector: MobileAgentStreamProjector,
): void {
  const payload = parseAgentCompletionDeltaPayload(event.payload);
  switch (payload.kind) {
    case "text":
      projectTextDelta(runtime, event.trace_id, payload.text ?? "");
      projector.setStatus({ tone: "loading", text: "生成中" });
      projector.commitReply(runtime);
      break;
    case "thinking":
      appendThinkingText(runtime, payload.thinking ?? "");
      projector.setStatus({ tone: "loading", text: "思考中" });
      projector.commitReply(runtime);
      break;
    case "tool_call_start":
      projectToolCallStartDelta(runtime, event.trace_id, payload);
      projector.setStatus({
        tone: "loading",
        text: toolStatusText("正在准备工具", payload.tool_name, payload.tool_call_id),
      });
      projector.commitReply(runtime);
      break;
    case "tool_call_delta":
      projectToolCallDelta(runtime, event.trace_id, payload);
      projector.setStatus({
        tone: "loading",
        text: toolStatusText("正在准备工具", payload.tool_name, payload.tool_call_id),
      });
      projector.commitReply(runtime);
      break;
    case "tool_call_end":
      projectToolCallEndDelta(runtime, event.trace_id, payload);
      projector.setStatus({
        tone: "loading",
        text: toolStatusText("工具调用已提交", payload.tool_name, payload.tool_call_id),
      });
      projector.commitReply(runtime);
      break;
    default:
      break;
  }
}

function projectStartedTool(
  event: AgentStreamEvent,
  runtime: MobileAgentStreamRuntime,
  projector: MobileAgentStreamProjector,
): void {
  const tool = projectToolStarted(runtime, event);
  projector.setStatus({ tone: "loading", text: toolStatusText("正在调用工具", tool.toolName, tool.toolCallId) });
  projector.commitReply(runtime);
}

function projectFinishedTool(
  event: AgentStreamEvent,
  runtime: MobileAgentStreamRuntime,
  projector: MobileAgentStreamProjector,
): void {
  const tool = projectToolFinished(runtime, event);
  projector.setStatus({ tone: "loading", text: toolStatusText("工具已返回", tool.toolName, tool.toolCallId) });
  projector.commitReply(runtime);
}

function projectAwaitingHuman(
  event: AgentStreamEvent,
  runtime: MobileAgentStreamRuntime,
  projector: MobileAgentStreamProjector,
): void {
  const payload = parseAgentAwaitingHumanStreamPayload(event.payload);
  if (payload.approval?.id || payload.tool === "codex_approval") {
    projectAwaitingHumanApproval(runtime, event, payload);
  } else {
    runtime.message = payload.prompt;
  }
  runtime.terminal = true;
  projector.setStatus({ tone: "success", text: "等待用户输入" });
  projector.commitReply(runtime);
}

function projectFinalMessage(
  event: AgentStreamEvent,
  runtime: MobileAgentStreamRuntime,
  projector: MobileAgentStreamProjector,
): void {
  const payload = parseAgentStreamMessagePayload(event.payload);
  const finalized = consumeToolTagStreamChunk(runtime.toolTags, "", true);
  appendVisibleText(runtime, finalized.visibleText);
  projectToolTagEvents(runtime, event.trace_id, finalized.events);
  const finalText = stripToolTagCalls(payload.text);
  runtime.message = finalText.trim() ? finalText : runtime.message;
  projector.commitReply(runtime);
}

function projectTextDelta(runtime: MobileAgentStreamRuntime, traceId: string, text: string): void {
  const consumed = consumeToolTagStreamChunk(runtime.toolTags, text);
  appendVisibleText(runtime, consumed.visibleText);
  projectToolTagEvents(runtime, traceId, consumed.events);
}

function projectDone(
  event: AgentStreamEvent,
  runtime: MobileAgentStreamRuntime,
  projector: MobileAgentStreamProjector,
): void {
  const payload = parseAgentDonePayload(event.payload);
  runtime.sessionEnded = Boolean(payload.session_ended);
  runtime.terminal = true;
  projector.setStatus({ tone: "success", text: runtime.sessionEnded ? "会话已结束" : "回复已返回" });
  projector.commitReply(runtime);
}

function projectError(
  event: AgentStreamEvent,
  runtime: MobileAgentStreamRuntime,
  projector: MobileAgentStreamProjector,
): void {
  const payload = parseAgentErrorPayload(event.payload);
  runtime.terminal = true;
  projector.setStatus({ tone: "error", text: payload.message });
  projector.commitReply(runtime);
}

function appendVisibleText(runtime: MobileAgentStreamRuntime, text: string): void {
  if (!text) {
    return;
  }
  runtime.message = `${runtime.message}${text}`;
}

function appendThinkingText(runtime: MobileAgentStreamRuntime, text: string): void {
  if (!text) {
    return;
  }
  runtime.thinking = `${runtime.thinking}${text}`;
}

function toolStatusText(prefix: string, toolName?: string, toolCallId?: string): string {
  const label = toolName?.trim() || toolCallId?.trim();
  return label ? `${prefix}：${label}` : prefix;
}

function parseStreamEndSessionId(payload: Record<string, unknown>): string | undefined {
  const sessionId = payload.session_id;
  return typeof sessionId === "string" && sessionId.trim() ? sessionId.trim() : undefined;
}
