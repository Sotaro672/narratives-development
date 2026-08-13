// frontend/console/shell/src/features/production/presentation/viewModels/buildProductionQuantityRowVMs.ts

import type { ModelVariationSummary } from "../../application/detail/types";
import type { ProductionQuantityRowVM } from "./productionQuantityRowVM";

type ProductionModelInput = {
  ModelID: string;
  Quantity: number;
  DisplayOrder?: number;
};

/**
 * Production Create 用。
 *
 * production models と model variation index を結合して
 * ProductionQuantityRowVM を生成する。
 *
 * Production Detail では backend BFF の rows を正とするため使用しない。
 */
export function buildProductionQuantityRowVMs(
  models: ProductionModelInput[],
  modelIndex: Record<string, ModelVariationSummary>,
): ProductionQuantityRowVM[] {
  return models.map((model) => {
    const modelId = model.ModelID;
    const meta = modelIndex[modelId];

    const size = meta?.size;
    const color = meta?.color;

    const volumeValue = meta?.volumeValue ?? meta?.volume?.value;
    const volumeUnit = meta?.volumeUnit ?? meta?.volume?.unit;

    const variationLabel =
      meta?.kind === "alcohol"
        ? volumeValue !== undefined && volumeUnit
          ? `${volumeValue}${volumeUnit}`
          : undefined
        : size || color
          ? [size, color].filter(Boolean).join(" / ")
          : undefined;

    return {
      modelId,
      kind: meta?.kind,
      modelNumber: meta?.modelNumber ?? "",
      size,
      color,
      rgb: typeof meta?.rgb === "number" ? meta.rgb : undefined,
      volumeValue,
      volumeUnit,
      variationLabel,
      displayOrder: model.DisplayOrder,
      quantity: model.Quantity,
    };
  });
}