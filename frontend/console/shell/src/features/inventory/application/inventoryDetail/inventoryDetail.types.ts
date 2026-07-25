// frontend/console/inventory/src/application/inventoryDetail/inventoryDetail.types.ts

import type { InventoryRow } from "../inventoryTypes";
import type {
  ProductBlueprintPatchDTO,
  TokenBlueprintPatchDTO,
} from "../../infrastructure/http/inventoryRepositoryHTTP.types";

export type InventoryDetailViewModel = {
  // inventory docId を正とする
  inventoryId: string;

  // GET /inventory/{inventoryId} の response に含まれる値を正とする
  // split / 合成 / 追加取得はしない
  productBlueprintId: string;
  tokenBlueprintId: string;

  // Header 表示用
  productName: string;
  tokenName: string;
  headerTitle: string;

  // ProductBlueprint category 表示用
  productBlueprintCategoryName: string;
  productBlueprintCategoryCode?: string;
  productBlueprintCategoryKind?: string;
  categoryFields?: Record<string, unknown> | null;

  // GET /inventory/{inventoryId} の response に含まれる patch を正とする
  productBlueprintPatch: ProductBlueprintPatchDTO;
  tokenBlueprintPatch?: TokenBlueprintPatchDTO;

  updatedAt?: string;
  totalStock: number;

  // InventoryCard に渡す行
  // rows は GET /inventory/{inventoryId} の rows を正とする
  rows: InventoryRow[];
};