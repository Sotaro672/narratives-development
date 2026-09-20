//frontend\mall\src\components\ui\Card.tsx
import type { HTMLAttributes, ReactNode } from "react";

import "./Card.css";

type CardPadding = "sm" | "md" | "lg";

type CardProps = HTMLAttributes<HTMLDivElement> & {
  children: ReactNode;
  interactive?: boolean;
  highlighted?: boolean;
  busy?: boolean;
  padding?: CardPadding;
};

export default function Card({
  children,
  interactive = false,
  highlighted = false,
  busy = false,
  padding = "md",
  className = "",
  tabIndex,
  role,
  ...props
}: CardProps) {
  const classes = [
    "ui-card",
    `ui-card--padding-${padding}`,
    interactive ? "ui-card--interactive" : "",
    highlighted ? "ui-card--highlighted" : "",
    busy ? "ui-card--busy" : "",
    className,
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <div
      className={classes}
      role={role ?? (interactive ? "button" : undefined)}
      tabIndex={tabIndex ?? (interactive && !busy ? 0 : undefined)}
      aria-busy={busy || undefined}
      {...props}
    >
      {children}
    </div>
  );
}