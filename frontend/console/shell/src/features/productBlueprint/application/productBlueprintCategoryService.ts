// frontend/console/shell/src/features/productBlueprint/application/productBlueprintCategoryService.ts

import {
  listProductBlueprintCategoryTreeApi,
} from "../infrastructure/api/productBlueprintApi";

import {
  toProductBlueprintCategorySnapshot,
  type ProductBlueprintCategorySnapshot,
} from "../domain/productBlueprintCategory";

/**
 * 商品カテゴリツリーを取得し、
 * ProductBlueprintで利用するcategory snapshotの配列へ変換する。
 */
export async function listProductBlueprintCategorySnapshots():
  Promise<ProductBlueprintCategorySnapshot[]> {
  const categories =
    await listProductBlueprintCategoryTreeApi();

  if (
    categories.length === 0
  ) {
    throw new Error(
      "商品カテゴリマスタが0件です。",
    );
  }

  return categories.map(
    toProductBlueprintCategorySnapshot,
  );
}