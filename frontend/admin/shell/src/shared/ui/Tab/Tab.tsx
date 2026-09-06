// frontend/admin/shell/src/shared/ui/Tab.tsx

import type { HTMLAttributes, ReactNode } from "react";

import "./Tab.css";

export type TabTone =
  | "neutral"
  | "warning"
  | "success"
  | "danger";

type TabProps = HTMLAttributes<HTMLSpanElement> & {
  children: ReactNode;
  tone?: TabTone;
};

export default function Tab({
  children,
  tone = "neutral",
  className = "",
  ...props
}: TabProps) {
  const classes = [
    "ui-tab",
    `ui-tab--${tone}`,
    className,
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <span className={classes} {...props}>
      {children}
    </span>
  );
}