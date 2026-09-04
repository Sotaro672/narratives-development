// frontend/amol/src/features/brand/api/brandApi.ts

import { requestJson } from "../../../lib/http";
import type { BrandDetail } from "../../shared/types/brand";
import type { MallListItem } from "../../shared/types/list";

const BRAND_BASE_PATH = "/mall/brands";
const LIST_BASE_PATH = "/mall/lists";

function requireId(value: string, fieldName: string): string {
  if (!value) {
    throw new Error(`${fieldName}が指定されていません。`);
  }

  return value;
}

export async function fetchBrandById(
  brandId: string,
): Promise<BrandDetail> {
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

async function fetchBrandListItemById(
  listId: string,
): Promise<MallListItem> {
  const id = requireId(listId, "listId");

  return requestJson<MallListItem>(
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

export async function fetchBrandListItemsByIds(
  listIds: string[],
): Promise<MallListItem[]> {
  if (listIds.length === 0) {
    return [];
  }

  const results = await Promise.allSettled(
    listIds.map((listId) => fetchBrandListItemById(listId)),
  );

  return results
    .filter(
      (result): result is PromiseFulfilledResult<MallListItem> =>
        result.status === "fulfilled",
    )
    .map((result) => result.value);
}