// frontend/console/production/src/presentation/viewModels/buildProductionQuantityRowVMs.ts

import type { ModelVariationSummary } from "../../application/detail/types";
import type { ProductionQuantityRowVM } from "./productionQuantityRowVM";
import type { NormalizedProductionModel } from "./normalizeProductionModels";

/**
 * 正規化済み production.models と modelIndex を join して、
 * UI が使う ProductionQuantityRowVM を生成する。
 *
 * - VM の正キーは modelId
 * - displayOrder は production.models 側（あれば）を優先し、
 *   なければ modelIndex 側の displayOrder を使う
 */
export function buildProductionQuantityRowVMs(
  models: NormalizedProductionModel[],
  modelIndex: Record<string, ModelVariationSummary>,
): ProductionQuantityRowVM[] {
  const safe = Array.isArray(models) ? models : [];

  return safe.map((m, index) => {
    const modelId = String(m.modelId ?? "").trim() || String(index);
    const meta = modelId ? modelIndex[modelId] : undefined;

    const quantity = Number.isFinite(m.quantity)
      ? Math.max(0, Math.floor(m.quantity))
      : 0;

    const modelNumber = String(m.modelNumber ?? meta?.modelNumber ?? "").trim();
    const size = String(m.size ?? meta?.size ?? "").trim();
    const color = String(m.color ?? meta?.color ?? "").trim();
    const rgb = (m.rgb ?? meta?.rgb ?? null) as any;

    const displayOrderNum =
      typeof m.displayOrder === "number" ? m.displayOrder : Number(m.displayOrder);

    const displayOrderFromModel = Number.isFinite(displayOrderNum)
      ? displayOrderNum
      : undefined;

    return {
      modelId, // ✅ VM の正キー
      modelNumber,
      size,
      color,
      rgb,
      displayOrder: displayOrderFromModel ?? meta?.displayOrder,
      quantity,
    };
  });
}
