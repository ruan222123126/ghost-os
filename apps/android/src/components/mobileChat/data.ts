import type { ConversationPlaceholderSection, SidebarHistoryItem, UiIconName } from "./types";

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

export const CONVERSATION_PLACEHOLDER_SECTIONS: ConversationPlaceholderSection[] = [
  {
    title: "1. 运行环境与首屏表现",
    body: "先确认当前是在开发模式还是生产构建中观察卡顿。开发模式会保留更多校验和热更新逻辑，滚动、输入和组件重渲染都可能更重。",
    bullets: ["记录首屏加载时间", "检查资源体积与请求数量", "确认是否存在重复初始化"],
  },
  {
    title: "2. 交互卡顿排查",
    body: "会话页最容易暴露输入框、滚动容器和长文本渲染的问题。这里用较长内容撑开页面，便于观察顶部栏、底部输入框和向下滚动按钮的层级关系。",
    bullets: ["滚动区域应独立于底部输入框", "长文本不能挤压按钮", "弹出菜单遮罩需要阻止底层滚动"],
  },
  {
    title: "3. 状态与协议边界",
    body: "移动端只负责感知和交互，不直接做核心编排。发送消息后保留 trace_id，便于把 UI 操作和 Bridge 响应串起来。",
    bullets: ["请求保持 trace_id", "错误直接展示", "新会话只清空本地会话状态"],
  },
  {
    title: "4. 后续内容占位",
    body: "这段内容用于验证上下滑动效果。真实接入后可以替换为流式回复、工具执行状态、引用卡片或任务步骤。",
    bullets: ["占位段落一", "占位段落二", "占位段落三"],
  },
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
