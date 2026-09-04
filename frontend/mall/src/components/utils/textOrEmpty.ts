// frontend/amol/src/components/utils/textOrEmpty.ts

export function textOrEmpty(
  value: unknown,
): string {
  return String(value ?? "").trim();
}