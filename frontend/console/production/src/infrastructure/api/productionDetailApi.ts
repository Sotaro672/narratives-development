// ======================================================================
// Infrastructure API for Production Detail
//   - Production 本体の取得
//   - ProductBlueprint 詳細 + ModelVariations の取得
// ======================================================================

import type { Production } from "../../application/productionCreateService";

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
    console.log("[productionDetailApi] fetchProductionById:", {
      productionId,
      data,
    });
    return data;
  } catch (e) {
    console.error("[productionDetailApi] failed to fetchProductionById:", {
      productionId,
      error: e,
    });
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
    console.warn(
      "[productionDetailApi] fetchProductBlueprintDetailForProduction: empty productBlueprintId",
    );
    return { detail: null, models: [] };
  }

  try {
    console.log(
      "[productionDetailApi] fetchProductBlueprintDetailForProduction request:",
      { productBlueprintId: pbId },
    );

    const [detail, models] = await Promise.all([
      getProductBlueprintDetail(pbId),
      listModelVariationsByProductBlueprintId(pbId),
    ]);

    console.log(
      "[productionDetailApi] fetchProductBlueprintDetailForProduction response:",
      { detail, models },
    );

    return {
      detail: detail ?? null,
      models: Array.isArray(models) ? models : [],
    };
  } catch (e) {
    console.error(
      "[productionDetailApi] failed to fetchProductBlueprintDetailForProduction:",
      { productBlueprintId: pbId, error: e },
    );
    return { detail: null, models: [] };
  }
}
