import type { SidebarHistoryItem } from "../components/MobileChatHome";
import type {
  AgentPayload,
  MobileAssistantPart,
  ChatSelectedSkill,
  MobileToolCard,
  MobileToolCardStatus,
  MobileConversationMessage,
  MobileSessionRunState,
  MobileSessionRunStatus,
  MobileSessionView,
  SessionDetail,
  SessionMessage,
  SessionMetadata,
  SessionRuntimeSelection,
  SessionTurnDraft,
  SessionTurnDraftPendingQuestion,
  SessionTurnDraftTool,
  SessionToolCall,
  SessionToolResult,
  StatusMessage,
  StoredMobileConversation,
} from "../mobileTypes";
import { parseToolTagText, type ParsedToolTagCall } from "./mobileToolTags";
import { parseAgentMessageWithSelectedSkill } from "./selectedSkillMessage";

export function mergeHistoryItems(input: {
  bridgeConnected: boolean;
  pinned: Set<string>;
  sessionViews: Record<string, MobileSessionView>;
  sessions: SessionMetadata[];
  sessionsLoaded: boolean;
  storedConversations: StoredMobileConversation[];
}): SidebarHistoryItem[] {
  const items = new Map<string, SidebarHistoryItem>();
  const storedById = new Map(input.storedConversations.map((conversation) => [conversation.id, conversation]));
  const bridgeLoaded = input.bridgeConnected && input.sessionsLoaded;
  const baseConversations = bridgeLoaded ? [] : input.storedConversations;

  for (const conversation of baseConversations) {
    items.set(conversation.id, createHistoryItem(conversation.id, {
      pinned: input.pinned,
      stored: conversation,
      view: input.sessionViews[conversation.id],
    }));
  }
  if (bridgeLoaded) {
    for (const session of input.sessions) {
      items.set(session.id, createHistoryItem(session.id, {
        pinned: input.pinned,
        session,
        stored: storedById.get(session.id),
        view: input.sessionViews[session.id],
      }));
    }
  }
  for (const view of Object.values(input.sessionViews)) {
    if (items.has(view.id) || !shouldShowLocalSessionView(view, bridgeLoaded)) {
      continue;
    }
    items.set(view.id, createHistoryItem(view.id, { pinned: input.pinned, view }));
  }

  return [...items.values()];
}

export function reconcileStoredConversationsWithBridge(
  storedConversations: StoredMobileConversation[],
  sessions: SessionMetadata[],
  keepIds: Set<string> = new Set(),
): StoredMobileConversation[] {
  const sessionsById = new Map(sessions.map((session) => [session.id, session]));
  return storedConversations
    .filter((conversation) => sessionsById.has(conversation.id) || keepIds.has(conversation.id))
    .map((conversation) => {
      const session = sessionsById.get(conversation.id);
      if (!session) {
        return conversation;
      }
      return {
        ...conversation,
        title: session.title.trim() || conversation.title,
      };
    });
}

export function syncSessionViewTitles(
  current: Record<string, MobileSessionView>,
  sessions: SessionMetadata[],
): Record<string, MobileSessionView> {
  let changed = false;
  const next = { ...current };
  for (const session of sessions) {
    const title = session.title.trim();
    if (!next[session.id]) {
      continue;
    }
    const nextTitle = title || next[session.id].title;
    if (next[session.id].bridgeOwned && next[session.id].title === nextTitle) {
      continue;
    }
    changed = true;
    next[session.id] = {
      ...next[session.id],
      bridgeOwned: true,
      title: nextTitle,
      updatedAt: session.updated_at,
    };
  }
  return changed ? next : current;
}

export function upsertSessionView(
  current: Record<string, MobileSessionView>,
  sessionId: string,
  patch: Partial<Omit<MobileSessionView, "id">>,
): Record<string, MobileSessionView> {
  const id = sessionId.trim();
  if (!id) {
    return current;
  }

  const existing = current[id];
  const now = new Date().toISOString();
  return {
    ...current,
    [id]: {
      id,
      bridgeOwned: patch.bridgeOwned ?? existing?.bridgeOwned ?? false,
      hasOlderHistory: patch.hasOlderHistory ?? existing?.hasOlderHistory ?? false,
      loadingOlderHistory: patch.loadingOlderHistory ?? existing?.loadingOlderHistory ?? false,
      messages: patch.messages ?? existing?.messages ?? [],
      nextHistoryBefore: hasOwnProperty(patch, "nextHistoryBefore")
        ? patch.nextHistoryBefore
        : existing?.nextHistoryBefore ?? null,
      reply: hasOwnProperty(patch, "reply") ? patch.reply : existing?.reply,
      run: patch.run ?? existing?.run ?? createIdleRunState(),
      title: patch.title ?? existing?.title ?? sessionFallbackTitle(id),
      unread: patch.unread ?? existing?.unread ?? false,
      updatedAt: patch.updatedAt ?? existing?.updatedAt ?? now,
    },
  };
}

export function createIdleRunState(statusText = ""): MobileSessionRunState {
  return {
    sessionEnded: false,
    status: "idle",
    statusText,
  };
}

export function createRunningRunState(
  requestId?: string,
  traceId?: string,
  statusText = "运行中",
): MobileSessionRunState {
  return {
    requestId,
    sessionEnded: false,
    status: "running",
    statusText,
    traceId,
  };
}

export function createSuccessRunState(
  statusText: string,
  sessionEnded = false,
  traceId?: string,
): MobileSessionRunState {
  return {
    sessionEnded,
    status: "success",
    statusText,
    traceId,
  };
}

export function isActiveSessionTurnDraft(
  draft: SessionTurnDraft | null | undefined,
): draft is SessionTurnDraft {
  return draft?.status === "streaming" || draft?.status === "awaiting_human";
}

export function sessionTurnDraftToRunState(draft: SessionTurnDraft): MobileSessionRunState {
  if (draft.status === "error") {
    return {
      sessionEnded: false,
      status: "error",
      statusText: draft.error?.trim() || "运行失败",
      traceId: draft.trace_id,
    };
  }

  return createRunningRunState(
    undefined,
    draft.trace_id,
    draft.status === "awaiting_human" ? "等待用户输入" : "生成中",
  );
}

export function sessionTurnDraftToAgentPayload(
  draft: SessionTurnDraft,
  sessionId: string,
  runtimeSelection?: SessionRuntimeSelection | null,
): AgentPayload {
  const parts = buildSessionTurnDraftParts(draft, runtimeSelection);
  const message = draft.assistant_segments.map((segment) => segment.content).join("");
  const thinking = draft.thinking_segments.map((segment) => segment.content).join("");
  const tools = parts.flatMap((part) => part.kind === "tool" ? [part.tool] : []);

  return {
    message,
    parts: parts.length > 0 ? parts : undefined,
    session_ended: false,
    session_id: sessionId,
    thinking: thinking || undefined,
    tools: tools.length > 0 ? tools : undefined,
  };
}

export function statusToRunState(
  status: StatusMessage,
  requestId?: string,
  traceId?: string,
): MobileSessionRunState {
  return {
    requestId,
    sessionEnded: status.text === "会话已结束",
    status: statusToneToRunStatus(status.tone),
    statusText: status.text,
    traceId,
  };
}

export function runStateToStatus(run: MobileSessionRunState, idleStatus: StatusMessage): StatusMessage {
  if (run.status === "idle") {
    return run.statusText ? { tone: "idle", text: run.statusText } : idleStatus;
  }
  return {
    tone: run.status === "running" ? "loading" : run.status,
    text: run.statusText,
  };
}

export function createUserConversationMessage(
  text: string,
  sessionId: string,
  selectedSkill?: ChatSelectedSkill,
): MobileConversationMessage {
  const now = Date.now();
  return {
    id: `${sessionId || "pending"}:user:${now}`,
    role: "user",
    selectedSkill,
    sessionId: sessionId || undefined,
    text,
  };
}

export function createAssistantConversationMessage(reply: AgentPayload, sessionId: string): MobileConversationMessage {
  const now = Date.now();
  const parts = cloneAssistantParts(reply.parts) ?? buildAssistantPartsFromReplyFields(reply, sessionId, now);
  return {
    id: `${sessionId}:assistant:${now}`,
    ...(parts ? { parts } : {}),
    role: "assistant",
    sessionId,
    text: reply.message,
    thinking: reply.thinking,
    tools: reply.tools,
  };
}

export function normalizeConversationSessionIds(
  messages: MobileConversationMessage[],
  sessionId: string,
): MobileConversationMessage[] {
  return messages.map((message) => ({
    ...message,
    sessionId,
  }));
}

export function sessionDetailToConversationMessages(detail: SessionDetail): MobileConversationMessage[] {
  const messages: MobileConversationMessage[] = [];
  const assistantByToolCallId = new Map<string, MobileConversationMessage>();
  let lastAssistant: MobileConversationMessage | undefined;

  for (const message of detail.messages) {
    if (message.role === "tool") {
      mergeToolMessageIntoAssistant(message, detail.id, assistantByToolCallId, lastAssistant);
      continue;
    }

    if (message.role !== "user" && message.role !== "assistant") {
      lastAssistant = undefined;
      continue;
    }

    const conversationMessage = sessionMessageToConversationMessage(message, detail.id);
    if (!conversationMessage) {
      lastAssistant = undefined;
      continue;
    }

    messages.push(conversationMessage);
    if (conversationMessage.role !== "assistant") {
      lastAssistant = undefined;
      continue;
    }

    lastAssistant = conversationMessage;
    for (const tool of conversationMessage.tools ?? []) {
      if (tool.toolCallId?.trim()) {
        assistantByToolCallId.set(tool.toolCallId.trim(), conversationMessage);
      }
    }
  }

  return messages;
}

export function sessionFallbackTitle(sessionId: string): string {
  const shortId = sessionId.trim().slice(0, 8);
  return shortId ? `会话 ${shortId}` : "新会话";
}

export function findStoredTitle(
  conversations: StoredMobileConversation[],
  sessionId: string,
): string | undefined {
  return conversations.find((conversation) => conversation.id === sessionId)?.title.trim() || undefined;
}

export function emptyStoredConversation(sessionId: string): StoredMobileConversation {
  const now = new Date().toISOString();
  return {
    created_at: now,
    id: sessionId,
    messages: [],
    title: sessionFallbackTitle(sessionId),
    updated_at: now,
  };
}

export function createClientRunId(label: string): string {
  return `mobile-${label}-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

export function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

export function isAgentRunCancellationMessage(message: string): boolean {
  const normalized = message.trim().toLowerCase();
  if (!normalized) {
    return false;
  }
  return normalized === "agent run cancelled"
    || normalized === "agent stream closed before terminal event"
    || normalized.endsWith(": agent run cancelled")
    || normalized === "context canceled"
    || normalized.endsWith(": context canceled");
}

function createHistoryItem(
  id: string,
  input: {
    pinned: Set<string>;
    session?: SessionMetadata;
    stored?: StoredMobileConversation;
    view?: MobileSessionView;
  },
): SidebarHistoryItem {
  const run = input.view?.run;
  return {
    id,
    pinned: input.pinned.has(id),
    status: run?.status === "idle" ? undefined : run?.status,
    title: input.session?.title.trim() || input.view?.title.trim() || input.stored?.title.trim() || sessionFallbackTitle(id),
    unread: Boolean(input.view?.unread),
    updatedAt: input.session?.updated_at || input.view?.updatedAt || input.stored?.updated_at || new Date().toISOString(),
  };
}

function shouldShowLocalSessionView(view: MobileSessionView, bridgeLoaded: boolean): boolean {
  if (!bridgeLoaded) {
    return true;
  }
  return view.bridgeOwned && (view.run.status === "running" || view.messages.length > 0 || Boolean(view.reply));
}

function statusToneToRunStatus(tone: StatusMessage["tone"]): MobileSessionRunStatus {
  switch (tone) {
    case "loading":
      return "running";
    case "success":
      return "success";
    case "error":
      return "error";
    case "idle":
      return "idle";
    default:
      return "idle";
  }
}

function buildSessionTurnDraftParts(
  draft: SessionTurnDraft,
  runtimeSelection?: SessionRuntimeSelection | null,
): MobileAssistantPart[] {
  const textParts = new Map(
    draft.assistant_segments.map((segment): [string, MobileAssistantPart] => [
      segment.id,
      { id: segment.id, kind: "text", text: segment.content },
    ]),
  );
  const toolParts = new Map(
    draft.tools.map((tool): [string, MobileAssistantPart] => [
      tool.id,
      { id: tool.id, kind: "tool", tool: sessionTurnDraftToolToCard(draft, tool) },
    ]),
  );
  const questionParts = new Map(
    draft.pending_questions.map((question): [string, MobileAssistantPart] => {
      const card = pendingQuestionToToolCard(draft, question, runtimeSelection);
      return [question.question_id, { id: card.id, kind: "tool", tool: card }];
    }),
  );

  const parts: MobileAssistantPart[] = [];
  const usedTextIds = new Set<string>();
  const usedToolIds = new Set<string>();
  const usedQuestionIds = new Set<string>();

  for (const item of draft.item_order) {
    const [kind, id] = splitDraftOrderItem(item);
    if (kind === "assistant" && id) {
      appendDraftPart(parts, textParts.get(id));
      usedTextIds.add(id);
      continue;
    }
    if (kind === "tool" && id) {
      appendDraftPart(parts, toolParts.get(id));
      usedToolIds.add(id);
      continue;
    }
    if (kind === "question" && id) {
      appendDraftPart(parts, questionParts.get(id));
      usedQuestionIds.add(id);
    }
  }

  for (const segment of draft.assistant_segments) {
    if (!usedTextIds.has(segment.id)) {
      appendDraftPart(parts, textParts.get(segment.id));
    }
  }
  for (const tool of draft.tools) {
    if (!usedToolIds.has(tool.id)) {
      appendDraftPart(parts, toolParts.get(tool.id));
    }
  }
  for (const question of draft.pending_questions) {
    if (!usedQuestionIds.has(question.question_id)) {
      appendDraftPart(parts, questionParts.get(question.question_id));
    }
  }
  return parts;
}

function appendDraftPart(parts: MobileAssistantPart[], part: MobileAssistantPart | undefined): void {
  if (part) {
    parts.push(part);
  }
}

function splitDraftOrderItem(item: string): [string, string] {
  const separatorIndex = item.indexOf(":");
  if (separatorIndex < 0) {
    return ["", ""];
  }
  return [item.slice(0, separatorIndex), item.slice(separatorIndex + 1)];
}

function sessionTurnDraftToolToCard(
  draft: SessionTurnDraft,
  tool: SessionTurnDraftTool,
): MobileToolCard {
  const status = normalizeDraftToolStatus(draft.status, tool.tool_status);
  return {
    id: tool.id,
    input: tool.tool_input || (status === "pending" || status === "running" ? tool.content || undefined : undefined),
    output: status === "success" ? tool.content || undefined : undefined,
    error: status === "error" ? tool.content || draft.error || undefined : undefined,
    status,
    toolCallId: tool.tool_call_id,
    toolName: tool.tool_name,
    traceId: tool.trace_id || draft.trace_id,
  };
}

function pendingQuestionToToolCard(
  draft: SessionTurnDraft,
  question: SessionTurnDraftPendingQuestion,
  runtimeSelection?: SessionRuntimeSelection | null,
): MobileToolCard {
  const isCodex = runtimeSelection?.runtime === "codex";
  const approvalFields = isCodex
    ? {
      approvalId: question.question_id,
      approvalPrompt: question.prompt,
    }
    : {};
  return {
    ...approvalFields,
    id: `stream-question:${draft.trace_id}:${question.question_id}`,
    input: question.prompt,
    status: "pending",
    toolCallId: isCodex ? `approval:${question.question_id}` : question.question_id,
    toolName: isCodex ? "codex_approval" : "ask_human",
    traceId: draft.trace_id,
  };
}

function normalizeDraftToolStatus(
  draftStatus: SessionTurnDraft["status"],
  rawStatus: string | undefined,
): MobileToolCardStatus {
  const status = rawStatus === "running" || rawStatus === "success" || rawStatus === "error"
    ? rawStatus
    : "pending";
  if (draftStatus === "error" && (status === "pending" || status === "running")) {
    return "error";
  }
  return status;
}

function sessionMessageToConversationMessage(
  message: SessionMessage,
  sessionId: string,
): MobileConversationMessage | null {
  const rawText = sessionMessageText(message);
  const parsedSelectedSkill = message.role === "user"
    ? parseAgentMessageWithSelectedSkill(rawText)
    : { message: rawText };
  const parsedToolTags = message.role === "assistant" ? parseToolTagText(rawText) : undefined;
  const assistantContent = message.role === "assistant"
    ? buildAssistantConversationContent(message, sessionId, parsedToolTags)
    : undefined;
  const text = parsedToolTags?.visibleText ?? parsedSelectedSkill.message;
  if (
    !text.trim()
    && !message.thinking?.trim()
    && (assistantContent?.tools.length ?? 0) === 0
    && !parsedSelectedSkill.selectedSkill
  ) {
    return null;
  }
  return {
    id: `${sessionId}:${message.index}:${message.role}`,
    ...(assistantContent?.parts.length ? { parts: assistantContent.parts } : {}),
    role: message.role === "assistant" ? "assistant" : "user",
    selectedSkill: parsedSelectedSkill.selectedSkill,
    sessionId,
    text,
    thinking: message.thinking,
    tools: assistantContent?.tools.length ? assistantContent.tools : undefined,
  };
}

function sessionMessageText(message: SessionMessage): string {
  if (message.text !== undefined) {
    return message.text;
  }
  return (message.content ?? [])
    .map((part) => part.text ?? "")
    .filter(Boolean)
    .join("\n");
}

function hasOwnProperty<T extends object>(value: T, key: PropertyKey): boolean {
  return Object.prototype.hasOwnProperty.call(value, key);
}

function buildAssistantConversationContent(
  message: SessionMessage,
  sessionId: string,
  parsedToolTags: ReturnType<typeof parseToolTagText> | undefined,
): { parts: MobileAssistantPart[]; tools: MobileToolCard[] } {
  const structuredTools = (message.tool_calls ?? []).map((toolCall) =>
    buildToolCardFromSessionToolCall(toolCall, sessionId, message.index),
  );
  const parts: MobileAssistantPart[] = [];

  if (parsedToolTags?.units.length) {
    let tagToolIndex = 0;
    let textPartIndex = 0;
    for (const unit of parsedToolTags.units) {
      if (unit.type === "text") {
        if (!unit.text) {
          continue;
        }
        parts.push(buildAssistantTextPart(sessionId, message.index, textPartIndex, unit.text));
        textPartIndex += 1;
        continue;
      }
      parts.push(buildAssistantToolPart(
        buildToolCardFromTagCall(unit, sessionId, message.index, tagToolIndex),
      ));
      tagToolIndex += 1;
    }
  } else {
    const text = sessionMessageText(message);
    if (text) {
      parts.push(buildAssistantTextPart(sessionId, message.index, 0, text));
    }
  }

  for (const tool of structuredTools) {
    parts.push(buildAssistantToolPart(tool));
  }

  return {
    parts,
    tools: parts.flatMap((part) => part.kind === "tool" ? [part.tool] : []),
  };
}

function buildToolCardFromSessionToolCall(
  toolCall: SessionToolCall,
  sessionId: string,
  messageIndex: number,
): MobileToolCard {
  return {
    id: `${sessionId}:${messageIndex}:tool-call:${toolCall.id}`,
    input: formatToolInput(toolCall.arguments),
    status: "pending",
    toolCallId: toolCall.id,
    toolName: toolCall.name,
  };
}

function buildToolCardFromTagCall(
  call: ParsedToolTagCall,
  sessionId: string,
  messageIndex: number,
  toolIndex: number,
): MobileToolCard {
  return {
    id: `${sessionId}:${messageIndex}:tag-tool:${toolIndex}`,
    input: call.argsText,
    status: "pending",
    toolName: `tool#${call.toolId}`,
  };
}

function mergeToolMessageIntoAssistant(
  message: SessionMessage,
  sessionId: string,
  assistantByToolCallId: Map<string, MobileConversationMessage>,
  lastAssistant: MobileConversationMessage | undefined,
): void {
  if (!message.tool_result) {
    return;
  }

  const toolCallId = message.tool_call_id?.trim();
  const target = toolCallId ? assistantByToolCallId.get(toolCallId) : undefined;
  const assistant = target ?? lastAssistant;
  if (!assistant) {
    return;
  }

  const resultCard = buildToolCardFromResult(message, sessionId);
  const tools = [...(assistant.tools ?? [])];
  const index = resolveToolResultMergeIndex(tools, toolCallId, Boolean(target));
  let mergedTool = resultCard;
  if (index >= 0) {
    mergedTool = {
      ...tools[index],
      ...resultCard,
      id: tools[index].id,
      input: tools[index].input ?? resultCard.input,
      toolCallId: tools[index].toolCallId ?? resultCard.toolCallId,
      toolName: resultCard.toolName ?? tools[index].toolName,
    };
    tools[index] = mergedTool;
  } else {
    tools.push(mergedTool);
  }

  assistant.tools = tools;
  assistant.parts = mergeToolCardIntoAssistantParts(assistant.parts, mergedTool);
  if (toolCallId) {
    assistantByToolCallId.set(toolCallId, assistant);
  }
}

function buildToolCardFromResult(message: SessionMessage, sessionId: string): MobileToolCard {
  const result = message.tool_result as SessionToolResult;
  const text = sessionMessageText(message);
  return {
    id: `${sessionId}:${message.index}:tool-result`,
    error: result.error,
    output: result.output ?? (result.status === "success" ? text || undefined : undefined),
    status: result.status,
    toolCallId: message.tool_call_id,
    toolName: result.tool,
    traceId: result.trace_id,
  };
}

function resolveToolResultMergeIndex(
  tools: MobileToolCard[],
  toolCallId: string | undefined,
  hasMatchedAssistant: boolean,
): number {
  if (toolCallId) {
    const matchedIndex = tools.findIndex((tool) => tool.toolCallId === toolCallId);
    if (matchedIndex >= 0) {
      return matchedIndex;
    }
  }

  if (!hasMatchedAssistant) {
    for (let index = tools.length - 1; index >= 0; index -= 1) {
      const tool = tools[index];
      if (tool.status === "pending" || tool.status === "running") {
        return index;
      }
    }
  }
  return -1;
}

function formatToolInput(input: Record<string, unknown>): string {
  return JSON.stringify(input, null, 2);
}

function buildAssistantTextPart(
  sessionId: string,
  messageIndex: number,
  textPartIndex: number,
  text: string,
): MobileAssistantPart {
  return {
    id: `${sessionId}:${messageIndex}:text:${textPartIndex}`,
    kind: "text",
    text,
  };
}

function buildAssistantToolPart(tool: MobileToolCard): MobileAssistantPart {
  return {
    id: tool.id,
    kind: "tool",
    tool,
  };
}

function mergeToolCardIntoAssistantParts(
  parts: MobileAssistantPart[] | undefined,
  tool: MobileToolCard,
): MobileAssistantPart[] {
  const next = parts ? [...parts] : [];
  const existingIndex = next.findIndex((part) =>
    part.kind === "tool" && (part.tool.id === tool.id || part.tool.toolCallId === tool.toolCallId));

  if (existingIndex < 0) {
    next.push(buildAssistantToolPart(tool));
    return next;
  }

  const existing = next[existingIndex];
  if (existing.kind !== "tool") {
    return next;
  }
  next[existingIndex] = {
    ...existing,
    tool: {
      ...existing.tool,
      ...tool,
    },
  };
  return next;
}

function cloneAssistantParts(parts: MobileAssistantPart[] | undefined): MobileAssistantPart[] | undefined {
  if (!parts?.length) {
    return undefined;
  }
  return parts.map((part) => {
    if (part.kind === "text") {
      return { ...part };
    }
    return {
      ...part,
      tool: { ...part.tool },
    };
  });
}

function buildAssistantPartsFromReplyFields(
  reply: AgentPayload,
  sessionId: string,
  now: number,
): MobileAssistantPart[] | undefined {
  const parts: MobileAssistantPart[] = [];
  if (reply.message) {
    parts.push({
      id: `${sessionId}:assistant:${now}:text`,
      kind: "text",
      text: reply.message,
    });
  }
  for (const tool of reply.tools ?? []) {
    parts.push(buildAssistantToolPart({ ...tool }));
  }
  return parts.length > 0 ? parts : undefined;
}
