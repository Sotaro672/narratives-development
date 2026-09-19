// frontend/console/shell/src/shared/ui/progress.tsx

import * as React from "react";

import "./progress.css";

function cn(...classes: Array<string | undefined | false | null>): string {
  return classes.filter(Boolean).join(" ");
}

export function clampProgressValue(value: number): number {
  if (!Number.isFinite(value)) {
    return 0;
  }

  return Math.min(100, Math.max(0, value));
}

export function formatProgressBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) {
    return "0 B";
  }

  const units = ["B", "KB", "MB", "GB"];
  let value = bytes;
  let unitIndex = 0;

  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }

  const digits = unitIndex === 0 ? 0 : value >= 10 ? 1 : 2;
  return `${value.toFixed(digits)} ${units[unitIndex]}`;
}

export type ProgressProps = {
  value: number;
  label?: React.ReactNode;
  ariaLabel: string;
  transferredBytes?: number;
  totalBytes?: number;
  showPercentage?: boolean;
  className?: string;
};

export function Progress({
  value,
  label,
  ariaLabel,
  transferredBytes,
  totalBytes,
  showPercentage = true,
  className,
}: ProgressProps) {
  const normalizedValue = clampProgressValue(value);

  const showBytes =
    typeof transferredBytes === "number" &&
    typeof totalBytes === "number" &&
    totalBytes > 0;

  return (
    <div className={cn("progress", className)}>
      {label || showPercentage ? (
        <div className="progress__header">
          {label ? (
            <span className="progress__label">
              {label}
            </span>
          ) : (
            <span aria-hidden="true" />
          )}

          {showPercentage ? (
            <span className="progress__percentage">
              {normalizedValue}%
            </span>
          ) : null}
        </div>
      ) : null}

      <div
        className="progress__track"
        role="progressbar"
        aria-label={ariaLabel}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={normalizedValue}
      >
        <div
          className="progress__bar"
          style={{ width: `${normalizedValue}%` }}
        />
      </div>

      {showBytes ? (
        <div className="progress__details">
          {formatProgressBytes(transferredBytes)}
          {" / "}
          {formatProgressBytes(totalBytes)}
        </div>
      ) : null}
    </div>
  );
}

export type ProgressIndeterminateProps = {
  ariaLabel: string;
  className?: string;
};

export function ProgressIndeterminate({
  ariaLabel,
  className,
}: ProgressIndeterminateProps) {
  return (
    <div
      className={cn("progress-indeterminate", className)}
      role="progressbar"
      aria-label={ariaLabel}
    >
      <div className="progress-indeterminate__bar" />
    </div>
  );
}

export type ProgressCurrentProps = {
  label: React.ReactNode;
  value: string;
  title?: string;
  className?: string;
};

export function ProgressCurrent({
  label,
  value,
  title,
  className,
}: ProgressCurrentProps) {
  return (
    <div className={cn("progress-current", className)}>
      <span className="progress-current__label">
        {label}
      </span>

      <span
        className="progress-current__value"
        title={title ?? value}
      >
        {value}
      </span>
    </div>
  );
}

export type ProgressMetricProps = {
  label: React.ReactNode;
  value: React.ReactNode;
  className?: string;
};

export function ProgressMetric({
  label,
  value,
  className,
}: ProgressMetricProps) {
  return (
    <div className={cn("progress-metric", className)}>
      <span className="progress-metric__label">
        {label}
      </span>

      <strong className="progress-metric__value">
        {value}
      </strong>
    </div>
  );
}

export type ProgressMessageVariant =
  | "warning"
  | "notice"
  | "error";

export type ProgressMessageProps = React.HTMLAttributes<HTMLDivElement> & {
  variant: ProgressMessageVariant;
};

export function ProgressMessage({
  variant,
  className,
  role,
  children,
  ...props
}: ProgressMessageProps) {
  const resolvedRole =
    role ??
    (variant === "warning" || variant === "error"
      ? "alert"
      : undefined);

  return (
    <div
      className={cn(
        "progress-message",
        `progress-message--${variant}`,
        className,
      )}
      role={resolvedRole}
      {...props}
    >
      {children}
    </div>
  );
}

export default Progress;