// frontend/amol/src/features/scan-result/utils/guards.ts

import {
  isFiniteNumber,
  isRecord,
} from "../../shared/utils/typeGuards";

export function getRecord(
  value: unknown,
  key: string,
): Record<string, unknown> | null {
  if (!isRecord(value)) {
    return null;
  }

  const raw = value[key];

  return isRecord(raw) ? raw : null;
}

export function getString(
  value: unknown,
  key: string,
): string {
  if (!isRecord(value)) {
    return "";
  }

  const raw = value[key];

  return typeof raw === "string"
    ? raw
    : "";
}

export function getNumber(
  value: unknown,
  key: string,
): number | null {
  if (!isRecord(value)) {
    return null;
  }

  const raw = value[key];

  return isFiniteNumber(raw)
    ? raw
    : null;
}

export function getStringArray(
  value: unknown,
  key: string,
): string[] {
  if (!isRecord(value)) {
    return [];
  }

  const raw = value[key];

  if (!Array.isArray(raw)) {
    return [];
  }

  return raw
    .filter(
      (item): item is string =>
        typeof item === "string",
    )
    .map((item) => item.trim())
    .filter(Boolean);
}