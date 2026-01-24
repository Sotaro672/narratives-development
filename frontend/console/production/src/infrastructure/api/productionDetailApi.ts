// ======================================================================
// Infrastructure API for Production Detail
//   - Production 本体の取得
//   - ProductBlueprint 詳細 + ModelVariations の取得
//   - ProductBlueprint の printed: notYet → printed 更新
// ======================================================================

import type { Production } from "../../application/create/ProductionCreateTypes";

import {
  getProductBlueprintDetail,
  listModelVariationsByProductBlueprintId,
} from "../../../../productBlueprint/src/application/productBlueprintDetailService";
import type { ModelVariationResponse } from "../../../../productBlueprint/src/application/productBlueprintDetailService";

import { ProductionRepositoryHTTP } from "../http/productionRepositoryHTTP";

// 型を必要ならアプリ層に再エクスポート
export type { Production, ModelVariationResponse };

// ======================================================================
// Production 詳細 API
// ======================================================================
export async function fetchProductionById(
  productionId: string,
): Promise<Production | null> {
  if (!productionId) return null;

  const repo = new ProductionRepositoryHTTP();

  try {
    const data = (await repo.getById(productionId)) as Production;
    return data;
  } catch (e) {
    return null;
  }
}

// ======================================================================
// ProductBlueprint 詳細 + ModelVariations API
// ======================================================================
export async function fetchProductBlueprintDetailForProduction(
  productBlueprintId: string | null | undefined,
): Promise<{
  detail: any | null;
  models: ModelVariationResponse[];
}> {
  const pbId = (productBlueprintId ?? "").trim();
  if (!pbId) {
    return { detail: null, models: [] };
  }

  try {
    const [detail, models] = await Promise.all([
      getProductBlueprintDetail(pbId),
      listModelVariationsByProductBlueprintId(pbId),
    ]);

    return {
      detail: detail ?? null,
      models: Array.isArray(models) ? models : [],
    };
  } catch (e) {
    return { detail: null, models: [] };
  }
}

// ======================================================================
// ProductBlueprint printed 更新 API
//   - printed: notYet → printed
//   - backend: POST /product-blueprints/{id}/mark-printed
// ======================================================================
export async function markProductBlueprintAsPrinted(
  productBlueprintId: string,
): Promise<void> {
  const id = productBlueprintId.trim();
  if (!id) return;

  const repo = new ProductionRepositoryHTTP();
  await repo.markProductBlueprintPrinted(id);
}
