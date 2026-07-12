import type { SidebarHistoryItem } from "./types";

export function compareHistoryItems(a: SidebarHistoryItem, b: SidebarHistoryItem): number {
  if (a.pinned !== b.pinned) {
    return a.pinned ? -1 : 1;
  }
  const updatedOrder = b.updatedAt.localeCompare(a.updatedAt);
  if (updatedOrder !== 0) {
    return updatedOrder;
  }
  return a.title.localeCompare(b.title, "zh-Hans");
}
