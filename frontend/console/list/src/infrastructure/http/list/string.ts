//frontend\console\list\src\infrastructure\http\list\string.ts
export function s(v: unknown): string {
  return String(v ?? "").trim();
}
