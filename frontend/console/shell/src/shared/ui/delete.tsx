// frontend/console/shell/src/shared/ui/delete.tsx
import type * as React from "react";
import "./delete.css";

export type DeleteButtonSize = "sm" | "md";

type DeleteButtonProps = {
  size?: DeleteButtonSize;
  className?: string;
  disabled?: boolean;
  ariaLabel?: string;
  title?: string;
  onClick: (event: React.MouseEvent<HTMLButtonElement>) => void;
};

function sizeClassName(size: DeleteButtonSize): string {
  switch (size) {
    case "sm":
      return "ui-delete-btn--sm";
    case "md":
    default:
      return "ui-delete-btn--md";
  }
}

export default function DeleteButton({
  size = "md",
  className = "",
  disabled = false,
  ariaLabel = "delete",
  title = "削除",
  onClick,
}: DeleteButtonProps) {
  return (
    <button
      type="button"
      className={["ui-delete-btn", sizeClassName(size), className]
        .filter(Boolean)
        .join(" ")}
      onClick={onClick}
      aria-label={ariaLabel}
      title={title}
      disabled={disabled}
    >
      <span className="ui-delete-btn__x">×</span>
    </button>
  );
}