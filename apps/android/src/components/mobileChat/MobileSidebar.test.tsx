// @vitest-environment jsdom
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { StoredSettings } from "../../mobileTypes";
import { MobileSidebar } from "./MobileSidebar";
import type { SidebarHistoryItem } from "./types";

describe("MobileSidebar", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the initial history batch only", () => {
    renderSidebar({ historyItems: historyItems(45) });

    expect(screen.getByText("Task 00")).toBeTruthy();
    expect(screen.getByText("Task 29")).toBeTruthy();
    expect(screen.queryByText("Task 30")).toBeNull();
  });

  it("loads the next history batch near the scroll bottom", () => {
    const { container } = renderSidebar({ historyItems: historyItems(45) });
    const scrollElement = getSidebarScrollElement(container);
    setScrollMetrics(scrollElement, {
      clientHeight: 600,
      scrollHeight: 2000,
      scrollTop: 1250,
    });

    act(() => {
      fireEvent.scroll(scrollElement);
    });

    expect(screen.getByText("Task 44")).toBeTruthy();
  });

  it("keeps a deep active history item visible when opening", () => {
    renderSidebar({
      activeHistoryId: "session-40",
      historyItems: historyItems(45),
    });

    expect(screen.getByText("Task 40")).toBeTruthy();
    expect(screen.getByText("Task 44")).toBeTruthy();
  });
});

function renderSidebar(options: {
  activeHistoryId?: string;
  historyItems?: SidebarHistoryItem[];
  open?: boolean;
} = {}) {
  return render(
    <MobileSidebar
      open={options.open ?? true}
      host={undefined}
      config={undefined}
      settings={settings()}
      historyItems={options.historyItems ?? historyItems(1)}
      activeHistoryId={options.activeHistoryId}
      onClose={vi.fn()}
      onNewSession={vi.fn()}
      onOpenSearch={vi.fn()}
      onSelectHistory={vi.fn()}
      onConnect={vi.fn(async () => undefined)}
      onOpenSettings={vi.fn()}
    />,
  );
}

function historyItems(count: number): SidebarHistoryItem[] {
  return Array.from({ length: count }, (_, index) => ({
    id: `session-${index}`,
    pinned: false,
    title: `Task ${String(index).padStart(2, "0")}`,
    updatedAt: `2026-01-01T00:${String(59 - index).padStart(2, "0")}:00.000Z`,
  }));
}

function settings(): StoredSettings {
  return {
    autoConnectEnabled: false,
    bridgeUrl: "http://127.0.0.1:8080",
    connectionMode: "http",
    persistComputerSessionsEnabled: false,
    remoteExecutionEnabled: true,
  };
}

function getSidebarScrollElement(container: HTMLElement): HTMLDivElement {
  const element = container.querySelector(".sidebar-scroll");
  if (!(element instanceof HTMLDivElement)) {
    throw new Error("sidebar scroll element not found");
  }
  return element;
}

function setScrollMetrics(
  element: HTMLDivElement,
  metrics: {
    clientHeight: number;
    scrollHeight: number;
    scrollTop: number;
  },
): void {
  Object.defineProperty(element, "clientHeight", {
    configurable: true,
    value: metrics.clientHeight,
  });
  Object.defineProperty(element, "scrollHeight", {
    configurable: true,
    value: metrics.scrollHeight,
  });
  Object.defineProperty(element, "scrollTop", {
    configurable: true,
    value: metrics.scrollTop,
    writable: true,
  });
}
