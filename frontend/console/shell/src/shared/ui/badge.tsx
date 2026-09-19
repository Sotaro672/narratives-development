// frontend/console/shell/src/shared/ui/badge.tsx

import type {
  CSSProperties,
  HTMLAttributes,
  ReactNode,
} from "react";

import "./badge.css";

export type BadgeVariant =
  | "default"
  | "secondary"
  | "active"
  | "info"
  | "success"
  | "warning"
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
  const classNames = [
    "badge",
    `badge--${variant}`,
    className,
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <span
      className={classNames}
      style={style}
    >
      {children}
    </span>
  );
}

export interface BadgeGroupProps
  extends HTMLAttributes<HTMLDivElement> {}

export function BadgeGroup({
  className = "",
  ...props
}: BadgeGroupProps) {
  const classNames = [
    "badge-group",
    className,
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <div
      className={classNames}
      {...props}
    />
  );
}