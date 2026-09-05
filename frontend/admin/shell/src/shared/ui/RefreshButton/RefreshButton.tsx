// frontend/admin/shell/src/shared/ui/RefreshButton/RefreshButton.tsx

import "./RefreshButton.css";

type RefreshButtonProps = {
  onClick?: () => void | Promise<void>;
  loading?: boolean;
  disabled?: boolean;
  title?: string;
  ariaLabel?: string;
  className?: string;
  size?: number;
};

export default function RefreshButton({
  onClick,
  loading = false,
  disabled = false,
  title = "リフレッシュ",
  ariaLabel = "リフレッシュ",
  className = "",
  size = 18,
}: RefreshButtonProps) {
  const buttonClassName = ["ui-refresh-button", className].filter(Boolean).join(" ");
  const iconClassName = [
    "ui-refresh-button__icon",
    loading && "ui-refresh-button__icon--loading",
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <button
      type="button"
      className={buttonClassName}
      aria-label={ariaLabel}
      aria-busy={loading}
      title={title}
      onClick={() => void onClick?.()}
      disabled={disabled || loading}
    >
      <svg
        width={size}
        height={size}
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
        className={iconClassName}
        aria-hidden="true"
      >
        <path d="M21 12a9 9 0 1 1-2.64-6.36L21 8" />
        <path d="M21 3v5h-5" />
      </svg>
    </button>
  );
}