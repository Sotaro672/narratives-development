//frontend\mall\src\components\ui\StatePanel.tsx
import type { ReactNode } from "react";

import "./StatePanel.css";

export type StatePanelVariant =
  | "default"
  | "empty"
  | "loading"
  | "error"
  | "success";

type StatePanelProps = {
  title: ReactNode;
  description?: ReactNode;
  icon?: ReactNode;
  action?: ReactNode;
  variant?: StatePanelVariant;
  className?: string;
};

export default function StatePanel({
  title,
  description,
  icon,
  action,
  variant = "default",
  className = "",
}: StatePanelProps) {
  const classes = [
    "ui-state-panel",
    `ui-state-panel--${variant}`,
    className,
  ]
    .filter(Boolean)
    .join(" ");

  const role =
    variant === "error"
      ? "alert"
      : variant === "loading" || variant === "success"
        ? "status"
        : undefined;

  return (
    <div
      className={classes}
      role={role}
      aria-live={role ? "polite" : undefined}
      aria-busy={variant === "loading" || undefined}
    >
      {icon ? (
        <div className="ui-state-panel__icon" aria-hidden="true">
          {icon}
        </div>
      ) : variant === "loading" ? (
        <span
          className="ui-state-panel__spinner"
          aria-hidden="true"
        />
      ) : null}

      <div className="ui-state-panel__content">
        <p className="ui-state-panel__title">
          {title}
        </p>

        {description ? (
          <p className="ui-state-panel__description">
            {description}
          </p>
        ) : null}
      </div>

      {action ? (
        <div className="ui-state-panel__action">
          {action}
        </div>
      ) : null}
    </div>
  );
}