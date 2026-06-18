import { useEffect, useLayoutEffect, useRef, useState } from "react";
import type { FormEvent, RefObject } from "react";
import { COMPOSER_MENU_OPTIONS } from "./data";
import { IconButton, UiIcon } from "./icons";
import "./ChatComposer.css";

interface ChatComposerProps {
  value: string;
  disabled: boolean;
  loading: boolean;
  onSubmit: (event: FormEvent<HTMLFormElement>) => Promise<void>;
  onChange: (value: string) => void;
  onOpenSettings: () => void;
}

const COMPOSER_TEXTAREA_COLLAPSED_HEIGHT_PX = 52;
const COMPOSER_TEXTAREA_MAX_HEIGHT_PX = 200;
const MULTILINE_TEXT_THRESHOLD = 30;

export function ChatComposer(props: ChatComposerProps) {
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const composerRef = useRef<HTMLFormElement>(null);
  const [focused, setFocused] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);
  const hasValue = props.value.trim().length > 0;
  const isMultiLine = props.value.includes("\n") || props.value.length > MULTILINE_TEXT_THRESHOLD;

  useAutosizeTextarea(textareaRef, props.value, isMultiLine);
  useCloseComposerMenu(composerRef, menuOpen, setMenuOpen);

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    setMenuOpen(false);
    await props.onSubmit(event);
  }

  function handleMenuOption(label: string, unavailable: boolean): void {
    if (unavailable) {
      return;
    }

    setMenuOpen(false);
    if (label === "技能") {
      props.onOpenSettings();
    }
  }

  return (
    <form ref={composerRef} className="composer-dock" onSubmit={(event) => void handleSubmit(event)}>
      {menuOpen ? (
        <div className="composer-attachment-menu" role="menu" aria-label="添加内容">
          {COMPOSER_MENU_OPTIONS.map((option) => (
            <button
              key={option.label}
              className="composer-menu-option"
              type="button"
              role="menuitem"
              disabled={option.unavailable}
              title={option.unavailable ? "暂未接入" : option.label}
              onClick={() => handleMenuOption(option.label, option.unavailable)}
            >
              <span className="composer-menu-icon">
                <UiIcon name={option.icon} />
              </span>
              <span>{option.label}</span>
            </button>
          ))}
        </div>
      ) : null}

      <div className={`composer-shell ${focused ? "is-focused" : ""} ${isMultiLine ? "is-multiline" : ""}`}>
        <div className="composer-textarea-wrap">
          <button
            className={`icon-button icon-button-composer composer-menu-trigger ${menuOpen ? "is-open" : ""}`}
            type="button"
            aria-label="添加内容"
            aria-expanded={menuOpen}
            aria-haspopup="menu"
            onClick={() => setMenuOpen((current) => !current)}
          >
            <UiIcon name="plus" />
          </button>
          <textarea
            ref={textareaRef}
            value={props.value}
            rows={1}
            placeholder="问问 Ghost-OS"
            className="composer-input"
            onChange={(event) => props.onChange(event.currentTarget.value)}
            onBlur={() => setFocused(false)}
            onFocus={() => setFocused(true)}
            onInput={() => syncTextareaHeight(textareaRef.current, isMultiLine)}
          />
          <div className="composer-actions">
            <IconButton label="语音输入" icon="mic" variant="composer" />
            <span className={`send-button-slot ${hasValue ? "is-visible" : ""}`} aria-hidden={!hasValue}>
              <button
                className="send-button"
                type="submit"
                disabled={props.disabled}
                aria-busy={props.loading}
                aria-label="发送任务"
                tabIndex={hasValue ? 0 : -1}
              >
                <UiIcon name="arrow-up" />
              </button>
            </span>
          </div>
        </div>
        <div className="composer-bottom-spacer" aria-hidden="true" />
      </div>
    </form>
  );
}

function useCloseComposerMenu(
  composerRef: RefObject<HTMLFormElement | null>,
  menuOpen: boolean,
  setMenuOpen: (open: boolean) => void,
) {
  useEffect(() => {
    if (!menuOpen) {
      return;
    }

    function handlePointerDown(event: PointerEvent): void {
      if (!(event.target instanceof Node) || composerRef.current?.contains(event.target)) {
        return;
      }

      setMenuOpen(false);
    }

    document.addEventListener("pointerdown", handlePointerDown);
    return () => document.removeEventListener("pointerdown", handlePointerDown);
  }, [composerRef, menuOpen, setMenuOpen]);
}

function useAutosizeTextarea(
  textareaRef: RefObject<HTMLTextAreaElement | null>,
  value: string,
  isMultiLine: boolean,
) {
  useLayoutEffect(() => {
    syncTextareaHeight(textareaRef.current, isMultiLine);
  }, [isMultiLine, textareaRef, value]);
}

function syncTextareaHeight(textarea: HTMLTextAreaElement | null, isMultiLine: boolean) {
  if (!textarea) {
    return;
  }

  textarea.style.transition = "none";
  textarea.style.height = "auto";
  const scrollHeight = textarea.scrollHeight;
  void textarea.offsetHeight;
  textarea.style.transition = "";
  textarea.style.height = isMultiLine
    ? `${Math.min(scrollHeight, COMPOSER_TEXTAREA_MAX_HEIGHT_PX)}px`
    : `${COMPOSER_TEXTAREA_COLLAPSED_HEIGHT_PX}px`;
}
