// frontend/console/mintRequest/src/application/util/primitive.ts

/**
 * Returns trimmed string if non-empty, otherwise empty string.
 */
export function asNonEmptyString(
  value: unknown,
): string {
  if (typeof value !== "string") {
    return "";
  }

  return value.trim();
}

/**
 * Returns trimmed string if non-empty, otherwise null.
 */
export function asStringOrNull(
  value: unknown,
): string | null {
  if (typeof value !== "string") {
    return null;
  }

  const normalizedValue =
    value.trim();

  return normalizedValue || null;
}

/**
 * Parse number safely. Non-finite -> 0.
 */
export function asNumber0(
  value: unknown,
): number {
  const numberValue =
    typeof value === "number"
      ? value
      : Number(value);

  return Number.isFinite(numberValue)
    ? numberValue
    : 0;
}

/**
 * Attempt to produce ISO string from common inputs.
 * - string: return as-is
 * - Date: toISOString
 * - other: String(value)
 */
export function asMaybeISO(
  value: unknown,
): string {
  if (
    value === null ||
    value === undefined
  ) {
    return "";
  }

  if (typeof value === "string") {
    return value;
  }

  if (value instanceof Date) {
    return value.toISOString();
  }

  return String(value);
}