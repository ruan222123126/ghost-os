import { Check, Copy } from "lucide-react";
import { useState } from "react";
import { COPY_FEEDBACK_RESET_DELAY_MS, copyTextToClipboard } from "../../../../shared/browserClipboard";

interface MessageCopyButtonProps {
  text: string;
  variant?: "default" | "code";
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

    const didCopy = await copyTextToClipboard(text);
    if (!didCopy) {
      return;
    }

    setCopied(true);
    window.setTimeout(() => setCopied(false), COPY_FEEDBACK_RESET_DELAY_MS);
  }

  return (
    <button type="button" className={className} onClick={handleCopy} aria-label={label} title={label}>
      <Icon className="copy-icon" aria-hidden="true" strokeWidth={1.8} />
      {!isCodeVariant ? <span className="copy-label">{label}</span> : null}
    </button>
  );
}
