import { Check, Copy } from "lucide-react";
import { useState } from "react";

const COPY_RESET_DELAY_MS = 1600;

interface MessageCopyButtonProps {
  text: string;
  variant?: "default" | "code";
}

async function copyWithClipboard(text: string): Promise<boolean> {
  if (!navigator.clipboard?.writeText) {
    return false;
  }

  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch {
    return false;
  }
}

function copyWithTextArea(text: string): boolean {
  const textArea = document.createElement("textarea");
  textArea.value = text;
  textArea.style.position = "fixed";
  textArea.style.opacity = "0";
  document.body.appendChild(textArea);
  textArea.select();

  try {
    return document.execCommand("copy");
  } catch {
    return false;
  } finally {
    document.body.removeChild(textArea);
  }
}

export function MessageCopyButton({ text, variant = "default" }: MessageCopyButtonProps) {
  const [copied, setCopied] = useState(false);
  const isCodeVariant = variant === "code";
  const label = copied ? "已复制" : "复制";
  const Icon = copied ? Check : Copy;
  const className = ["copy-button", isCodeVariant ? "is-code-block" : "", copied ? "is-copied" : ""]
    .filter(Boolean)
    .join(" ");

  async function handleCopy(): Promise<void> {
    if (!text) {
      return;
    }

    const didCopy = (await copyWithClipboard(text)) || copyWithTextArea(text);
    if (!didCopy) {
      return;
    }

    setCopied(true);
    window.setTimeout(() => setCopied(false), COPY_RESET_DELAY_MS);
  }

  return (
    <button type="button" className={className} onClick={handleCopy} aria-label={label} title={label}>
      <Icon className="copy-icon" aria-hidden="true" strokeWidth={1.8} />
      {!isCodeVariant ? <span className="copy-label">{label}</span> : null}
    </button>
  );
}
