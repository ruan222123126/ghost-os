// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { RuntimeMenu } from "./RuntimeMenu";

describe("RuntimeMenu", () => {
  afterEach(() => {
    cleanup();
  });

  it("switches the Codex model without updating the Ghost model", () => {
    const onSwitchCodexModel = vi.fn();
    const onSwitchModel = vi.fn(async () => true);
    const onClose = vi.fn();

    render(
      <RuntimeMenu
        agentRuntime="codex"
        codexModel="gpt-5.4"
        codexPermissionMode="safe-yolo"
        config={{ model: "gpt-4o", provider: "OpenAI" }}
        providerList={{ active_provider: "OpenAI", providers: [] }}
        status={{ tone: "success", text: "已连接" }}
        open
        onClose={onClose}
        onSwitchAgentRuntime={vi.fn()}
        onSwitchCodexModel={onSwitchCodexModel}
        onSwitchModel={onSwitchModel}
      />,
    );

    expect(screen.getByTitle("gpt-5.4").className).toContain("is-selected");

    fireEvent.click(screen.getByTitle("gpt-5.5"));

    expect(onSwitchCodexModel).toHaveBeenCalledWith("gpt-5.5");
    expect(onSwitchModel).not.toHaveBeenCalled();
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
