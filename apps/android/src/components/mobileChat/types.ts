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
  | "terminal"
  | "trash"
  | "x";

export interface SidebarHistoryItem {
  id: string;
  title: string;
  updatedAt: string;
  pinned: boolean;
  status?: "running" | "success" | "error";
  unread?: boolean;
}
