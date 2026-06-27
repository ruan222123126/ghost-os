// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ChatHeader } from "./ChatHeader";

describe("ChatHeader", () => {
  afterEach(() => {
    cleanup();
  });

  it("shows connection action instead of new session on the empty home header", () => {
    const onNewSession = vi.fn();
    const onOpenConnection = vi.fn();

    render(
      <ChatHeader
        agentRuntime="ghost"
        runtimeLabel="Bridge Runtime"
        config={undefined}
        providerList={undefined}
        status={{ tone: "idle", text: "未连接" }}
        hasConversation={false}
        runtimeMenuOpen={false}
        onOpenSidebar={vi.fn()}
        onToggleRuntimeMenu={vi.fn()}
        onCloseRuntimeMenu={vi.fn()}
        onSwitchAgentRuntime={vi.fn()}
        onSwitchModel={vi.fn()}
        onOpenConnection={onOpenConnection}
        onOpenMoreMenu={vi.fn()}
        onNewSession={onNewSession}
      />,
    );

    expect(screen.queryByRole("button", { name: "新会话" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "连接" }));

    expect(onOpenConnection).toHaveBeenCalledTimes(1);
    expect(onNewSession).not.toHaveBeenCalled();
  });
});
