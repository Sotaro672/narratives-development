// frontend/amol/src/features/brand/api/brandApi.ts

import {
  requestJson,
} from "../../../lib/http";

import {
  brandDetailFromJson,
  brandListItemFromJson,
} from "../mappers/brandMapper";

import type {
  BrandDetail,
  BrandListItem,
} from "../types/brand";

const BRAND_BASE_PATH =
  "/mall/brands";

const LIST_BASE_PATH =
  "/mall/lists";

function normalizeRequiredId(
  value: string,
  fieldName: string,
): string {
  const normalizedValue =
    value.trim();

  if (!normalizedValue) {
    throw new Error(
      `${fieldName}が指定されていません。`,
    );
  }

  return normalizedValue;
}

function normalizeIds(
  values: string[],
): string[] {
  if (!Array.isArray(values)) {
    return [];
  }

  return Array.from(
    new Set(
      values
        .map((value) =>
          typeof value === "string"
            ? value.trim()
            : "",
        )
        .filter(Boolean),
    ),
  );
}

/**
 * ブランド詳細を取得します。
 *
 * GET /mall/brands/:brandId
 */
export async function fetchBrandById(
  brandId: string,
): Promise<BrandDetail> {
  const normalizedBrandId =
    normalizeRequiredId(
      brandId,
      "brandId",
    );

  const raw =
    await requestJson<unknown>(
      `${BRAND_BASE_PATH}/${encodeURIComponent(
        normalizedBrandId,
      )}`,
      {
        method: "GET",
        auth: "none",

        messages: {
          requestErrorMessage:
            "ブランド情報の取得に失敗しました。",
          nonJsonErrorMessage:
            "ブランド情報がJSON形式ではありません。",
          invalidJsonErrorMessage:
            "ブランド情報のJSON形式が不正です。",
        },
      },
    );

  const brand =
    brandDetailFromJson(raw);

  if (!brand.brandId) {
    throw new Error(
      "ブランド情報にbrandIdが含まれていません。",
    );
  }

  return brand;
}

/**
 * リスト詳細を取得します。
 *
 * GET /mall/lists/:listId
 */
async function fetchBrandListItemById(
  listId: string,
): Promise<BrandListItem> {
  const normalizedListId =
    normalizeRequiredId(
      listId,
      "listId",
    );

  const raw =
    await requestJson<unknown>(
      `${LIST_BASE_PATH}/${encodeURIComponent(
        normalizedListId,
      )}`,
      {
        method: "GET",
        auth: "none",
        credentials: "include",

        messages: {
          requestErrorMessage:
            "リスト情報の取得に失敗しました。",
          nonJsonErrorMessage:
            "リスト情報がJSON形式ではありません。",
          invalidJsonErrorMessage:
            "リスト情報のJSON形式が不正です。",
        },
      },
    );

  return brandListItemFromJson(
    raw,
    normalizedListId,
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
  const normalizedListIds =
    normalizeIds(listIds);

  if (
    normalizedListIds.length === 0
  ) {
    return [];
  }

  const results =
    await Promise.allSettled(
      normalizedListIds.map(
        (listId) =>
          fetchBrandListItemById(
            listId,
          ),
      ),
    );

  return results
    .filter(
      (
        result,
      ): result is PromiseFulfilledResult<BrandListItem> =>
        result.status ===
        "fulfilled",
    )
    .map(
      (result) =>
        result.value,
    )
    .filter(
      (item) =>
        item.id.trim().length > 0,
    );
}