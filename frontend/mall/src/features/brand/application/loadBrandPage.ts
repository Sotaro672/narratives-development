// frontend/amol/src/features/brand/application/loadBrandPage.ts

import {
  fetchBrandById,
  fetchBrandListItemsByIds,
} from "../api/brandApi";

import type { BrandDetail } from "../../shared/types/brand";
import type { MallListItem } from "../../shared/types/list";

export type LoadBrandPageResult = {
  brand: BrandDetail;
  listItems: MallListItem[];
};

export async function loadBrandPage(
  brandId: string,
): Promise<LoadBrandPageResult> {
  if (!brandId) {
    throw new Error("brandIdが指定されていません。");
  }

  const brand = await fetchBrandById(brandId);

  if (brand.listIds.length === 0) {
    return {
      brand,
      listItems: [],
    };
  }

  const listItems = await fetchBrandListItemsByIds(brand.listIds);

  return {
    brand,
    listItems,
  };
}