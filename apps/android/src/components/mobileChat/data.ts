import type { UiIconName } from "./types";

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
