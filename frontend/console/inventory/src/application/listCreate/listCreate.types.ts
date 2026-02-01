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
 */
import type { PriceRow as ListPriceRow } from "../../../../list/src/presentation/hook/usePriceCard";

// ✅ Hook 側で使う Ref 型（useRef<HTMLInputElement | null>(null) を許容）
export type ImageInputRef = RefObject<HTMLInputElement | null>;

// 依存は維持しつつ、inventory/application 側で型名を統一して扱う
export type PriceRow = ListPriceRow;

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
