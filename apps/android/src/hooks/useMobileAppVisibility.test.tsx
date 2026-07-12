// @vitest-environment jsdom
import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  focusCallback: undefined as ((event: { payload: boolean }) => void) | undefined,
  isFocused: vi.fn(),
  stop: vi.fn(),
}));

vi.mock("@tauri-apps/api/window", () => ({
  getCurrentWindow: () => ({
    isFocused: mocks.isFocused,
    onFocusChanged: vi.fn(async (callback: (event: { payload: boolean }) => void) => {
      mocks.focusCallback = callback;
      return mocks.stop;
    }),
  }),
}));

vi.mock("../lib/bridgeBus", () => ({
  hasTauriRuntime: () => true,
}));

import { useMobileAppVisibility } from "./useMobileAppVisibility";

describe("useMobileAppVisibility", () => {
  beforeEach(() => {
    mocks.focusCallback = undefined;
    mocks.isFocused.mockReset().mockResolvedValue(true);
    mocks.stop.mockReset();
  });

  it("treats a backgrounded Tauri window as hidden even when the document stays visible", async () => {
    const { result, unmount } = renderHook(() => useMobileAppVisibility());
    await waitFor(() => expect(mocks.focusCallback).toBeDefined());
    expect(result.current).toBe(true);

    act(() => {
      mocks.focusCallback?.({ payload: false });
    });
    expect(result.current).toBe(false);

    act(() => {
      mocks.focusCallback?.({ payload: true });
    });
    expect(result.current).toBe(true);

    unmount();
    expect(mocks.stop).toHaveBeenCalled();
  });
});
