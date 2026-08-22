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
 
/** 
 * Inventoryの配送方法。 
 * 
 * backend/internal/domain/inventory/entity.go: 
 * - yamato 
 * - sagawa 
 * - post 
 * - custom 
 */ 
export const TRANSPORTATION_OPTIONS = [ 
  "yamato", 
  "sagawa", 
  "post", 
  "custom", 
] as const; 
 
export type TransportationOption = 
  (typeof TRANSPORTATION_OPTIONS)[number]; 
 
/** 
 * TransportationOptionの実行時判定。 
 */ 
export function isValidTransportationOption( 
  value: unknown, 
): value is TransportationOption { 
  return ( 
    value === "yamato" || 
    value === "sagawa" || 
    value === "post" || 
    value === "custom" 
  ); 
} 
 
// ========================================================= 
// Inventory category fields 
// ========================================================= 
 
export type InventoryCategoryFieldPrimitiveValue = string | number | boolean | null; 
 
export type InventoryCategoryFieldArrayValue = 
  InventoryCategoryFieldPrimitiveValue[]; 
 
export type InventoryCategoryFieldObjectValue = 
  Record<string, InventoryCategoryFieldPrimitiveValue>; 
 
export type InventoryCategoryFieldValue = 
  | InventoryCategoryFieldPrimitiveValue 
  | InventoryCategoryFieldArrayValue 
  | InventoryCategoryFieldObjectValue; 
 
export type InventoryCategoryFieldValues = 
  Record<string, InventoryCategoryFieldValue>; 
 
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
  productBlueprintCategoryPath: string[]; 
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
// Inventory ShippingAddress read model 
// GET /inventory/{inventoryId} 
// ========================================================= 
 
export type InventoryShippingAddressDTO = { 
  id: string; 
  name: string; 
  zipCode: string; 
  state: string; 
  city: string; 
  street: string; 
  street2: string; 
}; 
 
// ========================================================= 
// Inventory Transportation read model 
// GET /inventory/{inventoryId} 
// ========================================================= 
 
export type InventoryTransportationOptionDTO = { 
  /** 
   * 配送方法。 
   * 
   * - yamato 
   * - sagawa 
   * - post 
   * - custom 
   */ 
  transportationOption: TransportationOption; 
 
  /** 
   * custom の場合のみ TransportationFeeSetting.ID を保持する。 
   * yamato / sagawa / post の場合は未設定。 
   */ 
  transportationId?: string; 
 
  /** 
   * UI表示名。 
   * 
   * 例: 
   * - ヤマト運輸 
   * - 佐川急便 
   * - 日本郵便 
   * - 自社で登録した料金設定名 
   */ 
  name: string; 
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
 
  shippingAddressId?: string; 
  shippingAddress?: InventoryShippingAddressDTO | null; 
  shippingAddressOptions: InventoryShippingAddressDTO[]; 
 
  transportationOption?: TransportationOption; 
  transportationId?: string; 
  transportationOptions: InventoryTransportationOptionDTO[]; 
 
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
  categoryFields?: InventoryCategoryFieldValues; 
  productBlueprintPatch: ProductBlueprintPatchDTO; 
  tokenBlueprintPatch: TokenBlueprintPatchDTO; 
 
  shippingAddressId: string; 
  shippingAddress: InventoryShippingAddressDTO | null; 
  shippingAddressOptions: InventoryShippingAddressDTO[]; 
 
  transportationOption: TransportationOption | ""; 
  transportationId: string; 
  transportationOptions: InventoryTransportationOptionDTO[]; 
 
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
   * Backend response で未設定の場合は property 自体が存在しない。 
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