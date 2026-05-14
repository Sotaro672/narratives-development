// frontend/console/production/src/application/detail/buildModelVariationIndex.ts

import type { ModelVariationSummary } from "./types";
import type { ModelVariationResponse } from "../../../../model/src/infrastructure/repository/modelRepositoryHTTP";

import { listModelVariationsByProductBlueprintId } from "../../infrastructure/model/modelVariationGateway";

/* ---------------------------------------------------------
 * variations → index 変換（名揺れなし・レスポンスを正として読む）
 * --------------------------------------------------------- */
export function buildModelIndexFromVariations(
  variations: ModelVariationResponse[],
): Record<string, ModelVariationSummary> {
  const index: Record<string, ModelVariationSummary> = {};

  (Array.isArray(variations) ? variations : []).forEach((variation) => {
    const v = variation as any;

    const modelId = String(v?.id ?? "").trim();
    if (!modelId) return;

    const kind = String(v?.kind ?? "").trim();
    const modelNumber = String(v?.modelNumber ?? "").trim();

    const base: ModelVariationSummary = {
      modelId,
      productBlueprintId: String(v?.productBlueprintId ?? "").trim() || undefined,
      kind,
      modelNumber,
    };

    if (kind === "apparel") {
      index[modelId] = {
        ...base,
        size: String(v?.size ?? "").trim(),
        color: String(v?.color?.name ?? "").trim(),
        rgb:
          typeof v?.color?.rgb === "number" || typeof v?.color?.rgb === "string"
            ? v.color.rgb
            : null,
      };
      return;
    }

    if (kind === "alcohol") {
      const volumeValueRaw = v?.volume?.value;
      const volumeValue =
        typeof volumeValueRaw === "number" && Number.isFinite(volumeValueRaw)
          ? volumeValueRaw
          : undefined;

      const volumeUnit = String(v?.volume?.unit ?? "").trim();

      index[modelId] = {
        ...base,
        volumeValue,
        volumeUnit,
        volume:
          volumeValue !== undefined && volumeUnit
            ? {
                value: volumeValue,
                unit: volumeUnit,
              }
            : undefined,
      };
      return;
    }

    index[modelId] = base;
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