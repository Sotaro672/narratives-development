// frontend/console/production/src/application/detail/buildModelVariationIndex.ts

import type { ModelVariationSummary } from "./types";
import type { ModelVariationResponse } from "../../../../model/src/infrastructure/repository/modelRepositoryHTTP";

import { listModelVariationsByProductBlueprintId } from "../../infrastructure/model/modelVariationGateway";

/* ---------------------------------------------------------
 * variations → index 変換
 * --------------------------------------------------------- */
export function buildModelIndexFromVariations(
  variations: ModelVariationResponse[],
): Record<string, ModelVariationSummary> {
  const index: Record<string, ModelVariationSummary> = {};

  (Array.isArray(variations) ? variations : []).forEach((v, i) => {
    const modelId = String(v?.id ?? "").trim() || String(i);

    index[modelId] = {
      modelId,
      modelNumber: v?.modelNumber ?? "",
      size: v?.size ?? "",
      color: v?.color?.name ?? "",
      rgb: v?.color?.rgb ?? null,
      // displayOrder は variations 側に無い想定なので注入しない（必要なら別途 join）
    };
  });

  return index;
}

/* ---------------------------------------------------------
 * productBlueprintId → ModelVariation index（usecase）
 * --------------------------------------------------------- */
export async function loadModelVariationIndexByProductBlueprintId(
  productBlueprintId: string,
): Promise<Record<string, ModelVariationSummary>> {
  const id = productBlueprintId.trim();
  if (!id) return {};

  const list = await listModelVariationsByProductBlueprintId(id);
  return buildModelIndexFromVariations(list);
}
