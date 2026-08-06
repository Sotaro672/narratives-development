// frontend/console/shell/src/features/productBlueprint/application/productBlueprintCategoryService.ts

import {
  listProductBlueprintCategoriesApi,
  type ListProductBlueprintCategoriesParams,
} from "../infrastructure/api/productBlueprintApi";

import {
  toProductBlueprintCategorySnapshot,
  type ProductBlueprintCategorySnapshot,
} from "../domain/productBlueprintCategory";

/**
 * 商品カテゴリマスタを取得し、
 * ProductBlueprintで利用するcategory snapshotの配列へ変換する。
 */
export async function listProductBlueprintCategorySnapshots(
  params?: ListProductBlueprintCategoriesParams,
): Promise<ProductBlueprintCategorySnapshot[]> {
  const categories =
    await listProductBlueprintCategoriesApi(
      params,
    );

  return categories.map(
    toProductBlueprintCategorySnapshot,
  );
}