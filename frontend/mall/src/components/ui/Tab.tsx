// frontend/amol/src/components/ui/Tab.tsx
import type { ButtonHTMLAttributes, ReactNode } from "react";

import "./tab.css";

type TabProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  children: ReactNode;
};

export default function Tab({
  children,
  className = "",
  type = "button",
  ...props
}: TabProps) {
  const classes = ["tab", className].filter(Boolean).join(" ");

  return (
    <button type={type} className={classes} {...props}>
      {children}
    </button>
  );
}