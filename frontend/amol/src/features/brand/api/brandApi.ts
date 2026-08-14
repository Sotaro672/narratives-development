// frontend/amol/src/features/brand/api/brandApi.ts

import { requestJson } from "../../../lib/http";
import type { BrandDetail, BrandListItem } from "../types/brand";

const BRAND_BASE_PATH = "/mall/brands";
const LIST_BASE_PATH = "/mall/lists";

function requireId(value: string, fieldName: string): string {
  if (!value) {
    throw new Error(`${fieldName}が指定されていません。`);
  }

  return value;
}

/**
 * ブランド詳細を取得します。
 *
 * GET /mall/brands/:brandId
 */
export async function fetchBrandById(brandId: string): Promise<BrandDetail> {
  const id = requireId(brandId, "brandId");

  return requestJson<BrandDetail>(
    `${BRAND_BASE_PATH}/${encodeURIComponent(id)}`,
    {
      method: "GET",
      auth: "none",
      messages: {
        requestErrorMessage: "ブランド情報の取得に失敗しました。",
        nonJsonErrorMessage: "ブランド情報がJSON形式ではありません。",
        invalidJsonErrorMessage: "ブランド情報のJSON形式が不正です。",
      },
    },
  );
}

/**
 * リスト詳細を取得します。
 *
 * GET /mall/lists/:listId
 */
async function fetchBrandListItemById(listId: string): Promise<BrandListItem> {
  const id = requireId(listId, "listId");

  return requestJson<BrandListItem>(
    `${LIST_BASE_PATH}/${encodeURIComponent(id)}`,
    {
      method: "GET",
      auth: "none",
      credentials: "include",
      messages: {
        requestErrorMessage: "リスト情報の取得に失敗しました。",
        nonJsonErrorMessage: "リスト情報がJSON形式ではありません。",
        invalidJsonErrorMessage: "リスト情報のJSON形式が不正です。",
      },
    },
  );
}

/**
 * ブランドに紐づくリストをまとめて取得します。
 *
 * 一部の取得に失敗しても、正常に取得できたリストは返します。
 */
export async function fetchBrandListItemsByIds(
  listIds: string[],
): Promise<BrandListItem[]> {
  if (listIds.length === 0) {
    return [];
  }

  const results = await Promise.allSettled(
    listIds.map((listId) => fetchBrandListItemById(listId)),
  );

  return results
    .filter(
      (result): result is PromiseFulfilledResult<BrandListItem> =>
        result.status === "fulfilled",
    )
    .map((result) => result.value);
}