// frontend/admin/shell/src/shared/ui/TextLink/TextLink.tsx

import type { ButtonHTMLAttributes, ReactNode } from "react";

import "./TextLink.css";

export type TextLinkTone = "inherit" | "accent";

export type TextLinkProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  children: ReactNode;
  tone?: TextLinkTone;
};

export default function TextLink({
  children,
  tone = "inherit",
  className = "",
  type = "button",
  ...props
}: TextLinkProps) {
  const classes = [
    "ui-text-link",
    `ui-text-link--${tone}`,
    className,
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <button {...props} type={type} className={classes}>
      {children}
    </button>
  );
}