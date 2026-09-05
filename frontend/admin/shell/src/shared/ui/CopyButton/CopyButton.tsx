// frontend/admin/shell/src/shared/ui/CopyButton/CopyButton.tsx

import { useEffect, useState } from "react";

import "./CopyButton.css";

type CopyButtonProps = {
  value: string;
  disabled?: boolean;
  title?: string;
  ariaLabel?: string;
  copiedLabel?: string;
};

export default function CopyButton({
  value,
  disabled = false,
  title = "コピー",
  ariaLabel = "コピー",
  copiedLabel = "コピーしました",
}: CopyButtonProps) {
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!copied) return;

    const timeoutId = window.setTimeout(() => {
      setCopied(false);
    }, 1800);

    return () => {
      window.clearTimeout(timeoutId);
    };
  }, [copied]);

  const handleCopy = async (): Promise<void> => {
    if (disabled || !value) return;

    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
    } catch {
      setCopied(false);
    }
  };

  return (
    <span className="ui-copy-control">
      <button
        type="button"
        className="ui-copy-button"
        onClick={() => void handleCopy()}
        disabled={disabled || !value}
        aria-label={ariaLabel}
        title={title}
      >
        <svg
          width="18"
          height="18"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
          aria-hidden="true"
        >
          <rect width="14" height="14" x="8" y="8" rx="2" ry="2" />
          <path d="M16 8V6a2 2 0 0 0-2-2H6a2 2 0 0 0-2 2v8a2 2 0 0 0 2 2h2" />
        </svg>
      </button>

      <span
        className={`ui-copy-control__feedback ${
          copied ? "ui-copy-control__feedback--visible" : ""
        }`}
        role="status"
        aria-live="polite"
      >
        {copied ? copiedLabel : ""}
      </span>
    </span>
  );
}