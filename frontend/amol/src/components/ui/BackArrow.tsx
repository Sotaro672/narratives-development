import type { ButtonHTMLAttributes } from "react";

type BackArrowProps = ButtonHTMLAttributes<HTMLButtonElement>;

export default function BackArrow({
  type = "button",
  ariaLabel = "戻る",
  ...props
}: BackArrowProps & { ariaLabel?: string }) {
  return (
    <button
      type={type}
      aria-label={ariaLabel}
      {...props}
      style={{
        border: "none",
        background: "transparent",
        padding: 0,
        margin: 0,
        display: "inline-flex",
        alignItems: "center",
        justifyContent: "center",
        cursor: "pointer",
        color: "var(--color-text)",
      }}
    >
      <svg
        width="20"
        height="20"
        viewBox="0 0 24 24"
        fill="none"
        aria-hidden="true"
      >
        <path
          d="M15 6L9 12L15 18"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      </svg>
    </button>
  );
}