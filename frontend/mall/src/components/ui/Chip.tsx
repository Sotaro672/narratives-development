//frontend\mall\src\components\ui\Chip.tsx
import type { ButtonHTMLAttributes, ReactNode } from "react";

import "./Chip.css";

type ChipVariant = "neutral" | "info" | "success" | "warning" | "danger";
type ChipSize = "sm" | "md";

type ChipProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  children: ReactNode;
  variant?: ChipVariant;
  size?: ChipSize;
  selected?: boolean;
};

export default function Chip({
  children,
  variant = "neutral",
  size = "md",
  selected = false,
  className = "",
  type = "button",
  ...props
}: ChipProps) {
  const classes = [
    "ui-chip",
    `ui-chip--${variant}`,
    `ui-chip--${size}`,
    selected ? "ui-chip--selected" : "",
    className,
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <button
      type={type}
      className={classes}
      aria-pressed={selected}
      {...props}
    >
      {children}
    </button>
  );
}