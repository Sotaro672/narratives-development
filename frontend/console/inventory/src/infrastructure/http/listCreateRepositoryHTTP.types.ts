// frontend/console/inventory/src/infrastructure/http/listCreateRepositoryHTTP.types.ts

// ---------------------------------------------------------
// ✅ ListCreate DTO（出品作成画面）
// - backend/internal/application/query/dto/list_create_dto.go と対応
// - modelResolver 結果を PriceCard に渡すため priceRows/totalStock を追加
// ---------------------------------------------------------

export type ListCreateModelKind = "apparel" | "alcohol" | string;

export type ListCreatePriceRowDTO = {
  modelId: string;

  /**
   * model domain の variation.kind 由来。
   *
   * - apparel: size / color / rgb を表示
   * - alcohol: volumeValue / volumeUnit を表示
   */
  kind?: ListCreateModelKind | null;

  /**
   * ✅ modelRefs.displayOrder に対応（並び順はこれの昇順のみ）
   * - backend の productBlueprintPatch.ModelRefs.DisplayOrder を詰めて渡す想定
   * - 未設定は null を保持（＝並び順なし）
   */
  displayOrder?: number | null;

  /**
   * apparel category 用。
   */
  size?: string | null;

  /**
   * apparel category 用。
   */
  color?: string | null;

  /**
   * apparel category 用。
   */
  rgb?: number | null;

  /**
   * alcohol category 用。
   *
   * 例: 720, 1000
   */
  volumeValue?: number | null;

  /**
   * alcohol category 用。
   *
   * 例: "ml", "L"
   */
  volumeUnit?: string | null;

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

  /**
   * ProductBlueprintCategory.code。
   *
   * PriceCard の category 表示分岐で使用する。
   * 例:
   * - "apparel.tops"
   * - "alcohol.sake"
   */
  productBlueprintCategory?: string | null;

  /**
   * ProductBlueprintCategory.kind。
   *
   * 必須ではないが、呼び出し側で code がない場合の補助情報として使用できる。
   * 例:
   * - "apparel"
   * - "alcohol"
   */
  productBlueprintCategoryKind?: string | null;

  // ✅ PriceCard 用（modelResolver の結果）
  priceRows?: ListCreatePriceRowDTO[];
  totalStock?: number;
};