// frontend/console/inventory/src/application/inventoryDetail/inventoryDetail.types.ts

import type { InventoryRow } from "../inventoryTypes";
import type {
  ProductBlueprintPatchDTO,
  TokenBlueprintPatchDTO,
} from "../../infrastructure/http/inventoryRepositoryHTTP";

// DTO 側に brandName が増えても落とさないための拡張型（UIで参照しやすくする）
export type ProductBlueprintPatchDTOEx = ProductBlueprintPatchDTO & {
  brandId?: string;
  brandName?: string;
  productName?: string;
};

// ✅ tokenBlueprint patch を ViewModel に保持できるようにする
export type TokenBlueprintPatchDTOEx = TokenBlueprintPatchDTO & {
  tokenName?: string;
  brandId?: string;
  brandName?: string;
};

export type InventoryDetailViewModel = {
  /** 画面用の一意キー（pbId + tbId） */
  inventoryKey: string;

  /** 方針A: 詳細が対象とする inventoryId の集合 */
  inventoryIds: string[];

  tokenBlueprintId: string;
  productBlueprintId: string;

  /** 方針Aでは原則空 */
  modelId: string;

  // ✅ 画面でそのまま表示できるように ViewModel 直下にも持つ（重要）
  productName?: string;
  brandId?: string;
  brandName?: string;

  // ✅ tokenBlueprint patch（token名など）
  tokenBlueprintPatch?: TokenBlueprintPatchDTOEx;

  // 元データも保持（編集フォームなどで利用する想定）
  productBlueprintPatch: ProductBlueprintPatchDTOEx;

  rows: InventoryRow[];
  totalStock: number;

  /** max(updatedAt) */
  updatedAt?: string;
};
