// frontend/amol/src/components/ui/TextButton.tsx
import type { ButtonHTMLAttributes, ReactNode } from "react";

import "./text-button.css";

type TextButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  children: ReactNode;
};

export default function TextButton({
  children,
  className = "",
  type = "button",
  ...props
}: TextButtonProps) {
  const classes = ["text-button", className].filter(Boolean).join(" ");

  return (
    <button type={type} className={classes} {...props}>
      {children}
    </button>
  );
}