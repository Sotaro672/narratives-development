//frontend\console\list\src\infrastructure\http\list\ids.ts
import { s } from "./string";

/**
 * ✅ list のドキュメントID用の正規化
 * - これは "listId__imageId" など事故混入の保険
 * - ただし inventoryId (pb__tb) には絶対に使わない（方針A）
 */
export function normalizeListDocId(v: unknown): string {
  const id = s(v);
  if (!id) return "";
  return id.split("__")[0];
}
