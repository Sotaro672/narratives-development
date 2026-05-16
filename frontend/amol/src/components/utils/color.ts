// frontend/amol/src/components/utils/color.ts

export function rgbToCssColor(rgb: number): string {
  const value = Number.isFinite(rgb) ? rgb : 0;
  const normalized = value & 0xffffff;

  return `#${normalized.toString(16).padStart(6, "0")}`;
}

export function toSafeColorRGB(value: unknown): number {
  const n = Number(value);
  return Number.isFinite(n) ? n : 0;
}