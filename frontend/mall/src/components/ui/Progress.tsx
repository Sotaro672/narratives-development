//frontend\mall\src\components\ui\Progress.tsx
import type { HTMLAttributes } from "react";

import "./Progress.css";

type ProgressVariant = "primary" | "info" | "success" | "danger";
type ProgressSize = "sm" | "md";

type ProgressProps = Omit<HTMLAttributes<HTMLDivElement>, "aria-label"> & {
  "aria-label": string;
  value?: number;
  max?: number;
  indeterminate?: boolean;
  variant?: ProgressVariant;
  size?: ProgressSize;
};

export default function Progress({
  "aria-label": ariaLabel,
  value = 0,
  max = 100,
  indeterminate = false,
  variant = "primary",
  size = "md",
  className = "",
  ...props
}: ProgressProps) {
  const safeMax = Number.isFinite(max) && max > 0 ? max : 100;
  const safeValue = Number.isFinite(value)
    ? Math.min(safeMax, Math.max(0, value))
    : 0;
  const percentage = (safeValue / safeMax) * 100;

  const classes = [
    "ui-progress",
    `ui-progress--${variant}`,
    `ui-progress--${size}`,
    indeterminate ? "ui-progress--indeterminate" : "",
    className,
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <div
      className={classes}
      role="progressbar"
      aria-label={ariaLabel}
      aria-valuemin={indeterminate ? undefined : 0}
      aria-valuemax={indeterminate ? undefined : safeMax}
      aria-valuenow={indeterminate ? undefined : safeValue}
      {...props}
    >
      <div
        className="ui-progress__bar"
        style={indeterminate ? undefined : { width: `${percentage}%` }}
      />
    </div>
  );
}