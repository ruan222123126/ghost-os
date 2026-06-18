import type { SidebarHistoryItem, ToolCardViewModel, UiIconName } from "./types";

export const INITIAL_HISTORY_LIST: SidebarHistoryItem[] = [
  { id: 1, title: "Bridge 连接与移动端任务", pinned: true },
  { id: 2, title: "Agent 执行链路设计", pinned: false },
  { id: 3, title: "Tauri App UI Development wi...", pinned: false },
  { id: 4, title: "移动端卡顿原因与优化建议", pinned: false },
  { id: 5, title: "AI Agent 手机端连接方案", pinned: false },
  { id: 6, title: "Bridge Runtime 参数说明", pinned: false },
];

export const DRAWING_PLACEHOLDERS = [
  { id: 1, title: "绘画草稿 01", desc: "角色设定、姿态参考、画面比例" },
  { id: 2, title: "场景氛围板", desc: "光线、色彩、空间层次" },
  { id: 3, title: "移动端图标方案", desc: "线稿、填色、导出规格" },
  { id: 4, title: "产品概念图", desc: "三视图、材质、局部细节" },
  { id: 5, title: "启动页插画", desc: "主视觉、背景、留白区域" },
  { id: 6, title: "头像变体", desc: "表情、服饰、风格统一" },
  { id: 7, title: "工作流缩略图", desc: "节点、连线、状态标识" },
  { id: 8, title: "空状态插画", desc: "轻量占位、低对比背景" },
];

export const MOCK_TOOL_CARDS = [
  {
    title: "pnpm build --filter ghost-os-mobile",
    details: "vite v7.3.5 building for production...\ntransforming modules...\nrendering chunks...",
    statusLabel: "运行中",
    tone: "running",
  },
  {
    title: "cat apps/android/src/components/mobileChat/Messages.tsx",
    details: "AssistantReply now renders Markdown content and keeps session metadata secondary.",
    statusLabel: "完成",
    tone: "success",
  },
  {
    title: "adb shell am start -n ghost.os/.MainActivity",
    details: "Error: device offline\nCheck the USB debugging session before retrying.",
    statusLabel: "失败",
    tone: "error",
  },
] as const satisfies ReadonlyArray<ToolCardViewModel>;

export const EMPTY_STATE_SUGGESTIONS = [
  { icon: "sparkles", text: "启动 Agent", tone: "blue" },
  { icon: "eye", text: "分析屏幕", tone: "purple" },
  { icon: "lightbulb", text: "制定执行计划", tone: "yellow" },
  { icon: "file-text", text: "总结会话", tone: "orange" },
] as const satisfies ReadonlyArray<{ icon: UiIconName; text: string; tone: "blue" | "orange" | "purple" | "yellow" }>;

export const COMPOSER_MENU_OPTIONS = [
  { icon: "camera", label: "相机", unavailable: true },
  { icon: "image", label: "照片", unavailable: true },
  { icon: "paperclip", label: "文件", unavailable: true },
  { icon: "puzzle", label: "技能", unavailable: false },
] as const satisfies ReadonlyArray<{ icon: UiIconName; label: string; unavailable: boolean }>;
