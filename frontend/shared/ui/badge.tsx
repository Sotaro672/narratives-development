import React from "react";
import type { CSSProperties, ReactNode } from "react";

interface BadgeProps {
  children: ReactNode;
  className?: string;
  style?: CSSProperties;
  variant?: "secondary" | "default";
}

export function Badge({
  children,
  className = "",
  style,
  variant = "secondary",
}: BadgeProps) {
  const bg = variant === "secondary" ? "#eef2ff" : "#e5e7eb";
  const color = variant === "secondary" ? "#3730a3" : "#111827";

  return (
    <span
      className={className}
      style={{
        display: "inline-flex",
        alignItems: "center",
        justifyContent: "center",
        borderRadius: 9999,
        fontSize: 11,
        fontWeight: 600,
        padding: "2px 6px",
        background: bg,
        color,
        ...style,
      }}
    >
      {children}
    </span>
  );
}
