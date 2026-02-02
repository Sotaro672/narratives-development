//frontend\console\list\src\infrastructure\http\list\types.ts
/**
 * =============
 * Types
 * =============
 * ※ backend の List エンティティに完全一致しなくてもOK（必要な分だけ）
 */

export type CreateListInput = {
  // backend が docId を要求する場合に備えて（未指定なら inventoryId を採用）
  id?: string;

  // ルート（inventory/list/create から作成する想定）
  // ✅ 方針A: inventoryId は「pb__tb」をそのまま通す（絶対に split しない）
  inventoryId?: string;

  // UI 入力
  title: string;
  description: string;

  // PriceCard の rows（UI 側は保持していてOK / backend には modelId + price のみ送る）
  priceRows?: Array<{
    modelId?: string;
    price: number | null;

    // UI 用（backend には送らない）
    size: string;
    color: string;
    stock: number;
    rgb?: number | null;
  }>;

  // 画面の「出品｜保留」（※ create payload には送らない）
  decision?: "list" | "hold";

  // 担当者など（必要に応じて）
  assigneeId?: string;

  // 作成者など（バックエンドで auth から取るなら省略可）
  createdBy?: string;
};

/**
 * ✅ NEW: Update 用
 * - listDetail 側の PriceRow は id = modelId なので、row.id も受ける
 * - backend に送るのは modelId + price だけ（DisallowUnknownFields 対策）
 */
export type UpdateListInput = {
  listId: string;

  title?: string;
  description?: string;

  // detail 側の priceRows（id=modelId）
  priceRows?: Array<{
    // create系: modelId
    modelId?: string;

    // detail系: id (= modelId)
    id?: string;

    price: number | null;

    // UI 用（backend には送らない）
    size?: string;
    color?: string;
    stock?: number;
    rgb?: number | null;
  }>;

  // UI の "list" | "hold" を backend の status に変換して送る（必要な場合のみ）
  decision?: "list" | "hold";

  assigneeId?: string;

  // バックエンドで auth から取るなら省略可
  updatedBy?: string;
};

export type ListDTO = Record<string, any>;
export type ListAggregateDTO = Record<string, any>;

/**
 * ✅ ListImage DTO（backend 依存を避けるため Record<any> を基本にする）
 * - ただし bucket/objectPath/publicUrl 等、URL 生成に必要なキーは代表的な候補を吸収する
 */
export type ListImageDTO = Record<string, any>;

/**
 * ✅ Signed URL 発行の戻り（Policy A）
 *
 * IMPORTANT:
 * - backend の返却は uploadUrl/publicUrl/objectPath/id... になりがち
 * - UI/呼び出し側は signedUrl を使いたいケースがあるため、この DTO では signedUrl を正とし、
 *   issueListImageSignedUrlHTTP 内で uploadUrl → signedUrl に正規化する
 */
export type SignedListImageUploadDTO = {
  id?: string;

  bucket?: string;

  // ✅ 必須（= GCS 上の objectPath / key）
  objectPath: string;

  // ✅ PUT 先（= signed URL）
  signedUrl: string;

  // ✅ 表示用（public access を想定）
  publicUrl?: string;

  // optional metadata
  expiresAt?: string;
  contentType?: string;
  size?: number;
  displayOrder?: number;
  fileName?: string;
};

/**
 * ✅ ListDetail DTO（型ガイド用）
 */
export type ListDetailDTO = ListDTO & {
  createdByName?: string;
  updatedByName?: string;

  createdBy?: string;
  updatedBy?: string;

  createdAt?: string;
  updatedAt?: string;

  // ✅ listImage bucket からの画像URL
  imageId?: string;
  imageUrls?: string[];
};
