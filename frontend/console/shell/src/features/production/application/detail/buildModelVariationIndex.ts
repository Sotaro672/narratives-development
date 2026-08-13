// frontend/console/shell/src/features/production/application/detail/buildModelVariationIndex.ts

import type { ModelVariationSummary } from "./types";
import {
  isAlcoholModelVariationResponse,
  isApparelModelVariationResponse,
  type ModelVariationResponse,
} from "../../../model/infrastructure/codec/modelVariationCodec";

/**
 * Codecで検証済みのModelVariationResponseを
 * modelIdをキーとしたindexへ変換する。
 */
export function buildModelIndexFromVariations(
  variations: ModelVariationResponse[],
): Record<string, ModelVariationSummary> {
  const index: Record<string, ModelVariationSummary> = {};

  for (const variation of variations) {
    const modelId = variation.id.trim();
    if (!modelId) {
      continue;
    }

    const productBlueprintId = variation.productBlueprintId.trim();
    const base: ModelVariationSummary = {
      modelId,
      productBlueprintId: productBlueprintId || undefined,
      kind: variation.kind,
      modelNumber: variation.modelNumber.trim(),
    };

    if (isApparelModelVariationResponse(variation)) {
      index[modelId] = {
        ...base,
        size: variation.size.trim(),
        color: variation.color.name.trim(),
        rgb: variation.color.rgb,
      };
      continue;
    }

    if (isAlcoholModelVariationResponse(variation)) {
      const { value: volumeValue, unit: volumeUnit } = variation.volume;

      index[modelId] = {
        ...base,
        volumeValue,
        volumeUnit,
        volume: {
          value: volumeValue,
          unit: volumeUnit,
        },
      };
    }
  }

  return index;
}