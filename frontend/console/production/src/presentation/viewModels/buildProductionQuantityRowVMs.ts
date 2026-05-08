// frontend/console/production/src/presentation/viewModels/buildProductionQuantityRowVMs.ts

import type { ModelVariationSummary } from "../../application/detail/types";
import type { ProductionQuantityRowVM } from "./productionQuantityRowVM";

/**
 * backend の production.Models（[{ ModelID, Quantity, (optional) DisplayOrder... }]）と
 * modelIndex（variations）を join して、UI が使う ProductionQuantityRowVM を生成する。
 *
 * - 正: modelId は production.Models[].ModelID
 * - 表示メタ（modelNumber/size/color/rgb）は modelIndex を正として使う
 * - 表示順ロジックは displayOrder のみを利用（無ければ undefined のまま）
 *
 * NOTE:
 * ProductionQuantityRowVM の rgb は number | undefined なので、
 * null/string はここで落として undefined に寄せる。
 */
export function buildProductionQuantityRowVMs(
  models: any[],
  modelIndex: Record<string, ModelVariationSummary>,
): ProductionQuantityRowVM[] {
  const safe = Array.isArray(models) ? models : [];

  return safe.map((m, index) => {
    const modelId = String(m?.ModelID ?? "").trim() || String(index);
    const meta = modelId ? modelIndex[modelId] : undefined;

    const qNum = Number(m?.Quantity);
    const quantity = Number.isFinite(qNum) ? Math.max(0, Math.floor(qNum)) : 0;

    // ✅ 表示メタは variations を正にする（production 側は持たない想定）
    const modelNumber = String(meta?.modelNumber ?? "").trim();
    const size = String(meta?.size ?? "").trim();
    const color = String(meta?.color ?? "").trim();

    // ✅ rgb: ProductionQuantityRowVM は number | undefined
    // - meta.rgb が number のときだけ採用
    const rgbRaw = meta?.rgb;
    const rgb = typeof rgbRaw === "number" ? rgbRaw : undefined;

    // ✅ displayOrder のみを表示順ロジックとして扱う
    const displayOrderNum =
      typeof m?.DisplayOrder === "number" ? m.DisplayOrder : Number(m?.DisplayOrder);
    const displayOrder = Number.isFinite(displayOrderNum) ? displayOrderNum : undefined;

    const vm: ProductionQuantityRowVM = {
      modelId,
      modelNumber,
      size,
      color,
      rgb,
      displayOrder,
      quantity,
    };

    return vm;
  });
}