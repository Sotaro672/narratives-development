// frontend/console/inventory/src/infrastructure/http/inventoryRepositoryHTTP.types.ts

import type {
  CategoryFieldValues,
  ProductBlueprintCategorySnapshot,
} from "../../../../productBlueprint/src/domain/entity/productBlueprintCategory";

// ---------------------------------------------------------
// Inventory 用：商品情報ヘッダー DTO
// ---------------------------------------------------------
export type InventoryProductSummary = {
  id: string;
  productName: string;
  brandName?: string;
};

// ---------------------------------------------------------
// Inventory 一覧DTO（管理一覧）
// GET /inventory
// ---------------------------------------------------------
export type InventoryListRowDTO = {
  productBlueprintId: string;
  productName: string;

  tokenBlueprintId: string; // detail遷移のキー
  tokenName: string;

  modelNumber: string;
  availableStock: number;
  reservedCount: number;
};

// ---------------------------------------------------------
// inventoryIds 解決 DTO
// GET /inventory/ids?productBlueprintId=...&tokenBlueprintId=...
// ---------------------------------------------------------
export type InventoryIDsByProductAndTokenDTO = {
  productBlueprintId: string;
  tokenBlueprintId: string;
  inventoryIds: string[];
};

// ---------------------------------------------------------
// Inventory Detail DTOs
// GET /inventory/{inventoryId}
// ---------------------------------------------------------
export type TokenBlueprintSummaryDTO = {
  id: string;
  name?: string;
  symbol?: string;
};

export type ProductBlueprintSummaryDTO = {
  id: string;
  name?: string;
};

// ---------------------------------------------------------
// ProductBlueprint の modelRefs（displayOrder 含む）
// infrastructure mapper で backend raw の ModelID / DisplayOrder を
// modelId / displayOrder に変換してから保持する。
// ---------------------------------------------------------
export type ProductBlueprintModelRefDTO = {
  modelId: string;
  displayOrder: number;
};

// ---------------------------------------------------------
// ProductBlueprint patch
//
// productBlueprintCategory は ProductBlueprintCard が期待する
// ProductBlueprintCategorySnapshot と同じ型を使う。
// backend raw の ID / Code / NameJa / NameEn / Kind / Path は
// inventoryRepositoryHTTP.mappers.ts で
// id / code / nameJa / nameEn / kind / path へ変換する。
// ---------------------------------------------------------
export type ProductBlueprintPatchDTO = {
  productName?: string | null;
  description?: string | null;

  brandId?: string | null;
  brandName?: string | null;
  companyId?: string | null;

  productBlueprintCategory?: ProductBlueprintCategorySnapshot | null;
  categoryFields?: CategoryFieldValues | null;

  itemType?: string | null;
  fit?: string | null;
  material?: string | null;
  weight?: number | null;
  qualityAssurance?: string[] | null;

  productIdTag?: string | { type?: string } | null;

  modelRefs?: ProductBlueprintModelRefDTO[] | null;
};

// ---------------------------------------------------------
// TokenBlueprint patch（Inventory 詳細で使用）
// ---------------------------------------------------------
export type TokenBlueprintPatchDTO = {
  tokenName?: string | null;
  symbol?: string | null;
  brandId?: string | null;
  brandName?: string | null;
  description?: string | null;
  iconUrl?: string | null;
};

export type InventoryDetailRowDTO = {
  modelId?: string;
  tokenBlueprintId?: string;
  token?: string;
  modelNumber: string;
  size: string;
  color: string;
  rgb?: number | null;
  stock: number;
};

export type InventoryDetailDTO = {
  inventoryId: string;

  inventoryIds?: string[];

  tokenBlueprintId: string;
  productBlueprintId: string;
  modelId?: string;

  productBlueprintPatch: ProductBlueprintPatchDTO;

  tokenBlueprintPatch?: TokenBlueprintPatchDTO;

  tokenBlueprint?: TokenBlueprintSummaryDTO;
  productBlueprint?: ProductBlueprintSummaryDTO;

  rows: InventoryDetailRowDTO[];
  totalStock: number;

  updatedAt?: string;
};