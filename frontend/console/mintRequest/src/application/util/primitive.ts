// frontend/console/mintRequest/src/application/util/primitive.ts

/**
 * Returns trimmed string if non-empty, otherwise empty string.
 */
export function asNonEmptyString(v: unknown): string {
  return typeof v === "string" && v.trim() ? v.trim() : "";
}

/**
 * Returns trimmed string if non-empty, otherwise null.
 */
export function asStringOrNull(v: unknown): string | null {
  const s = typeof v === "string" ? v.trim() : "";
  return s ? s : null;
}

/**
 * Parse number safely. Non-finite -> 0.
 */
export function asNumber0(v: unknown): number {
  const n = typeof v === "number" ? v : Number(v);
  return Number.isFinite(n) ? n : 0;
}

/**
 * Uniq + trim + drop empty.
 */
export function uniqStrings(xs: unknown[]): string[] {
  const seen = new Set<string>();
  const out: string[] = [];

  for (const x of xs ?? []) {
    const s = String(x ?? "").trim();
    if (!s) continue;
    if (seen.has(s)) continue;
    seen.add(s);
    out.push(s);
  }

  return out;
}

/**
 * Attempt to produce ISO string from common inputs.
 * - string: return as-is
 * - Date: toISOString
 * - other: String(v)
 */
export function asMaybeISO(v: unknown): string {
  if (v === null || v === undefined) return "";
  if (typeof v === "string") return v;
  if (v instanceof Date) return v.toISOString();
  return String(v);
}
