// frontend/console/inventory/src/application/listCreate/listCreate.types.ts
import type { RefObject } from "react";

/**
 * ⚠ Layering note:
 * 本来、inventory の application 層が list の presentation 層（hook）型に依存するのは依存方向として望ましくありません。
 * ただし今回は既存実装・移行コストを優先し、この依存を「コメントで注意喚起した上で」維持します。
 *
 * 将来的には:
 * - inventory/application 側に PriceRowVM を定義
 * - list/presentation は inventory/application の型を参照
 * の方向に寄せるのが推奨です。
 *
 * ✅ 今回の改修:
 * - list/presentation への依存（usePriceCard の型 import）を廃止し、
 *   inventory/application 側で priceCard.types.ts を正とする
 */
import type { PriceRow } from "./priceCard.types";

// ✅ Hook 側で使う Ref 型（useRef<HTMLInputElement | null>(null) を許容）
export type ImageInputRef = RefObject<HTMLInputElement | null>;

// ============================================================
// ✅ PriceRow（期待値）
// - infrastructure の ListCreatePriceRowDTO をここで PriceRow として扱い、
//   list/presentation の usePriceCard → PriceRowVM へそのまま流す
// ============================================================

export type ListCreateRouteParams = {
  inventoryId?: string; // 期待値: "pb__tb"
  productBlueprintId?: string; // optional
  tokenBlueprintId?: string; // optional
};

export type ResolvedListCreateParams = {
  inventoryId: string; // ✅ 常に "pb__tb" を保持
  productBlueprintId: string;
  tokenBlueprintId: string;
  raw: ListCreateRouteParams;
};

// ============================================================
// ✅ POST /lists: 期待値どおり「modelId + price のみ」
// ============================================================

export type CreateListPriceRow = {
  modelId: string;
  price: number | null;
};

// ------------------------------------------------------------
// Re-export (optional but convenient)
// ------------------------------------------------------------
export type { PriceRow };
