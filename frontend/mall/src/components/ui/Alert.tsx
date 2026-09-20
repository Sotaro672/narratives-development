//frontend\mall\src\components\ui\Alert.tsx
import type { HTMLAttributes, ReactNode } from "react";

import "./Alert.css";

type AlertVariant = "info" | "success" | "warning" | "error";

type AlertProps = HTMLAttributes<HTMLDivElement> & {
  children: ReactNode;
  variant?: AlertVariant;
};

export default function Alert({
  children,
  variant = "info",
  className = "",
  role,
  ...props
}: AlertProps) {
  const classes = [
    "ui-alert",
    `ui-alert--${variant}`,
    className,
  ]
    .filter(Boolean)
    .join(" ");

  const resolvedRole =
    role ?? (variant === "warning" || variant === "error" ? "alert" : undefined);

  return (
    <div
      className={classes}
      role={resolvedRole}
      {...props}
    >
      {children}
    </div>
  );
}