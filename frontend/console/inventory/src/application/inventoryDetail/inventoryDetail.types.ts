// frontend/console/inventory/src/application/inventoryDetail/inventoryDetail.types.ts

import type { InventoryRow } from "../inventoryTypes";
import type {
  ProductBlueprintPatchDTO,
  TokenBlueprintPatchDTO,
} from "../../infrastructure/http/inventoryRepositoryHTTP";

export type InventoryDetailViewModel = {
  /** 画面用の一意キー（pbId + tbId） */
  inventoryKey: string;

  // ✅ 画面でそのまま表示できるように ViewModel 直下にも持つ（重要）
  productName?: string;
  brandName?: string;

  // ✅ tokenBlueprint patch（token名など）
  tokenBlueprintPatch?: TokenBlueprintPatchDTO;

  // 元データも保持（編集フォームなどで利用する想定）
  productBlueprintPatch: ProductBlueprintPatchDTO;

  rows: InventoryRow[];
  totalStock: number;

  /** max(updatedAt) */
  updatedAt?: string;
};
