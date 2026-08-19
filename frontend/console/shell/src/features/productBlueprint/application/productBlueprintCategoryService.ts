// frontend/console/shell/src/features/productBlueprint/application/productBlueprintCategoryService.ts

import {
  listProductBlueprintCategoryTreeApi,
} from "../infrastructure/api/productBlueprintApi";

import type {
  ProductBlueprintCategoryPath,
} from "../domain/productBlueprintCategory";

/**
 * 商品カテゴリツリーを取得し、
 * ProductBlueprintで利用するproductBlueprintCategoryPathの配列へ変換する。
 */
export async function listProductBlueprintCategoryPaths():
  Promise<ProductBlueprintCategoryPath[]> {
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
    (category) => [
      ...category.productBlueprintCategoryPath,
    ],
  );
}