// frontend/console/productBlueprint/src/application/productBlueprintDeletedDetailService.ts

import { restoreProductBlueprintHTTP } from "../infrastructure/repository/productBlueprintRepositoryHTTP";

/**
 * 削除済み商品設計を復旧するサービス関数
 * - backend の /product-blueprints/{id}/restore を叩く thin-layer
 */
export async function restoreDeletedProductBlueprint(
  blueprintId: string,
): Promise<void> {
  const id = blueprintId?.trim();
  if (!id) {
    console.error(
      "[productBlueprintDeletedDetailService] restore called with empty id"
    );
    throw new Error("restoreDeletedProductBlueprint: blueprintId が空です");
  }

  console.log(
    `[productBlueprintDeletedDetailService] restore request start: blueprintId=${id}`,
  );

  try {
    await restoreProductBlueprintHTTP(id);
    console.log(
      `[productBlueprintDeletedDetailService] restore request success: blueprintId=${id}`,
    );
  } catch (err) {
    console.error(
      `[productBlueprintDeletedDetailService] restore request FAILED: blueprintId=${id}`,
      err,
    );
    throw err;
  }
}
