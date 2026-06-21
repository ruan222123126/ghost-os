import { describe, expect, it } from "vitest";
import { buildMobileToolCardViewModel, getMobileToolStatusLabel, getMobileToolTone } from "./mobileToolCardViewModel";

describe("mobile tool card view model", () => {
  it("builds promoted command title, status tone, and output details", () => {
    expect(buildMobileToolCardViewModel({
      id: "tool-1",
      input: JSON.stringify({ command: "echo ok" }),
      output: "ok\nextra",
      status: "success",
      toolName: "bash_exec",
      traceId: "trace-1",
    })).toEqual({
      title: "echo ok",
      tone: "success",
      statusLabel: "SUCCESS",
      details: "ok\nextra",
      titleMode: "status",
      showTerminalIcon: true,
    });
  });

  it("uses preparing details for unnamed pending tools", () => {
    expect(buildMobileToolCardViewModel({
      id: "tool-2",
      status: "pending",
    }, {
      fallbackTitle: "工具",
      preparingDetails: "准备输出",
    })).toEqual({
      title: "工具",
      tone: "running",
      statusLabel: "RUNNING",
      details: "准备输出",
      titleMode: "status",
      showTerminalIcon: true,
    });
  });

  it("uses plain Chinese titles without terminal icons for file actions", () => {
    expect(buildMobileToolCardViewModel({
      id: "tool-read",
      input: JSON.stringify({ path: "/tmp/main.go" }),
      output: "package main",
      status: "success",
      toolName: "read_file",
    })).toMatchObject({
      title: "阅读文件 /tmp/main.go",
      titleMode: "plain",
      showTerminalIcon: false,
    });

    expect(buildMobileToolCardViewModel({
      id: "tool-write",
      output: JSON.stringify({ content_summary: { lines: 4 }, path: "/tmp/out.txt" }),
      status: "success",
      toolName: "write_file",
    })).toMatchObject({
      title: "创建文件 /tmp/out.txt +4",
      titleMode: "plain",
      showTerminalIcon: false,
    });

    expect(buildMobileToolCardViewModel({
      id: "tool-edit",
      input: JSON.stringify({
        diff_text: "@@ -1,2 +1,3 @@\n-old\n+new\n+extra\n same\n",
        path: "/tmp/out.txt",
      }),
      output: "applied 1 hunks to /tmp/out.txt",
      status: "success",
      toolName: "apply_diff",
    })).toMatchObject({
      title: "编辑文件 /tmp/out.txt +2 -1",
      titleMode: "plain",
      showTerminalIcon: false,
    });
  });

  it("uses plain Chinese titles for web, codex, skill, search, and screen actions", () => {
    expect(buildMobileToolCardViewModel({
      id: "tool-web",
      input: JSON.stringify({ query: "Ghost OS release notes" }),
      output: JSON.stringify([{ title: "Ghost OS" }]),
      status: "success",
      toolName: "web_search",
    })).toMatchObject({
      title: "网络搜索 Ghost OS release notes",
      titleMode: "plain",
      showTerminalIcon: false,
    });

    expect(buildMobileToolCardViewModel({
      id: "tool-codex",
      input: JSON.stringify({ op: "status", session_id: "cmd-123" }),
      status: "running",
      toolName: "codex_cli",
    })).toMatchObject({
      title: "调用codex status",
      titleMode: "plain",
      showTerminalIcon: false,
    });

    expect(buildMobileToolCardViewModel({
      id: "tool-sfind",
      output: JSON.stringify({ action: "load", items: [{ name: "release_flow" }] }),
      status: "success",
      toolName: "sfind",
    })).toMatchObject({
      title: "加载技能 release_flow",
      titleMode: "plain",
      showTerminalIcon: false,
    });

    expect(buildMobileToolCardViewModel({
      id: "tool-search",
      output: JSON.stringify({ query: "bridge runtime selector" }),
      status: "success",
      toolName: "search_files",
    })).toMatchObject({
      title: "搜索 bridge runtime selector（关键词）",
      titleMode: "plain",
      showTerminalIcon: false,
    });

    expect(buildMobileToolCardViewModel({
      id: "tool-screen",
      input: JSON.stringify({ action: "screenshot", display_id: 67 }),
      status: "running",
      toolName: "screen_control",
    })).toMatchObject({
      title: "截取屏幕",
      titleMode: "plain",
      showTerminalIcon: false,
    });
  });

  it("normalizes status labels by tone and keeps explicit errors in details", () => {
    expect(getMobileToolTone("failed")).toBe("error");
    expect(getMobileToolTone("pending")).toBe("running");
    expect(getMobileToolStatusLabel("success", "completed")).toBe("SUCCESS");
    expect(getMobileToolStatusLabel("success", "rate_limited")).toBe("RATE LIMITED");
    expect(buildMobileToolCardViewModel({
      error: "command failed",
      id: "tool-error",
      status: "error",
      toolName: "bash_exec",
    })).toMatchObject({
      details: "error: command failed",
      tone: "error",
    });
  });
});
