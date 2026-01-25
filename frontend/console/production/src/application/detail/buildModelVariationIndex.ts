//frontend\console\production\src\application\detail\buildModelVariationIndex.ts
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

  variations.forEach((v) => {
    index[v.id] = {
      id: v.id,
      modelNumber: v.modelNumber,
      size: v.size,
      color: v.color?.name ?? "",
      rgb: v.color?.rgb ?? null,
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
