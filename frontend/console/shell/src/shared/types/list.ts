// frontend/console/shell/src/shared/types/list.ts

/**
 * backend/internal/domain/list/entity.go のListドメインを
 * フロントエンドで参照するための正規型定義。
 *
 * Listに関する共通の型・定数は、このファイルを正規基準とする。
 */

export const LIST_STATUSES = [
  "listing",
  "suspended",
] as const;

export type ListStatus =
  (typeof LIST_STATUSES)[number];

/**
 * ListStatusの実行時判定。
 */
export function isValidListStatus(
  value: unknown,
): value is ListStatus {
  return (
    value === "listing" ||
    value === "suspended"
  );
}

/**
 * Listの配送方法。
 *
 * backend/internal/domain/list/entity.go:
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

/**
 * リスト価格行。
 *
 * backend:
 * ListPriceRow {
 *   modelId: string
 *   price: int
 * }
 */
export type ListPriceRow = {
  modelId: string;
  price: number;
};

/**
 * Listドメイン。
 *
 * readableId:
 * - 空文字の場合は未設定
 *
 * imageId:
 * - プライマリー画像のListImage ID
 * - 空文字の場合はプライマリー画像未設定
 *
 * transportationOption:
 * - yamato / sagawa / post / custom
 *
 * transportationId:
 * - custom の場合のみ TransportationFeeSetting.ID を保持
 * - yamato / sagawa / post の場合は未設定
 *
 * createdAt / updatedAt:
 * - ISO 8601形式の文字列
 */
export type List = {
  id: string;
  readableId: string;
  status: ListStatus;

  assigneeId: string;
  title: string;
  inventoryId: string;

  imageId: string;
  description: string;
  prices: ListPriceRow[];

  transportationOption: TransportationOption;
  transportationId?: string;

  createdBy: string;
  createdAt: string;

  updatedBy?: string | null;
  updatedAt?: string | null;
};

/**
 * backend/internal/domain/list/entity.go と共通の制約値。
 */
export const MAX_TITLE_LENGTH = 200;
export const MAX_DESCRIPTION_LENGTH = 2_000;

export const MIN_PRICE = 0;
export const MAX_PRICE = 10_000_000;

export const MAX_READABLE_ID_LENGTH = 64;
export const MAX_IMAGE_ID_LENGTH = 128;