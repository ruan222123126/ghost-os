import type { MobileToolCard } from "../mobileTypes";

export type MobileToolTone = "running" | "success" | "error";
export type MobileToolCardTitleMode = "status" | "plain";

export interface MobileToolCardViewModel {
  title: string;
  tone: MobileToolTone;
  statusLabel: string;
  details: string;
  titleMode: MobileToolCardTitleMode;
  showTerminalIcon: boolean;
}

interface MobileToolCardViewModelOptions {
  fallbackTitle?: string;
  preparingDetails?: string;
}

interface ActionItem {
  kind: ActionKind;
  text: string;
}

interface FormattedToolAction {
  actionKind?: ActionKind;
  actionText?: string;
  text: string;
  variant: "action" | "command" | "default";
}

type ActionKind = "list" | "load" | "unload" | "read" | "write" | "edit" | "run" | "screen" | "web" | "search" | "codex";
type RecordValue = Record<string, unknown>;

const TOOL_ACTION_LIMIT = 80;
const TOOL_ACTION_TRUNCATE_AT = 77;
const SUMMARY_LIMIT = 80;
const SUMMARY_TRUNCATE_AT = 77;
const RUNNING_STATUSES = new Set(["running", "pending", "in_progress"]);
const ERROR_STATUSES = new Set(["error", "failed"]);
const TOOL_FALLBACK_TITLE = "工具";
const TOOL_PREFIX = "tools.";
const SCRIPT_EXEC_TOOL = "script_exec";
const FILE_ACTION_LABELS = { edit: "编辑文件", read: "阅读文件", write: "创建文件" } as const;
const SKILL_ACTION_LABELS = { load: "加载技能", unload: "卸载技能" } as const;
const SCREEN_ACTION_LABELS = { screen: "截取屏幕" } as const;
const TOOL_ALIASES: Record<string, string> = {
  apply_diff: "apply_diff", bash: "bash_exec", bash_exec: "bash_exec",
  codex: "codex_cli", codex_cli: "codex_cli", edit: "apply_diff",
  fetch_web: "fetch_webpage", fetch_webpage: "fetch_webpage", patch: "apply_diff",
  read: "read_file", read_file: "read_file", run: "bash_exec",
  screen_action: "screen_action", screen_control: "screen_control",
  script: SCRIPT_EXEC_TOOL, script_exec: SCRIPT_EXEC_TOOL,
  search: "search_files", search_files: "search_files",
  web: "fetch_webpage", web_search: "web_search",
  write: "write_file", write_file: "write_file",
};
const PROMOTED_ACTION_TOOL_NAMES = new Set([
  "apply_diff", "bash_exec", "codex_cli", "fetch_webpage", "list_files", "read_file",
  "screen_action", "screen_control", "search_files", "sfind", "web_search", "write_file",
]);

export function buildMobileToolCardViewModel(
  tool: MobileToolCard,
  options: MobileToolCardViewModelOptions = {},
): MobileToolCardViewModel {
  const action = formatToolAction(tool);
  const tone = getMobileToolTone(tool.status);
  const plainTitle = buildPlainActionTitle(action, tool);
  const details = formatToolCardDetails(tool);

  return {
    title: plainTitle || buildToolCardTitle(action, options),
    tone,
    statusLabel: getMobileToolStatusLabel(tone, tool.status),
    details: details || options.preparingDetails || "正在准备工具输出…",
    titleMode: plainTitle ? "plain" : "status",
    showTerminalIcon: !plainTitle,
  };
}

export function getMobileToolTone(status?: string): MobileToolTone {
  const normalized = normalizeStatus(status);
  if (!normalized || RUNNING_STATUSES.has(normalized)) {
    return "running";
  }
  if (ERROR_STATUSES.has(normalized)) {
    return "error";
  }
  return "success";
}

export function getMobileToolStatusLabel(tone: MobileToolTone, status?: string): string {
  if (tone === "running") {
    return "RUNNING";
  }
  if (tone === "error") {
    return "FAILED";
  }
  const normalized = normalizeStatus(status);
  if (!normalized || normalized === "ok" || normalized === "done" || normalized === "completed") {
    return "SUCCESS";
  }
  return normalized.replace(/_/g, " ").toUpperCase();
}

function formatToolAction(tool: MobileToolCard): FormattedToolAction {
  const bashCommand = resolveBashExecCommand(tool);
  if (bashCommand) {
    return { text: bashCommand, variant: "command" };
  }

  const promotedAction = resolvePromotedToolAction(tool);
  if (promotedAction) {
    return {
      actionKind: promotedAction.kind,
      actionText: promotedAction.text,
      text: promotedAction.text ? `${promotedAction.kind} ${promotedAction.text}` : promotedAction.kind,
      variant: "action",
    };
  }

  if (tool.toolName) {
    return { text: tool.toolName, variant: "default" };
  }

  const fallback = firstNonEmptyLine(tool.output, tool.input, tool.error);
  if (!fallback) {
    return { text: TOOL_FALLBACK_TITLE, variant: "default" };
  }
  if (fallback.length <= TOOL_ACTION_LIMIT) {
    return { text: fallback, variant: "default" };
  }
  return {
    text: `${fallback.slice(0, TOOL_ACTION_TRUNCATE_AT)}...`,
    variant: "default",
  };
}

function buildToolCardTitle(action: FormattedToolAction, options: MobileToolCardViewModelOptions): string {
  if (action.text === TOOL_FALLBACK_TITLE) {
    return options.fallbackTitle ?? TOOL_FALLBACK_TITLE;
  }
  return action.text;
}

function buildPlainActionTitle(action: FormattedToolAction, tool: MobileToolCard): string {
  const fileActionTitle = buildFileActionTitle(action);
  if (fileActionTitle) {
    return fileActionTitle;
  }

  const toolName = normalizeToolName(tool.toolName);
  if ((toolName === "web_search" || toolName === "fetch_webpage") && action.actionKind === "web") {
    return joinLabelAndTarget("网络搜索", action.actionText);
  }
  if (toolName === "codex_cli" && action.actionKind === "codex") {
    return joinLabelAndTarget("调用codex", action.actionText);
  }
  if (toolName === "search_files" && action.actionKind === "search") {
    return buildKeywordSearchTitle(action.actionText);
  }
  if (toolName === "sfind" && isSkillToolAction(action)) {
    return joinLabelAndTarget(SKILL_ACTION_LABELS[action.actionKind], action.actionText);
  }
  if ((toolName === "screen_action" || toolName === "screen_control") && isScreenToolAction(action)) {
    return joinLabelAndTarget(SCREEN_ACTION_LABELS[action.actionKind], action.actionText);
  }
  return "";
}

function buildFileActionTitle(action: FormattedToolAction): string {
  if (!isFileToolAction(action)) {
    return "";
  }
  return joinLabelAndTarget(FILE_ACTION_LABELS[action.actionKind], action.actionText);
}

function formatToolCardDetails(tool: MobileToolCard): string {
  const details: string[] = [];
  if (tool.error?.trim()) {
    details.push(`error: ${normalizeInline(tool.error)}`);
  }
  details.push(...collectUniqueOutputParts([tool.output]));
  return details.join("\n");
}

function resolveBashExecCommand(tool: MobileToolCard): string {
  if (normalizeToolName(tool.toolName) !== "bash_exec") {
    return "";
  }
  return readBashExecCommand(tool.input) || readBashExecCommand(tool.output);
}

function readBashExecCommand(rawText?: string): string {
  const parsedCommand = readBashExecCommandArgs(parseRecord(rawText));
  if (parsedCommand) {
    return parsedCommand;
  }
  const matched = rawText?.match(/"(?:command|cmd)"\s*:\s*"([^"]*)/s);
  return normalizeInline(matched?.[1]);
}

function readBashExecCommandArgs(args?: RecordValue): string {
  return normalizeInline(readString(args, "command") || readString(args, "cmd"));
}

function resolvePromotedToolAction(tool: MobileToolCard): ActionItem | undefined {
  const toolName = normalizeToolName(tool.toolName);
  if (!supportsPromotedActionTitle(toolName) || toolName === "bash_exec") {
    return undefined;
  }
  if (toolName === "sfind") {
    return buildSkillAction(parseRecord(tool.output) || parseRecord(tool.input));
  }
  return buildDirectAction(toolName, parseRecord(tool.input) || parseRecord(tool.output));
}

function buildDirectAction(toolName: string, args?: RecordValue): ActionItem | undefined {
  if (toolName === "codex_cli") {
    return { kind: "codex", text: formatCodexTarget(args) };
  }
  if (toolName === "read_file") {
    return { kind: "read", text: formatPath(readString(args, "path")) };
  }
  if (toolName === "list_files") {
    return { kind: "list", text: formatPath(readString(args, "path")) || "." };
  }
  if (toolName === "write_file") {
    return { kind: "write", text: appendDelta(formatPath(readString(args, "path")), resolveWriteDelta(args)) };
  }
  if (toolName === "apply_diff") {
    return { kind: "edit", text: appendDelta(formatPath(readString(args, "path")), resolveEditDelta(args)) };
  }
  if (toolName === "screen_action" || toolName === "screen_control") {
    return buildScreenAction(args);
  }
  if (toolName === "fetch_webpage") {
    return { kind: "web", text: truncateSummary(formatWebTarget(readString(args, "url"))) };
  }
  if (toolName === "web_search") {
    return { kind: "web", text: truncateSummary(readString(args, "query")) };
  }
  if (toolName === "search_files") {
    return { kind: "search", text: truncateSummary(readString(args, "query")) };
  }
  return undefined;
}

function buildSkillAction(payload?: RecordValue): ActionItem | undefined {
  const action = normalizeToolName(readString(payload, "action"));
  if (action !== "load" && action !== "unload") {
    return undefined;
  }
  return { kind: action, text: readFirstItemName(payload) };
}

function buildScreenAction(args?: RecordValue): ActionItem | undefined {
  const action = normalizeToolName(readString(args, "action"));
  if (action === "screenshot") {
    return { kind: "screen", text: "" };
  }
  return undefined;
}

function resolveWriteDelta(args?: RecordValue): { added?: number; removed?: number } {
  return {
    added: readPositiveInt(readRecord(args, "content_summary"), "lines") || countLines(readString(args, "content")),
    removed: readPositiveInt(args, "removed_lines"),
  };
}

function resolveEditDelta(args?: RecordValue): { added?: number; removed?: number } {
  const diffSummary = readRecord(args, "diff_summary");
  const counted = countUnifiedDiffDelta(readString(args, "diff_text"));
  return {
    added: readPositiveInt(diffSummary, "added_lines") || counted.added,
    removed: readPositiveInt(diffSummary, "removed_lines") || counted.removed,
  };
}

function countUnifiedDiffDelta(diffText: string): { added?: number; removed?: number } {
  if (!diffText.trim()) {
    return {};
  }
  let added = 0;
  let removed = 0;
  let inHunk = false;
  for (const line of diffText.split(/\r?\n/)) {
    if (line.startsWith("@@")) {
      inHunk = true;
      continue;
    }
    if (!inHunk || line.startsWith("\\")) {
      continue;
    }
    if (line.startsWith("+")) {
      added += 1;
      continue;
    }
    if (line.startsWith("-")) {
      removed += 1;
    }
  }
  return {
    added: added || undefined,
    removed: removed || undefined,
  };
}

function appendDelta(baseText: string, delta: { added?: number; removed?: number }): string {
  const parts: string[] = [];
  if (delta.added) {
    parts.push(`+${delta.added}`);
  }
  if (delta.removed) {
    parts.push(`-${delta.removed}`);
  }
  if (parts.length === 0) {
    return baseText;
  }
  return baseText ? `${baseText} ${parts.join(" ")}` : parts.join(" ");
}

function normalizeToolName(rawName?: string): string {
  const normalized = rawName?.trim().toLowerCase() || "";
  if (!normalized) {
    return "";
  }
  const withoutPrefix = normalized.startsWith(TOOL_PREFIX)
    ? normalized.slice(TOOL_PREFIX.length)
    : normalized;
  if (TOOL_ALIASES[withoutPrefix]) {
    return TOOL_ALIASES[withoutPrefix];
  }
  for (const suffix of ["_exec", "_file"]) {
    if (!withoutPrefix.endsWith(suffix)) {
      continue;
    }
    const baseName = withoutPrefix.slice(0, withoutPrefix.length - suffix.length);
    if (TOOL_ALIASES[baseName]) {
      return TOOL_ALIASES[baseName];
    }
  }
  return withoutPrefix;
}

function supportsPromotedActionTitle(toolName?: string): boolean {
  return PROMOTED_ACTION_TOOL_NAMES.has(normalizeToolName(toolName));
}

function parseRecord(rawText?: string): RecordValue | undefined {
  if (!rawText?.trim()) {
    return undefined;
  }
  try {
    const parsed: unknown = JSON.parse(rawText);
    if (typeof parsed === "object" && parsed !== null && !Array.isArray(parsed)) {
      return parsed as RecordValue;
    }
  } catch {
    return undefined;
  }
  return undefined;
}

function readRecord(value: RecordValue | undefined, key: string): RecordValue | undefined {
  const nested = value?.[key];
  if (typeof nested !== "object" || nested === null || Array.isArray(nested)) {
    return undefined;
  }
  return nested as RecordValue;
}

function readString(value: RecordValue | undefined, key: string): string {
  const raw = value?.[key];
  return typeof raw === "string" ? raw.trim() : "";
}

function readPositiveInt(value: RecordValue | undefined, key: string): number | undefined {
  const raw = value?.[key];
  const parsed = typeof raw === "number" ? raw : Number.parseInt(typeof raw === "string" ? raw : "", 10);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : undefined;
}

function readFirstItemName(payload?: RecordValue): string {
  const items = payload?.items;
  if (!Array.isArray(items)) {
    return "";
  }
  const firstItem = items.find((item) => typeof item === "object" && item !== null && !Array.isArray(item));
  return readString(firstItem as RecordValue | undefined, "name");
}

function collectUniqueOutputParts(values: Array<string | undefined>): string[] {
  const outputParts: string[] = [];
  for (const value of values) {
    if (!value?.trim()) {
      continue;
    }
    if (outputParts.some((current) => current.trim() === value.trim())) {
      continue;
    }
    outputParts.push(value);
  }
  return outputParts;
}

function firstNonEmptyLine(...values: Array<string | undefined>): string {
  for (const value of values) {
    const line = value?.split("\n")[0]?.trim();
    if (line) {
      return line;
    }
  }
  return "";
}

function formatCodexTarget(args?: RecordValue): string {
  return truncateSummary([readString(args, "op"), readString(args, "prompt")].filter(Boolean).join(" "));
}

function formatWebTarget(rawURL: string): string {
  if (!rawURL) {
    return "";
  }
  try {
    const parsed = new URL(rawURL);
    const pathname = parsed.pathname === "/" ? "" : parsed.pathname;
    return `${parsed.hostname}${pathname}${parsed.search}`;
  } catch {
    return rawURL;
  }
}

function formatPath(rawPath: string): string {
  if (!rawPath) {
    return "";
  }
  const normalized = rawPath.replace(/\\/g, "/");
  if (normalized === "/" || /^[A-Za-z]:\/$/.test(normalized)) {
    return normalized;
  }
  return normalized.replace(/\/+$/, "");
}

function countLines(content: string): number | undefined {
  if (!content.trim()) {
    return undefined;
  }
  return content.split("\n").length;
}

function truncateSummary(value: string): string {
  const normalized = normalizeInline(value);
  if (normalized.length <= SUMMARY_LIMIT) {
    return normalized;
  }
  return `${normalized.slice(0, SUMMARY_TRUNCATE_AT)}...`;
}

function normalizeInline(value?: string): string {
  return value?.replace(/\s+/g, " ").trim() || "";
}

function normalizeStatus(status?: string): string {
  return status?.trim().toLowerCase() || "";
}

function buildKeywordSearchTitle(rawTarget?: string): string {
  const target = rawTarget?.trim() || "";
  return target ? `搜索 ${target}（关键词）` : "搜索（关键词）";
}

function joinLabelAndTarget(label: string, rawTarget?: string): string {
  const target = rawTarget?.trim() || "";
  return target ? `${label} ${target}` : label;
}

function isFileToolAction(
  action: FormattedToolAction,
): action is FormattedToolAction & { actionKind: keyof typeof FILE_ACTION_LABELS } {
  return action.actionKind === "read" || action.actionKind === "write" || action.actionKind === "edit";
}

function isSkillToolAction(
  action: FormattedToolAction,
): action is FormattedToolAction & { actionKind: keyof typeof SKILL_ACTION_LABELS } {
  return action.actionKind === "load" || action.actionKind === "unload";
}

function isScreenToolAction(
  action: FormattedToolAction,
): action is FormattedToolAction & { actionKind: keyof typeof SCREEN_ACTION_LABELS } {
  return action.actionKind === "screen";
}
