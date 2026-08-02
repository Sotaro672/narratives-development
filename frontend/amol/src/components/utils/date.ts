// frontend/amol/src/components/utils/date.ts

const dateTimeFormatter = new Intl.DateTimeFormat("ja-JP", {
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
  hour: "2-digit",
  minute: "2-digit",
});

export function formatDateTime(
  value: string | null | undefined,
): string {
  const normalizedValue = value?.trim();

  if (!normalizedValue) {
    return "-";
  }

  const date = new Date(normalizedValue);

  if (Number.isNaN(date.getTime())) {
    return "-";
  }

  return dateTimeFormatter.format(date);
}