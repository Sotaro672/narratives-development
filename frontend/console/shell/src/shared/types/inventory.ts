// frontend/console/shell/src/shared/types/inventory.ts

// =========================================================
// Inventory common
// =========================================================

export type InventoryProductBlueprintCategoryKind =
  | "apparel"
  | "alcohol"
  | "cosmetics"
  | "healthcare"
  | "other";

// =========================================================
// Inventory category fields
// =========================================================

export type InventoryCategoryFieldPrimitiveValue = string | number | boolean | null;
export type InventoryCategoryFieldArrayValue = InventoryCategoryFieldPrimitiveValue[];
export type InventoryCategoryFieldObjectValue = Record<string, InventoryCategoryFieldPrimitiveValue>;
export type InventoryCategoryFieldValue =
  | InventoryCategoryFieldPrimitiveValue
  | InventoryCategoryFieldArrayValue
  | InventoryCategoryFieldObjectValue;
export type InventoryCategoryFieldValues = Record<string, InventoryCategoryFieldValue>;

// =========================================================
// Inventory Management
// GET /inventory
// =========================================================

export type InventoryListRowDTO = {
  productBlueprintId: string;
  productName: string;
  tokenBlueprintId: string;
  tokenName: string;
  availableStock: number;
  reservedCount: number;
};

export type InventorySortKey =
  | "productName"
  | "tokenName"
  | "availableStock"
  | "reservedCount";

// =========================================================
// Inventory ProductBlueprint read model
// GET /inventory/{inventoryId}
// =========================================================

export type InventoryProductBlueprintCategoryDTO = {
  id: string;
  code: string;
  nameJa: string;
  nameEn: string;
  kind: InventoryProductBlueprintCategoryKind;
  path: string[];
};

export type InventoryProductIDTagDTO = {
  type: string;
};

export type ProductBlueprintModelRefDTO = {
  modelId: string;
  displayOrder: number;
};

export type ProductBlueprintPatchDTO = {
  productName: string;
  description: string;
  brandId: string;
  brandName: string;
  companyId: string;
  productBlueprintCategory: InventoryProductBlueprintCategoryDTO;
  categoryFields?: InventoryCategoryFieldValues;
  productIdTag: InventoryProductIDTagDTO;
  assigneeId: string;
  modelRefs: ProductBlueprintModelRefDTO[];
};

// =========================================================
// Inventory TokenBlueprint read model
// GET /inventory/{inventoryId}
// =========================================================

export type TokenBlueprintPatchDTO = {
  id: string;
  tokenName: string;
  symbol: string;
  brandId: string;
  brandName: string;
  companyId: string;
  description: string;
  minted: boolean;
  metadataUri: string;
  iconUrl?: string;
};

// =========================================================
// Inventory Detail Row
// GET /inventory/{inventoryId}
// =========================================================

export type InventoryDetailRowDTO = {
  modelId: string;
  kind?: InventoryProductBlueprintCategoryKind;
  modelNumber: string;
  stock: number;

  // apparel
  size?: string;
  color?: string;
  rgb?: number;

  // alcohol
  volumeValue?: number;
  volumeUnit?: string;
};

// =========================================================
// Inventory Detail DTO
// GET /inventory/{inventoryId}
// =========================================================

export type InventoryDetailDTO = {
  inventoryId: string;
  productBlueprintId: string;
  tokenBlueprintId: string;
  productBlueprintPatch: ProductBlueprintPatchDTO;
  tokenBlueprintPatch: TokenBlueprintPatchDTO;
  rows: InventoryDetailRowDTO[];
  totalStock: number;
  updatedAt?: string;
};

// =========================================================
// Inventory Detail ViewModel
// =========================================================

export type InventoryDetailViewModel = {
  inventoryId: string;
  productBlueprintId: string;
  tokenBlueprintId: string;
  productName: string;
  tokenName: string;
  headerTitle: string;
  productBlueprintCategoryName: string;
  productBlueprintCategoryCode: string;
  productBlueprintCategoryKind: InventoryProductBlueprintCategoryKind;
  categoryFields?: InventoryCategoryFieldValues;
  productBlueprintPatch: ProductBlueprintPatchDTO;
  tokenBlueprintPatch: TokenBlueprintPatchDTO;
  updatedAt?: string;
  totalStock: number;
  rows: InventoryDetailRowDTO[];
};

// =========================================================
// List Create BFF
// GET /inventory/list-create/{inventoryId}
// =========================================================

export type ListCreatePriceRowDTO = {
  modelId: string;
  kind?: InventoryProductBlueprintCategoryKind;
  displayOrder?: number;
  stock: number;

  // apparel
  size?: string;
  color?: string;
  rgb?: number;

  // alcohol
  volumeValue?: number;
  volumeUnit?: string;

  /**
   * 出品作成画面で入力された価格。
   *
   * Backend response で未設定の場合は
   * property 自体が存在しない。
   */
  price?: number;
};

export type ListCreateDTO = {
  productBrandName: string;
  productName: string;
  tokenBrandName: string;
  tokenName: string;
  priceRows: ListCreatePriceRowDTO[];
};