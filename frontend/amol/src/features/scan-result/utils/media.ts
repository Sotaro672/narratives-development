//frontend\amol\src\features\scan-result\utils\media.ts
export function isUsableHttpImageUrl(value: string | null | undefined): boolean {
  const s = String(value ?? "").trim();
  if (!s) return false;
  if (s === ".keep") return false;
  if (s.startsWith("gs://")) return false;

  try {
    const u = new URL(s);
    return u.protocol === "http:" || u.protocol === "https:";
  } catch {
    return false;
  }
}

export function normalizeImageUrl(value: string | null | undefined): string {
  const s = String(value ?? "").trim();
  if (!isUsableHttpImageUrl(s)) return "";
  return encodeURI(s);
}