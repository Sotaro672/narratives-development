// frontend/console/inventory/src/application/inventoryDetail/inventoryDetail.utils.ts

export function asString(v: unknown): string {
  return String(v ?? "").trim();
}

export function uniqStrings(xs: string[]): string[] {
  const out: string[] = [];
  const seen = new Set<string>();
  for (const x of xs) {
    const s = asString(x);
    if (!s) continue;
    if (seen.has(s)) continue;
    seen.add(s);
    out.push(s);
  }
  return out;
}
