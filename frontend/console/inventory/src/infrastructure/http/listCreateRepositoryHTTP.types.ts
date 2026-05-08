// frontend/console/inventory/src/infrastructure/http/listCreateRepositoryHTTP.types.ts

// ---------------------------------------------------------
// ✅ ListCreate DTO（出品作成画面）
// - backend/internal/application/query/dto/list_create_dto.go と対応
// - modelResolver 結果を PriceCard に渡すため priceRows/totalStock を追加
// ---------------------------------------------------------
export type ListCreatePriceRowDTO = {
  modelId: string;

  /**
   * ✅ modelRefs.displayOrder に対応（並び順はこれの昇順のみ）
   * - backend の productBlueprintPatch.ModelRefs.DisplayOrder を詰めて渡す想定
   * - 未設定は null を保持（＝並び順なし）
   */
  displayOrder?: number | null;

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

  // ✅ PriceCard 用（modelResolver の結果）
  priceRows?: ListCreatePriceRowDTO[];
  totalStock?: number;
};
