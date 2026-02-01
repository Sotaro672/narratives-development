// frontend/console/inventory/src/infrastructure/http/listCreateRepositoryHTTP.types.ts

// ---------------------------------------------------------
// ✅ ListCreate DTO（出品作成画面）
// - backend/internal/application/query/dto/list_create_dto.go と対応
// - modelResolver 結果を PriceCard に渡すため priceRows/totalStock を追加
// ---------------------------------------------------------
export type ListCreatePriceRowDTO = {
  modelId: string;
  size: string;
  color: string;
  rgb?: number | null;
  stock: number;
  price?: number | null;
};

export type ListCreateDTO = {
  inventoryId?: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;

  productBrandName: string;
  productName: string;

  tokenBrandName: string;
  tokenName: string;

  // ✅ NEW: PriceCard 用（modelResolver の結果）
  priceRows?: ListCreatePriceRowDTO[];
  totalStock?: number;
};
