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
