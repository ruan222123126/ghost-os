// @vitest-environment jsdom
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MobileSearchPage } from "./MobileSearchPage";
import type { SidebarHistoryItem } from "./MobileChatHome";
import type { SessionMetadata } from "../mobileTypes";

describe("MobileSearchPage", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    cleanup();
    vi.useRealTimers();
    vi.clearAllMocks();
  });

  it("calls backend search and displays returned sessions", async () => {
    const onSearchSessions = vi.fn(async () => [sessionMetadata("session-backend", "Backend result")]);

    renderSearchPage({ onSearchSessions });

    fireEvent.change(screen.getByPlaceholderText("搜索对话"), { target: { value: " needle " } });
    expect(onSearchSessions).not.toHaveBeenCalled();

    await flushSearchDelay();

    expect(onSearchSessions).toHaveBeenCalledWith("needle");
    expect(screen.getByText("Backend result")).toBeTruthy();
  });

  it("shows explicit backend errors", async () => {
    const onSearchSessions = vi.fn(async () => {
      throw new Error("backend unavailable");
    });

    renderSearchPage({ onSearchSessions });

    fireEvent.change(screen.getByPlaceholderText("搜索对话"), { target: { value: "needle" } });
    await flushSearchDelay();

    expect(screen.getByText("搜索失败：backend unavailable")).toBeTruthy();
  });

  it("does not request backend for empty input", async () => {
    const onSearchSessions = vi.fn(async () => [sessionMetadata("session-backend", "Backend result")]);

    renderSearchPage({ onSearchSessions });
    await flushSearchDelay();

    expect(onSearchSessions).not.toHaveBeenCalled();
    expect(screen.getByText("近期对话")).toBeTruthy();
    expect(screen.getByText("Recent task")).toBeTruthy();
  });

  it("requires connection for non-empty search", async () => {
    const onSearchSessions = vi.fn(async () => [sessionMetadata("session-backend", "Backend result")]);

    renderSearchPage({ bridgeConnected: false, onSearchSessions });

    fireEvent.change(screen.getByPlaceholderText("搜索对话"), { target: { value: "needle" } });

    expect(screen.getByText("需要先连接电脑端")).toBeTruthy();
    expect(onSearchSessions).not.toHaveBeenCalled();
  });
});

async function flushSearchDelay(): Promise<void> {
  await act(async () => {
    vi.advanceTimersByTime(500);
    await Promise.resolve();
  });
}

function renderSearchPage(options: {
  bridgeConnected?: boolean;
  historyItems?: SidebarHistoryItem[];
  onSearchSessions?: (query: string) => Promise<SessionMetadata[]>;
} = {}) {
  return render(
    <MobileSearchPage
      open
      bridgeConnected={options.bridgeConnected ?? true}
      historyItems={options.historyItems ?? [historyItem("session-recent", "Recent task")]}
      onClose={vi.fn()}
      onSearchSessions={options.onSearchSessions ?? vi.fn(async () => [])}
      onSelectHistory={vi.fn()}
    />,
  );
}

function historyItem(id: string, title: string): SidebarHistoryItem {
  return {
    id,
    pinned: false,
    title,
    updatedAt: "2026-01-02T00:00:00.000Z",
  };
}

function sessionMetadata(id: string, title: string): SessionMetadata {
  return {
    id,
    title,
    created_at: "2026-01-01T00:00:00.000Z",
    updated_at: "2026-01-03T00:00:00.000Z",
    message_count: 2,
    token_count: 12,
  };
}
