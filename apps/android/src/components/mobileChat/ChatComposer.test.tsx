// @vitest-environment jsdom
import { useState } from "react";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { ChatSelectedSkill, SkillPayload } from "../../mobileTypes";
import { ChatComposer } from "./ChatComposer";

let measuredScrollHeight = 52;
const originalInnerHeightDescriptor = Object.getOwnPropertyDescriptor(window, "innerHeight");
const originalVisualViewportDescriptor = Object.getOwnPropertyDescriptor(window, "visualViewport");

interface MockVisualViewport extends EventTarget {
  height: number;
  offsetTop: number;
}

describe("ChatComposer", () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
    measuredScrollHeight = 52;
    restoreWindowViewportProperties();
  });

  it("switches to multiline when single-line layout overflows", () => {
    mockTextareaScrollHeight();
    measuredScrollHeight = 76;

    renderComposerHarness({ initialValue: "这是十一位中文输入" });

    expect(composerShell().classList.contains("is-multiline")).toBe(true);
  });

  it("keeps long text single-line when measured layout still fits", () => {
    mockTextareaScrollHeight();
    measuredScrollHeight = 52;

    renderComposerHarness({ initialValue: "abcdefghijklmnopqrstuvwxyzabcdefghi" });

    expect(composerShell().classList.contains("is-multiline")).toBe(false);
  });

  it("shows a stop button while loading without input text", () => {
    const onStop = vi.fn(async () => undefined);

    renderComposerHarness({ canStop: true, disabled: true, loading: true, onStop });

    fireEvent.click(screen.getByRole("button", { name: "停止生成" }));

    expect(onStop).toHaveBeenCalledTimes(1);
  });

  it("keeps the stop button visible but disabled while stopping", () => {
    renderComposerHarness({ canStop: false, disabled: true, loading: true, onStop: vi.fn(async () => undefined) });

    expect(screen.getByRole<HTMLButtonElement>("button", { name: "停止中" }).disabled).toBe(true);
  });

  it("raises the dock when the focused visual viewport is covered by the keyboard", async () => {
    const viewport = mockVisualViewport({ height: 500, offsetTop: 0 });
    mockAnimationFrame();
    Object.defineProperty(window, "innerHeight", { configurable: true, value: 800 });

    renderComposerHarness();

    fireEvent.focus(screen.getByRole("textbox"));

    await waitFor(() => {
      expect(composerDock().style.getPropertyValue("--composer-keyboard-inset")).toBe("300px");
    });

    viewport.height = 620;
    viewport.dispatchEvent(new Event("resize"));

    await waitFor(() => {
      expect(composerDock().style.getPropertyValue("--composer-keyboard-inset")).toBe("180px");
    });
  });

  it("opens the skill menu from the plus menu and selects a skill", () => {
    const onRefreshSkills = vi.fn(async () => undefined);

    renderComposerHarness({
      onRefreshSkills,
      onSelectSkill: true,
      skills: [buildSkill({ id: "skill_release", name: "release_flow" })],
    });

    fireEvent.click(screen.getByRole("button", { name: "添加内容" }));
    fireEvent.click(screen.getByRole("menuitem", { name: "技能" }));

    expect(onRefreshSkills).toHaveBeenCalledTimes(1);
    expect(screen.getByRole("menuitem", { name: "release_flow" })).toBeTruthy();

    fireEvent.click(screen.getByRole("menuitem", { name: "release_flow" }));

    expect(screen.getByRole("button", { name: "取消已选技能 release_flow" })).toBeTruthy();
    expect(screen.getByRole<HTMLTextAreaElement>("textbox").getAttribute("placeholder")).toBe("");
  });

  it("shows 暂无 when there is no skill list", () => {
    renderComposerHarness();

    fireEvent.click(screen.getByRole("button", { name: "添加内容" }));
    fireEvent.click(screen.getByRole("menuitem", { name: "技能" }));

    expect(screen.getByText("暂无")).toBeTruthy();
  });

  it("clears the selected skill when backspace is pressed on an empty draft", () => {
    renderComposerHarness({
      initialSelectedSkill: {
        id: "skill_release",
        name: "release_flow",
      },
      onSelectSkill: true,
    });

    const textarea = screen.getByRole("textbox");
    fireEvent.keyDown(textarea, { key: "Backspace" });

    expect(screen.queryByRole("button", { name: "取消已选技能 release_flow" })).toBeNull();
  });
});

function mockTextareaScrollHeight(): void {
  vi.spyOn(HTMLTextAreaElement.prototype, "scrollHeight", "get").mockImplementation(function scrollHeight(
    this: HTMLTextAreaElement,
  ) {
    return this.classList.contains("composer-single-line-measure") ? measuredScrollHeight : 52;
  });
}

function renderComposerHarness(options: {
  canStop?: boolean;
  disabled?: boolean;
  initialSelectedSkill?: ChatSelectedSkill | null;
  initialValue?: string;
  loading?: boolean;
  onRefreshSkills?: () => Promise<void> | void;
  onSelectSkill?: boolean;
  onStop?: () => Promise<void>;
  skills?: SkillPayload[];
} = {}) {
  const {
    canStop = false,
    disabled = false,
    initialSelectedSkill = null,
    initialValue = "",
    loading = false,
    onRefreshSkills,
    onSelectSkill = false,
    onStop,
    skills,
  } = options;

  function Harness() {
    const [value, setValue] = useState(initialValue);
    const [selectedSkill, setSelectedSkill] = useState<ChatSelectedSkill | null>(initialSelectedSkill);

    return (
      <ChatComposer
        canStop={canStop}
        disabled={disabled}
        loading={loading}
        selectedSkill={selectedSkill}
        skills={skills}
        onChange={setValue}
        onClearSelectedSkill={() => setSelectedSkill(null)}
        onRefreshSkills={onRefreshSkills}
        onSelectSkill={onSelectSkill ? (skill) => setSelectedSkill({ id: skill.id, name: skill.name }) : undefined}
        onStop={onStop}
        onSubmit={vi.fn(async (event) => {
          event.preventDefault();
        })}
        value={value}
      />
    );
  }

  return render(<Harness />);
}

function composerShell(): HTMLElement {
  const shell = screen.getByRole("textbox").closest(".composer-shell");
  if (!(shell instanceof HTMLElement)) {
    throw new Error("composer shell not found");
  }
  return shell;
}

function composerDock(): HTMLElement {
  const dock = screen.getByRole("textbox").closest(".composer-dock");
  if (!(dock instanceof HTMLElement)) {
    throw new Error("composer dock not found");
  }
  return dock;
}

function mockVisualViewport(input: { height: number; offsetTop: number }) {
  const viewport = new EventTarget() as MockVisualViewport;
  viewport.height = input.height;
  viewport.offsetTop = input.offsetTop;
  Object.defineProperty(window, "visualViewport", { configurable: true, value: viewport });
  return viewport;
}

function mockAnimationFrame(): void {
  vi.spyOn(window, "requestAnimationFrame").mockImplementation((callback) => {
    callback(0);
    return 1;
  });
  vi.spyOn(window, "cancelAnimationFrame").mockImplementation(() => undefined);
}

function restoreWindowViewportProperties(): void {
  if (originalInnerHeightDescriptor) {
    Object.defineProperty(window, "innerHeight", originalInnerHeightDescriptor);
  }
  if (originalVisualViewportDescriptor) {
    Object.defineProperty(window, "visualViewport", originalVisualViewportDescriptor);
  } else {
    Reflect.deleteProperty(window, "visualViewport");
  }
}

function buildSkill(overrides: Partial<SkillPayload> = {}): SkillPayload {
  return {
    description: "Default skill",
    enabled: true,
    id: "skill_default",
    name: "default_skill",
    path: "/tmp/skill",
    source: "repo",
    ...overrides,
  };
}
