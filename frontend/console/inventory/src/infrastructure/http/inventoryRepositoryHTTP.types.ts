// frontend/console/inventory/src/infrastructure/http/inventoryRepositoryHTTP.types.ts

// ---------------------------------------------------------
// Inventory 用：商品情報ヘッダー DTO
// ---------------------------------------------------------
export type InventoryProductSummary = {
  id: string;
  productName: string;
  brandName?: string;
};

// ---------------------------------------------------------
// ✅ Inventory 一覧DTO（管理一覧）
// GET /inventory
// ---------------------------------------------------------
export type InventoryListRowDTO = {
  productBlueprintId: string;
  productName: string;

  tokenBlueprintId: string; // ✅ 必須（detail遷移のキー）
  tokenName: string;

  modelNumber: string;
  // ✅ NEW: 画面側が使う値を落とさず返す
  availableStock: number;
  reservedCount: number;
};

// ---------------------------------------------------------
// ✅ inventoryIds 解決 DTO（方針A）
// GET /inventory/ids?productBlueprintId=...&tokenBlueprintId=...
// ---------------------------------------------------------
export type InventoryIDsByProductAndTokenDTO = {
  productBlueprintId: string;
  tokenBlueprintId: string;
  inventoryIds: string[];
};

// ---------------------------------------------------------
// ✅ Inventory Detail DTOs (exported)
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
// ✅ ProductBlueprint の modelRefs（displayOrder 含む）
// domain/productBlueprint/entity.go の ModelRef に対応
// ---------------------------------------------------------
export type ProductBlueprintModelRefDTO = {
  modelId: string;
  displayOrder: number;
};

// ✅ ProductBlueprintCard に合わせる（productIdTag は string）
export type ProductBlueprintPatchDTO = {
  productName?: string | null;
  brandName?: string | null;
  itemType?: string | null;
  fit?: string | null;
  material?: string | null;
  weight?: number | null;
  qualityAssurance?: string[] | null;
  productIdTag?: string | null;
  modelRefs?: ProductBlueprintModelRefDTO[] | null;
};

// ✅ NEW: TokenBlueprint patch（Inventory 詳細で使用）
export type TokenBlueprintPatchDTO = {
  tokenName?: string | null;
  symbol?: string | null;
  brandId?: string | null;
  brandName?: string | null;
  description?: string | null;
  iconUrl?: string | null;
};

export type InventoryDetailRowDTO = {
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

  // ✅ NEW: backend DTO が返す場合がある（方針A）
  inventoryIds?: string[];

  tokenBlueprintId: string;
  productBlueprintId: string;
  modelId: string;

  productBlueprintPatch: ProductBlueprintPatchDTO;

  // ✅ NEW: tokenBlueprint patch を保持できるようにする
  tokenBlueprintPatch?: TokenBlueprintPatchDTO;

  tokenBlueprint?: TokenBlueprintSummaryDTO;
  productBlueprint?: ProductBlueprintSummaryDTO;

  rows: InventoryDetailRowDTO[];
  totalStock: number;

  updatedAt?: string;
};
