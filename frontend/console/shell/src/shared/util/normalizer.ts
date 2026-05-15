// frontend/console/shell/src/shared/util/normalizer.ts

export function s(v: unknown): string {
  return String(v ?? "").trim();
}