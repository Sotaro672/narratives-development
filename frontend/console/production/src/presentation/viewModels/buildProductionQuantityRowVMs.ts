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
 *
 * ✅ 注意:
 * production.models 側の modelNumber/size/color が ""（空文字）で来るケースがあるため、
 * nullish coalescing (??) では meta にフォールバックできない。
 * 空文字の場合も meta を優先できるように `||` で吸収する。
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

    // ------------------------------------------------------
    // ✅ 空文字は meta にフォールバックさせる
    // ------------------------------------------------------
    const modelNumberFromModel = String((m as any)?.modelNumber ?? "").trim();
    const modelNumberFromMeta = String(meta?.modelNumber ?? "").trim();
    const modelNumber = modelNumberFromModel || modelNumberFromMeta;

    const sizeFromModel = String((m as any)?.size ?? "").trim();
    const sizeFromMeta = String(meta?.size ?? "").trim();
    const size = sizeFromModel || sizeFromMeta;

    const colorFromModel = String((m as any)?.color ?? "").trim();
    const colorFromMeta = String(meta?.color ?? "").trim();
    const color = colorFromModel || colorFromMeta;

    // rgb は 0（黒）が正なので、空判定は null/undefined のみでよい
    const rgb =
      (m as any)?.rgb !== undefined && (m as any)?.rgb !== null
        ? (m as any).rgb
        : meta?.rgb ?? null;

    const displayOrderNum =
      typeof (m as any).displayOrder === "number"
        ? (m as any).displayOrder
        : Number((m as any).displayOrder);

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
