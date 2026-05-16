// frontend/console/inventory/src/infrastructure/http/listCreateRepositoryHTTP.types.ts

// ---------------------------------------------------------
// ListCreate DTO（出品作成画面）
// - backend/internal/application/query/dto/list_create_dto.go と対応
// - GET /inventory/list-create/{inventoryId} の response を唯一の正とする
// - priceRows は backend 側で productCategory / model kind に応じた完成形になっている
// ---------------------------------------------------------

export type ListCreateModelKind = "apparel" | "alcohol" | string;

export type ListCreateModelRefDTO = {
  modelId: string;
  displayOrder?: number | null;
};

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
   * modelNumber。
   *
   * apparel / alcohol 共通で表示に使う。
   * 例:
   * - apparel: "M-001"
   * - alcohol: "s", "m"
   */
  modelNumber?: string | null;

  /**
   * modelRefs.displayOrder に対応。
   * backend の productBlueprintPatch.ModelRefs.DisplayOrder を詰めて渡す。
   */
  displayOrder?: number | null;

  stock: number;

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

  listImageUrl?: string | null;

  modelRefs?: ListCreateModelRefDTO[];

  /**
   * PriceCard 用。
   * GET /inventory/list-create/{inventoryId} の priceRows を正とする。
   */
  priceRows?: ListCreatePriceRowDTO[];

  totalStock?: number;
  priceNote?: string | null;
  currencyJpy?: boolean;
};