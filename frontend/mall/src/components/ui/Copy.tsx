// frontend/mall/src/components/ui/Copy.tsx

import { useEffect, useState } from "react";

import IconButton from "./IconButton";

import "./copy.css";

type CopyProps = {
  type?: "button" | "submit" | "reset";
  onClick?: () => void | Promise<void>;
  ariaLabel?: string;
  title?: string;
  disabled?: boolean;
  copiedLabel?: string;
};

export default function Copy({
  type = "button",
  onClick,
  ariaLabel = "コピー",
  title = "コピー",
  disabled = false,
  copiedLabel = "コピーしました",
}: CopyProps) {
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

  const handleClick = async (): Promise<void> => {
    if (disabled || !onClick) return;

    try {
      await onClick();
      setCopied(true);
    } catch {
      setCopied(false);
    }
  };

  return (
    <span className="copy-button__control">
      <IconButton
        type={type}
        variant="secondary"
        size="sm"
        aria-label={ariaLabel}
        title={title}
        disabled={disabled}
        onClick={() => {
          void handleClick();
        }}
      >
        <svg
          className="copy-button__icon"
          viewBox="0 0 24 24"
          aria-hidden="true"
          focusable="false"
        >
          <path
            d="M9 9a2 2 0 0 1 2-2h7a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2h-7a2 2 0 0 1-2-2V9zm-4 6V6a2 2 0 0 1 2-2h7"
            fill="none"
            stroke="currentColor"
            strokeWidth="1.8"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
        </svg>
      </IconButton>

      <span
        className={[
          "copy-button__feedback",
          copied ? "copy-button__feedback--visible" : "",
        ]
          .filter(Boolean)
          .join(" ")}
        role="status"
        aria-live="polite"
      >
        {copied ? copiedLabel : ""}
      </span>
    </span>
  );
}