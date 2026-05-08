// frontend/console/shell/src/shared/util/color.ts

/**
 * Shared color utilities for console packages.
 *
 * Conventions:
 * - "rgb int" means 0xRRGGBB (decimal number in JS/TS).
 * - Hex string accepts "#RRGGBB" or "RRGGBB" (case-insensitive).
 */

/** "#RRGGBB" | "RRGGBB" -> 0xRRGGBB */
export function hexToRgbInt(hex?: string): number | undefined {
  if (!hex) return undefined;

  const trimmed = String(hex).trim();
  if (!trimmed) return undefined;

  const h = trimmed.startsWith("#") ? trimmed.slice(1) : trimmed;
  if (!/^[0-9a-fA-F]{6}$/.test(h)) return undefined;

  const parsed = Number.parseInt(h, 16);
  if (Number.isNaN(parsed)) return undefined;

  // safety: clamp to RGB range
  if (parsed < 0x000000 || parsed > 0xffffff) return undefined;

  return parsed;
}

/**
 * 0xRRGGBB -> "#RRGGBB"
 * - Accepts string too (best-effort): "#RRGGBB" / "RRGGBB" / "0xRRGGBB" / "16777215"
 */
export function rgbIntToHex(
  rgb?: number | string | null,
): string | undefined {
  if (rgb == null) return undefined;

  // If it's already a string, try to interpret it.
  if (typeof rgb === "string") {
    const s = rgb.trim();
    if (!s) return undefined;

    // "#RRGGBB"
    if (/^#[0-9a-fA-F]{6}$/.test(s)) return s.toUpperCase();

    // "RRGGBB" (hex without '#')
    if (/^[0-9a-fA-F]{6}$/.test(s)) return `#${s.toUpperCase()}`;

    // "0xRRGGBB"
    if (/^0x[0-9a-fA-F]{6}$/.test(s)) return `#${s.slice(2).toUpperCase()}`;

    // numeric string (decimal)
    if (/^\d+$/.test(s)) {
      const n = Number.parseInt(s, 10);
      if (Number.isNaN(n)) return undefined;
      if (n < 0x000000 || n > 0xffffff) return undefined;
      return `#${Math.trunc(n).toString(16).padStart(6, "0").toUpperCase()}`;
    }

    return undefined;
  }

  // number
  if (typeof rgb !== "number" || Number.isNaN(rgb)) return undefined;

  const n = Math.trunc(rgb);
  if (n < 0x000000 || n > 0xffffff) return undefined;

  return `#${n.toString(16).padStart(6, "0").toUpperCase()}`;
}

/**
 * Best-effort coercion to rgb int (0xRRGGBB).
 * Accepts:
 * - number (0..0xFFFFFF)
 * - numeric string ("16777215")
 * - hex string ("#FFFFFF" / "FFFFFF" / "0xFFFFFF")
 */
export function coerceRgbInt(value: unknown): number | undefined {
  if (value == null) return undefined;

  if (typeof value === "number") {
    if (Number.isNaN(value)) return undefined;
    const n = Math.trunc(value);
    if (n < 0x000000 || n > 0xffffff) return undefined;
    return n;
  }

  if (typeof value === "string") {
    const s = value.trim();
    if (!s) return undefined;

    // try hex first (#RRGGBB / RRGGBB)
    const hexParsed = hexToRgbInt(s);
    if (typeof hexParsed === "number") return hexParsed;

    // "0xRRGGBB"
    if (/^0x[0-9a-fA-F]{6}$/.test(s)) {
      const n = Number.parseInt(s.slice(2), 16);
      if (Number.isNaN(n)) return undefined;
      if (n < 0x000000 || n > 0xffffff) return undefined;
      return n;
    }

    // then numeric (decimal)
    if (/^\d+$/.test(s)) {
      const n = Number.parseInt(s, 10);
      if (Number.isNaN(n)) return undefined;
      if (n < 0x000000 || n > 0xffffff) return undefined;
      return n;
    }
  }

  return undefined;
}
