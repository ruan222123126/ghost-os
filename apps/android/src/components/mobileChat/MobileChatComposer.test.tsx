// @vitest-environment jsdom
import { createRef } from "react";
import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { MobileChatComposer, type MobileChatComposerHandle } from "./MobileChatComposer";

describe("MobileChatComposer", () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it("restores a rejected draft without coupling it to the conversation state", async () => {
    const onSend = vi.fn(async () => false);
    render(
      <MobileChatComposer
        agentMode={null}
        canEnableCodexMode={false}
        canSend
        canStop={false}
        loading={false}
        onChangeAgentMode={vi.fn()}
        onSend={onSend}
      />,
    );

    fireEvent.change(screen.getByRole("textbox"), { target: { value: "保留这条草稿" } });
    fireEvent.click(screen.getByRole("button", { name: "发送任务" }));

    await waitFor(() => {
      expect(screen.getByRole<HTMLTextAreaElement>("textbox").value).toBe("保留这条草稿");
    });
    expect(onSend).toHaveBeenCalledWith("保留这条草稿", undefined);
  });

  it("keeps the submitted draft cleared until the pending send finally fails", async () => {
    let finishSend: ((sent: boolean) => void) | undefined;
    const onSend = vi.fn(() => new Promise<boolean>((resolve) => {
      finishSend = resolve;
    }));
    render(
      <MobileChatComposer
        agentMode={null}
        canEnableCodexMode={false}
        canSend
        canStop={false}
        loading={false}
        onChangeAgentMode={vi.fn()}
        onSend={onSend}
      />,
    );

    fireEvent.change(screen.getByRole("textbox"), { target: { value: "重连期间不要恢复" } });
    fireEvent.click(screen.getByRole("button", { name: "发送任务" }));

    expect(screen.getByRole<HTMLTextAreaElement>("textbox").value).toBe("");
    act(() => finishSend?.(false));
    await waitFor(() => {
      expect(screen.getByRole<HTMLTextAreaElement>("textbox").value).toBe("重连期间不要恢复");
    });
  });

  it("exposes isolated reset and suggestion-fill controls", () => {
    const ref = createRef<MobileChatComposerHandle>();
    render(
      <MobileChatComposer
        ref={ref}
        agentMode={null}
        canEnableCodexMode={false}
        canSend
        canStop={false}
        loading={false}
        onChangeAgentMode={vi.fn()}
        onSend={vi.fn(async () => true)}
      />,
    );

    act(() => ref.current?.setDraft("从建议填入"));
    expect(screen.getByRole<HTMLTextAreaElement>("textbox").value).toBe("从建议填入");

    act(() => ref.current?.reset());
    expect(screen.getByRole<HTMLTextAreaElement>("textbox").value).toBe("");
  });
});
