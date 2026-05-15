// frontend/console/inventory/src/application/inventoryDetail/inventoryDetail.types.ts

import type { InventoryRow } from "../inventoryTypes";
import type {
  ProductBlueprintPatchDTO,
  TokenBlueprintPatchDTO,
} from "../../infrastructure/http/inventoryRepositoryHTTP.types";

export type InventoryDetailViewModel = {
  // inventory docId を正とする
  inventoryId: string;

  // inventory テーブルに両方記載されている前提で、そこから取得する（split/合成しない）
  productBlueprintId: string;
  tokenBlueprintId: string;

  // Header 表示用（productName / tokenName のみ）
  productName: string;
  tokenName: string;
  headerTitle: string;

  productBlueprintPatch: ProductBlueprintPatchDTO;
  tokenBlueprintPatch?: TokenBlueprintPatchDTO;

  updatedAt?: string;
  totalStock: number;

  // InventoryCard に渡す最小
  rows: InventoryRow[];
};