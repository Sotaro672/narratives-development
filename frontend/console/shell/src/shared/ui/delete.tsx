// frontend/console/shell/src/shared/ui/delete.tsx

import type * as React from "react";
import { X } from "lucide-react";

import { Button } from "./button";

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
    <Button
      type="button"
      variant="destructive"
      size="icon"
      className={[sizeClassName(size), className]
        .filter(Boolean)
        .join(" ")}
      onClick={onClick}
      aria-label={ariaLabel}
      title={title}
      disabled={disabled}
    >
      <X
        className="ui-delete-btn__x"
        aria-hidden="true"
      />
    </Button>
  );
}