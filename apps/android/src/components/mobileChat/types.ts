export type UiIconName =
  | "arrow-down"
  | "arrow-up"
  | "camera"
  | "check"
  | "chevron-down"
  | "edit"
  | "eye"
  | "file-text"
  | "image"
  | "lightbulb"
  | "menu"
  | "mic"
  | "more"
  | "paperclip"
  | "pencil"
  | "pin"
  | "pin-off"
  | "plus"
  | "puzzle"
  | "search"
  | "settings"
  | "sparkles"
  | "trash";

export interface SidebarHistoryItem {
  id: number;
  title: string;
  pinned: boolean;
}

export type ToolTone = "running" | "success" | "error";

export type ToolCardTitleMode = "status" | "plain";

export interface ToolCardViewModel {
  title: string;
  details: string;
  statusLabel: string;
  tone: ToolTone;
  titleMode?: ToolCardTitleMode;
  showTerminalIcon?: boolean;
}
