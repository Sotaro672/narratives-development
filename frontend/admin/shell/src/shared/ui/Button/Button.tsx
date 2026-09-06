//frontend\admin\shell\src\shared\ui\Button\Button.tsx
import type {
  ButtonHTMLAttributes,
  ReactNode,
} from "react";

import "./Button.css";

export type ButtonVariant =
  | "primary"
  | "secondary"
  | "danger"
  | "ghost";

export type ButtonSize =
  | "sm"
  | "md"
  | "lg";

export type ButtonClassNameOptions = {
  variant?: ButtonVariant;
  size?: ButtonSize;
  iconOnly?: boolean;
  fullWidth?: boolean;
  className?: string;
};

export function getButtonClassName({
  variant = "secondary",
  size = "md",
  iconOnly = false,
  fullWidth = false,
  className = "",
}: ButtonClassNameOptions = {}): string {
  return [
    "ui-button",
    `ui-button--${variant}`,
    `ui-button--${size}`,
    iconOnly ? "ui-button--icon-only" : "",
    fullWidth ? "ui-button--full-width" : "",
    className,
  ]
    .filter(Boolean)
    .join(" ");
}

export type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  children: ReactNode;
  variant?: ButtonVariant;
  size?: ButtonSize;
  iconOnly?: boolean;
  fullWidth?: boolean;
  loading?: boolean;
};

export default function Button({
  children,
  variant = "secondary",
  size = "md",
  iconOnly = false,
  fullWidth = false,
  loading = false,
  disabled = false,
  className = "",
  type = "button",
  ...props
}: ButtonProps) {
  return (
    <button
      {...props}
      type={type}
      className={getButtonClassName({
        variant,
        size,
        iconOnly,
        fullWidth,
        className,
      })}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
    >
      {children}
    </button>
  );
}