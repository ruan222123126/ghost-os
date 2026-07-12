import { useEffect, useLayoutEffect, useRef, useState } from "react";
import type { CSSProperties, FormEvent, KeyboardEvent, RefObject } from "react";
import type { AgentModeSelection, ChatSelectedSkill, SkillPayload } from "../../mobileTypes";
import { COMPOSER_MENU_OPTIONS } from "./data";
import { UiIcon } from "./icons";
import "./ChatComposer.css";

interface ChatComposerProps {
  agentMode: AgentModeSelection;
  canEnableCodexMode?: boolean;
  canSubmit?: boolean;
  value: string;
  canStop?: boolean;
  disabled?: boolean;
  loading: boolean;
  selectedSkill?: ChatSelectedSkill | null;
  skills?: SkillPayload[];
  onChangeAgentMode?: (mode: AgentModeSelection) => void;
  onClearSelectedSkill?: () => void;
  onRefreshSkills?: () => Promise<boolean> | Promise<void> | boolean | void;
  onSelectSkill?: (skill: SkillPayload) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => Promise<void>;
  onStop?: () => Promise<void>;
  onChange: (value: string) => void;
}

const COMPOSER_SKILL_EMPTY_LABEL = "暂无";
const COMPOSER_CODEX_DISABLED_LABEL = "连接电脑后可用";
const COMPOSER_CODEX_MODE_LABEL = "codex模式";
const COMPOSER_TEXTAREA_COLLAPSED_HEIGHT_PX = 52;
const COMPOSER_TEXTAREA_MAX_HEIGHT_PX = 200;
const TEXTAREA_SCROLL_HEIGHT_EPSILON_PX = 1;

type ComposerDockStyle = CSSProperties & {
  "--composer-keyboard-inset": string;
};

type ComposerMenuView = "attachment" | "features" | "skills" | null;

export function ChatComposer(props: ChatComposerProps) {
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const lineMeasureRef = useRef<HTMLTextAreaElement>(null);
  const heightMeasureRef = useRef<HTMLTextAreaElement>(null);
  const composerRef = useRef<HTMLFormElement>(null);
  const [focused, setFocused] = useState(false);
  const [menuView, setMenuView] = useState<ComposerMenuView>(null);
  const hasSelectedSkill = Boolean(props.selectedSkill);
  const hasValue = props.value.trim().length > 0;
  const hasPendingSubmission = hasValue || hasSelectedSkill;
  const canSubmit = props.canSubmit ?? hasPendingSubmission;
  const showStop = props.loading && props.onStop !== undefined;
  const showAction = hasPendingSubmission || showStop;
  const wrapsPastSingleLine = useSingleLineOverflow(lineMeasureRef, props.value);
  const isMultiLine = props.value.includes("\n") || wrapsPastSingleLine;
  const keyboardInset = useKeyboardInset(focused);
  const attachmentMenuOpen = menuView === "attachment";
  const featureMenuOpen = menuView === "features";
  const skillMenuOpen = menuView === "skills";
  const agentModeActive = props.agentMode !== null;
  const codexModeToggleEnabled = agentModeActive || props.canEnableCodexMode === true;
  const codexModeDescription = props.agentMode === "normal"
    ? "normal 已开启"
    : props.agentMode === "plan"
    ? "plan 已开启"
    : codexModeToggleEnabled
    ? "ghost 模式"
    : COMPOSER_CODEX_DISABLED_LABEL;
  const enabledSkills = sortEnabledSkills(props.skills ?? []);
  const dockStyle: ComposerDockStyle = {
    "--composer-keyboard-inset": `${keyboardInset}px`,
  };

  useAutosizeTextarea(textareaRef, heightMeasureRef, props.value, isMultiLine);
  useCloseComposerMenu(composerRef, menuView !== null, () => setMenuView(null));

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    setMenuView(null);
    const submitPromise = props.onSubmit(event);
    focusComposerInput(textareaRef.current);
    await submitPromise;
  }

  function handleStop(): void {
    if (!props.canStop || props.onStop === undefined) {
      return;
    }

    setMenuView(null);
    void props.onStop();
  }

  function handleKeyDown(event: KeyboardEvent<HTMLTextAreaElement>): void {
    if (event.key !== "Backspace" || props.value !== "" || !props.selectedSkill || !props.onClearSelectedSkill) {
      return;
    }

    event.preventDefault();
    props.onClearSelectedSkill();
  }

  function handleMenuOption(option: (typeof COMPOSER_MENU_OPTIONS)[number]): void {
    if (option.unavailable) {
      return;
    }

    if (option.id === "skills") {
      setMenuView("skills");
      void props.onRefreshSkills?.();
      textareaRef.current?.focus();
      return;
    }

    if (option.id === "features") {
      setMenuView("features");
    } else {
      setMenuView(null);
    }

    textareaRef.current?.focus();
  }

  function handleSelectSkill(skill: SkillPayload): void {
    props.onSelectSkill?.(skill);
    setMenuView(null);
    textareaRef.current?.focus();
  }

  function handleChangeAgentMode(mode: AgentModeSelection): void {
    if (!props.onChangeAgentMode) {
      textareaRef.current?.focus();
      return;
    }

    if (mode !== null && mode !== props.agentMode && props.canEnableCodexMode !== true) {
      return;
    }

    props.onChangeAgentMode(mode);
    setMenuView(null);
    textareaRef.current?.focus();
  }

  return (
    <form ref={composerRef} className="composer-dock" style={dockStyle} onSubmit={(event) => void handleSubmit(event)}>
      {attachmentMenuOpen ? (
        <div className="composer-attachment-menu" role="menu" aria-label="添加内容">
          {COMPOSER_MENU_OPTIONS.map((option) => (
            <button
              key={option.label}
              className="composer-menu-option"
              type="button"
              role="menuitem"
              disabled={option.unavailable}
              title={option.unavailable ? "暂未接入" : option.label}
              onClick={() => handleMenuOption(option)}
            >
              <span className="composer-menu-icon">
                <UiIcon name={option.icon} />
              </span>
              <span>{option.label}</span>
            </button>
          ))}
        </div>
      ) : null}

      {featureMenuOpen ? (
        <div className="composer-feature-menu" role="menu" aria-label="功能">
          <div className="composer-skill-menu-head">
            <span>功能</span>
          </div>
          <div className="composer-feature-mode-switch" role="group" aria-label="Codex 模式">
            <span className="composer-feature-mode-label">{COMPOSER_CODEX_MODE_LABEL}</span>
            <button
              className={props.agentMode === "normal" ? "is-active" : ""}
              type="button"
              aria-pressed={props.agentMode === "normal"}
              disabled={!codexModeToggleEnabled && props.agentMode !== "normal"}
              onClick={() => handleChangeAgentMode(props.agentMode === "normal" ? null : "normal")}
            >
              normal
            </button>
            <button
              className={props.agentMode === "plan" ? "is-active" : ""}
              type="button"
              aria-pressed={props.agentMode === "plan"}
              disabled={!codexModeToggleEnabled && props.agentMode !== "plan"}
              onClick={() => handleChangeAgentMode(props.agentMode === "plan" ? null : "plan")}
            >
              plan
            </button>
          </div>
          <div className="composer-feature-status" role="status">{codexModeDescription}</div>
        </div>
      ) : null}

      {skillMenuOpen ? (
        <div className="composer-skill-menu" role="menu" aria-label="技能">
          <div className="composer-skill-menu-head">
            <span>技能</span>
            {props.onRefreshSkills ? (
              <button type="button" className="composer-skill-refresh" onClick={() => void props.onRefreshSkills?.()}>
                刷新
              </button>
            ) : null}
          </div>

          {enabledSkills.length === 0 ? (
            <div className="composer-skill-status" role="status">{COMPOSER_SKILL_EMPTY_LABEL}</div>
          ) : (
            <div className="composer-skill-list">
              {enabledSkills.map((skill) => (
                <button
                  key={skill.id}
                  type="button"
                  className="composer-skill-item"
                  role="menuitem"
                  onClick={() => handleSelectSkill(skill)}
                >
                  <span className="composer-skill-item-name">{skill.name}</span>
                </button>
              ))}
            </div>
          )}
        </div>
      ) : null}

      <div
        className={[
          "composer-shell",
          focused ? "is-focused" : "",
          isMultiLine ? "is-multiline" : "",
          hasSelectedSkill ? "has-selected-skill" : "",
        ].filter(Boolean).join(" ")}
      >
        <div className="composer-textarea-wrap">
          <button
            className={`icon-button icon-button-composer composer-menu-trigger ${menuView !== null ? "is-open" : ""}`}
            type="button"
            aria-label="添加内容"
            aria-expanded={menuView !== null}
            aria-haspopup="menu"
            onClick={() => setMenuView((current) => (current === "attachment" ? null : "attachment"))}
          >
            <UiIcon name="plus" />
          </button>
          {props.selectedSkill ? (
            <div className="composer-selected-skill-row">
              <button
                type="button"
                className="composer-selected-skill"
                aria-label={`取消已选技能 ${props.selectedSkill.name}`}
                onClick={props.onClearSelectedSkill}
              >
                {props.selectedSkill.name}
              </button>
            </div>
          ) : null}
          <textarea
            ref={textareaRef}
            value={props.value}
            rows={1}
            placeholder={props.selectedSkill ? "" : "问问 Ghost-OS"}
            className="composer-input"
            onChange={(event) => props.onChange(event.currentTarget.value)}
            onBlur={() => setFocused(false)}
            onFocus={() => setFocused(true)}
            onInput={(event) =>
              syncTextareaHeight(textareaRef.current, heightMeasureRef.current, isMultiLine, event.currentTarget.value)
            }
            onKeyDown={handleKeyDown}
          />
          <textarea
            ref={lineMeasureRef}
            aria-hidden="true"
            className="composer-input composer-single-line-measure"
            readOnly
            rows={1}
            tabIndex={-1}
            value={props.value}
          />
          <textarea
            ref={heightMeasureRef}
            aria-hidden="true"
            className="composer-input composer-height-measure"
            readOnly
            rows={1}
            tabIndex={-1}
            value={props.value}
          />
          <div className="composer-actions">
            <span className={`send-button-slot ${showAction ? "is-visible" : ""}`} aria-hidden={!showAction}>
              <button
                className={`send-button ${showStop ? "is-stop" : ""}`}
                type={showStop ? "button" : "submit"}
                disabled={showStop ? !props.canStop : props.disabled || !canSubmit}
                onClick={showStop ? handleStop : undefined}
                aria-busy={props.loading}
                aria-label={showStop ? (props.canStop ? "停止生成" : "停止中") : "发送任务"}
                tabIndex={showAction ? 0 : -1}
              >
                <UiIcon name={showStop ? "stop" : "arrow-up"} />
              </button>
            </span>
          </div>
        </div>
        <div className="composer-bottom-spacer" aria-hidden="true" />
      </div>
    </form>
  );
}

function useKeyboardInset(active: boolean) {
  const [inset, setInset] = useState(0);

  useLayoutEffect(() => {
    if (!active) {
      setInset(0);
      return;
    }

    const visualViewport = window.visualViewport;
    if (!visualViewport) {
      return;
    }
    const viewport: VisualViewport = visualViewport;

    let animationFrame: number | undefined;

    function updateInset(): void {
      if (animationFrame !== undefined) {
        window.cancelAnimationFrame(animationFrame);
      }

      animationFrame = window.requestAnimationFrame(() => {
        animationFrame = undefined;
        const nextInset = Math.max(0, Math.round(window.innerHeight - viewport.height - viewport.offsetTop));
        setInset((current) => (current === nextInset ? current : nextInset));
      });
    }

    updateInset();
    viewport.addEventListener("resize", updateInset);
    viewport.addEventListener("scroll", updateInset);
    window.addEventListener("orientationchange", updateInset);

    return () => {
      if (animationFrame !== undefined) {
        window.cancelAnimationFrame(animationFrame);
      }
      viewport.removeEventListener("resize", updateInset);
      viewport.removeEventListener("scroll", updateInset);
      window.removeEventListener("orientationchange", updateInset);
    };
  }, [active]);

  return inset;
}

function useCloseComposerMenu(
  composerRef: RefObject<HTMLFormElement | null>,
  enabled: boolean,
  onClose: () => void,
) {
  useEffect(() => {
    if (!enabled) {
      return;
    }

    function handlePointerDown(event: PointerEvent): void {
      if (!(event.target instanceof Node) || composerRef.current?.contains(event.target)) {
        return;
      }

      onClose();
    }

    document.addEventListener("pointerdown", handlePointerDown);
    return () => document.removeEventListener("pointerdown", handlePointerDown);
  }, [composerRef, enabled, onClose]);
}

function useSingleLineOverflow(textareaRef: RefObject<HTMLTextAreaElement | null>, value: string) {
  const [overflows, setOverflows] = useState(false);

  useLayoutEffect(() => {
    const textarea = textareaRef.current;
    if (!textarea) {
      return;
    }

    function updateOverflow(): void {
      if (!textarea) {
        return;
      }

      const nextOverflows =
        textarea.scrollHeight > COMPOSER_TEXTAREA_COLLAPSED_HEIGHT_PX + TEXTAREA_SCROLL_HEIGHT_EPSILON_PX;
      setOverflows((current) => (current === nextOverflows ? current : nextOverflows));
    }

    updateOverflow();

    if (typeof ResizeObserver === "undefined") {
      return;
    }

    const resizeObserver = new ResizeObserver(updateOverflow);
    resizeObserver.observe(textarea);
    return () => resizeObserver.disconnect();
  }, [textareaRef, value]);

  return overflows;
}

function useAutosizeTextarea(
  textareaRef: RefObject<HTMLTextAreaElement | null>,
  measureRef: RefObject<HTMLTextAreaElement | null>,
  value: string,
  isMultiLine: boolean,
) {
  useLayoutEffect(() => {
    syncTextareaHeight(textareaRef.current, measureRef.current, isMultiLine, value);
  }, [isMultiLine, measureRef, textareaRef, value]);
}

function syncTextareaHeight(
  textarea: HTMLTextAreaElement | null,
  measureTextarea: HTMLTextAreaElement | null,
  isMultiLine: boolean,
  value?: string,
) {
  if (!textarea) {
    return;
  }

  if (!isMultiLine) {
    textarea.style.height = `${COMPOSER_TEXTAREA_COLLAPSED_HEIGHT_PX}px`;
    return;
  }

  if (!measureTextarea) {
    return;
  }

  if (value !== undefined && measureTextarea.value !== value) {
    measureTextarea.value = value;
  }

  textarea.style.height = "auto";
  const nextHeight = Math.min(measureTextarea.scrollHeight, COMPOSER_TEXTAREA_MAX_HEIGHT_PX);
  textarea.style.height = `${Math.max(COMPOSER_TEXTAREA_COLLAPSED_HEIGHT_PX, nextHeight)}px`;
}

function sortEnabledSkills(skills: SkillPayload[]): SkillPayload[] {
  return skills
    .filter((skill) => skill.enabled)
    .sort((left, right) => {
      if (left.name !== right.name) {
        return left.name.localeCompare(right.name, "zh-Hans");
      }
      return left.source.localeCompare(right.source, "zh-Hans");
    });
}

function focusComposerInput(textarea: HTMLTextAreaElement | null): void {
  textarea?.focus({ preventScroll: true });
}
