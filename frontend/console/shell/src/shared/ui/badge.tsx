// frontend/console/shell/src/shared/ui/badge.tsx

import type { CSSProperties, ReactNode } from "react";

import "./badge.css";

export type BadgeVariant =
  | "default"
  | "secondary"
  | "active"
  | "danger";

interface BadgeProps {
  children: ReactNode;
  className?: string;
  style?: CSSProperties;
  variant?: BadgeVariant;
}

export function Badge({
  children,
  className = "",
  style,
  variant = "secondary",
}: BadgeProps) {
  return (
    <span
      className={`badge badge--${variant} ${className}`.trim()}
      style={style}
    >
      {children}
    </span>
  );
}