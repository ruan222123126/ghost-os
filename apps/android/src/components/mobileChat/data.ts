import type { UiIconName } from "./types";

export const EMPTY_STATE_SUGGESTIONS = [
  { icon: "sparkles", text: "启动 Agent", tone: "blue" },
  { icon: "eye", text: "分析屏幕", tone: "purple" },
  { icon: "lightbulb", text: "制定执行计划", tone: "yellow" },
  { icon: "file-text", text: "总结会话", tone: "orange" },
] as const satisfies ReadonlyArray<{ icon: UiIconName; text: string; tone: "blue" | "orange" | "purple" | "yellow" }>;

export const COMPOSER_MENU_OPTIONS = [
  { id: "files", icon: "paperclip", label: "文件", unavailable: true },
  { id: "skills", icon: "puzzle", label: "技能", unavailable: false },
  { id: "features", icon: "settings", label: "功能", unavailable: false },
] as const satisfies ReadonlyArray<{
  id: "features" | "files" | "skills";
  icon: UiIconName;
  label: string;
  unavailable: boolean;
}>;
