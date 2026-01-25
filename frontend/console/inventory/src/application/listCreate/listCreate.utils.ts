// frontend/console/inventory/src/application/listCreate/listCreate.utils.ts

export function s(v: unknown): string {
  return String(v ?? "").trim();
}

/**
 * ✅ 方針A: inventoryId は "pb__tb" をそのまま通す（splitしない）
 */
export function normalizeInventoryId(v: unknown): string {
  return s(v);
}

/**
 * ✅ listId は backend が採番する想定。
 * "__" を含んでいても正当なIDになり得るので split しない。
 */
export function normalizeListId(v: unknown): string {
  return s(v);
}

export function toNumberOrNull(v: unknown): number | null {
  if (v === null || v === undefined) return null;
  const n = typeof v === "number" ? v : Number(String(v).trim());
  if (!Number.isFinite(n)) return null;
  return Math.floor(n);
}
