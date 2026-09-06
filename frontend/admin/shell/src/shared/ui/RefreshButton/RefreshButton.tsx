// frontend/admin/shell/src/shared/ui/RefreshButton/RefreshButton.tsx

import type { ButtonSize } from "../Button/Button";
import Button from "../Button/Button";

import "./RefreshButton.css";

type RefreshButtonProps = {
  onClick?: () => void | Promise<void>;
  loading?: boolean;
  disabled?: boolean;
  title?: string;
  ariaLabel?: string;
  className?: string;
  iconSize?: number;
  buttonSize?: ButtonSize;
};

export default function RefreshButton({
  onClick,
  loading = false,
  disabled = false,
  title = "リフレッシュ",
  ariaLabel = "リフレッシュ",
  className = "",
  iconSize = 16,
  buttonSize = "sm",
}: RefreshButtonProps) {
  return (
    <Button
      variant="secondary"
      size={buttonSize}
      iconOnly
      loading={loading}
      disabled={disabled}
      className={className}
      aria-label={ariaLabel}
      title={title}
      onClick={() => void onClick?.()}
    >
      <svg
        width={iconSize}
        height={iconSize}
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
        className={loading ? "ui-refresh-button__icon--loading" : undefined}
        aria-hidden="true"
      >
        <path d="M21 12a9 9 0 1 1-2.64-6.36L21 8" />
        <path d="M21 3v5h-5" />
      </svg>
    </Button>
  );
}