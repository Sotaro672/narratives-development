// frontend/amol/src/components/ui/Tab.tsx

import type { ButtonHTMLAttributes, ReactNode } from "react";

import "./tab.css";

export type TabVariant = "pill" | "underline";

type TabProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  children: ReactNode;
  variant?: TabVariant;
  selected?: boolean;
  fullWidth?: boolean;
};

export default function Tab({
  children,
  variant = "pill",
  selected = false,
  fullWidth = false,
  className = "",
  type = "button",
  ...props
}: TabProps) {
  const classes = [
    "tab",
    `tab--${variant}`,
    selected ? "tab--selected" : "",
    fullWidth ? "tab--full-width" : "",
    className,
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <button
      type={type}
      className={classes}
      aria-selected={props["aria-selected"] ?? selected}
      {...props}
    >
      {children}
    </button>
  );
}