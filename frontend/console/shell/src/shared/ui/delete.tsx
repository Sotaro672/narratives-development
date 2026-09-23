// frontend/console/shell/src/shared/ui/delete.tsx

import type * as React from "react";
import { Trash2 } from "lucide-react";

import { Button } from "./button";

import "./delete.css";

export type DeleteButtonSize = "sm" | "md" | "lg";

type DeleteButtonProps = {
  size?: DeleteButtonSize;
  className?: string;
  disabled?: boolean;
  ariaLabel?: string;
  title?: string;
  onClick: (event: React.MouseEvent<HTMLButtonElement>) => void;
};

function sizeClassName(size: DeleteButtonSize): string {
  return `ui-delete-btn--${size}`;
}

export default function DeleteButton({
  size = "md",
  className = "",
  disabled = false,
  ariaLabel = "削除",
  title = "削除",
  onClick,
}: DeleteButtonProps) {
  return (
    <Button
      type="button"
      variant="destructive-outline"
      size="icon"
      className={[
        "ui-delete-btn",
        sizeClassName(size),
        className,
      ]
        .filter(Boolean)
        .join(" ")}
      onClick={onClick}
      aria-label={ariaLabel}
      title={title}
      disabled={disabled}
    >
      <Trash2
        className="ui-delete-btn__icon"
        aria-hidden="true"
      />
    </Button>
  );
}