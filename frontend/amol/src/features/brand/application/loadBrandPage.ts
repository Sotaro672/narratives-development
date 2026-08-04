// frontend/amol/src/features/brand/application/loadBrandPage.ts

import {
  fetchBrandById,
  fetchBrandListItemsByIds,
} from "../api/brandApi";

import type {
  BrandDetail,
  BrandListItem,
} from "../types/brand";

export type LoadBrandPageResult = {
  brand: BrandDetail;
  listItems: BrandListItem[];
};

export async function loadBrandPage(
  brandId: string,
): Promise<LoadBrandPageResult> {
  const normalizedBrandId =
    brandId.trim();

  if (!normalizedBrandId) {
    throw new Error(
      "brandIdが指定されていません。",
    );
  }

  const brand =
    await fetchBrandById(
      normalizedBrandId,
    );

  if (!brand.brandId) {
    throw new Error(
      "ブランド情報が不正です。",
    );
  }

  const listIds =
    Array.from(
      new Set(
        brand.listIds
          .map((listId) =>
            listId.trim(),
          )
          .filter(Boolean),
      ),
    );

  if (listIds.length === 0) {
    return {
      brand,
      listItems: [],
    };
  }

  const listItems =
    await fetchBrandListItemsByIds(
      listIds,
    );

  return {
    brand,
    listItems,
  };
}