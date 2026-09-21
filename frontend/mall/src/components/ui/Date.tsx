// frontend/mall/src/components/ui/Date.tsx

import type { HTMLAttributes } from "react";

export type DateVariant = "compact" | "dateTime";

type DateProps = Omit<
  HTMLAttributes<HTMLTimeElement>,
  "children" | "dateTime"
> & {
  value: string | null | undefined;
  variant?: DateVariant;
  fallback?: string;
};

const timeFormatter = new Intl.DateTimeFormat("ja-JP", {
  hour: "2-digit",
  minute: "2-digit",
});

const monthDayFormatter = new Intl.DateTimeFormat("ja-JP", {
  month: "2-digit",
  day: "2-digit",
});

const yearMonthDayFormatter = new Intl.DateTimeFormat("ja-JP", {
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
});

const dateTimeFormatter = new Intl.DateTimeFormat("ja-JP", {
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
  hour: "2-digit",
  minute: "2-digit",
});

function parseDate(
  value: string | null | undefined,
): globalThis.Date | null {
  const normalizedValue = value?.trim();

  if (!normalizedValue) {
    return null;
  }

  const date = new globalThis.Date(normalizedValue);

  return Number.isNaN(date.getTime()) ? null : date;
}

export function formatCompactDate(
  value: string | null | undefined,
  now = new globalThis.Date(),
): string {
  const date = parseDate(value);

  if (!date) {
    return "";
  }

  const isToday =
    date.getFullYear() === now.getFullYear() &&
    date.getMonth() === now.getMonth() &&
    date.getDate() === now.getDate();

  if (isToday) {
    return timeFormatter.format(date);
  }

  if (date.getFullYear() === now.getFullYear()) {
    return monthDayFormatter.format(date);
  }

  return yearMonthDayFormatter.format(date);
}

export function formatDateTime(
  value: string | null | undefined,
): string {
  const date = parseDate(value);

  return date ? dateTimeFormatter.format(date) : "-";
}

export default function DateDisplay({
  value,
  variant = "dateTime",
  fallback,
  ...props
}: DateProps) {
  const normalizedValue = value?.trim() ?? "";
  const formattedValue =
    variant === "compact"
      ? formatCompactDate(normalizedValue)
      : formatDateTime(normalizedValue);

  const defaultFallback =
    variant === "compact" ? "" : "-";

  const label =
    formattedValue || fallback || defaultFallback;

  if (!label) {
    return null;
  }

  if (!normalizedValue || !parseDate(normalizedValue)) {
    return <span>{label}</span>;
  }

  return (
    <time
      {...props}
      dateTime={normalizedValue}
    >
      {label}
    </time>
  );
}