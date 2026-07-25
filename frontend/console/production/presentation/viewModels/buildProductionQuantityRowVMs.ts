// frontend/console/production/src/presentation/viewModels/buildProductionQuantityRowVMs.ts

import type { ModelVariationSummary } from "../../application/detail/types";
import type { ProductionQuantityRowVM } from "./productionQuantityRowVM";

/**
 * backend の production.Models（[{ ModelID, Quantity, (optional) DisplayOrder... }]）と
 * modelIndex（variations）を join して、UI が使う ProductionQuantityRowVM を生成する。
 *
 * - 正: modelId は production.Models[].ModelID
 * - 表示メタは modelIndex を正として使う
 * - apparel: modelNumber / size / color / rgb
 * - alcohol: modelNumber / volumeValue / volumeUnit
 * - 表示順ロジックは displayOrder のみを利用（無ければ undefined）
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

    const kind = String(meta?.kind ?? "").trim() || undefined;

    // 表示メタは variations を正にする（production 側は持たない想定）
    const modelNumber = String(meta?.modelNumber ?? "").trim();

    // apparel 用
    const size = String(meta?.size ?? "").trim();
    const color = String(meta?.color ?? "").trim();

    // rgb: ProductionQuantityRowVM は number | undefined
    // meta.rgb が number のときだけ採用
    const rgbRaw = meta?.rgb;
    const rgb = typeof rgbRaw === "number" ? rgbRaw : undefined;

    // alcohol 用
    const volumeValueFromFlat =
      typeof meta?.volumeValue === "number" && Number.isFinite(meta.volumeValue)
        ? meta.volumeValue
        : undefined;

    const volumeValueFromNested =
      typeof meta?.volume?.value === "number" &&
      Number.isFinite(meta.volume.value)
        ? meta.volume.value
        : undefined;

    const volumeValue = volumeValueFromFlat ?? volumeValueFromNested;

    const volumeUnit =
      String(meta?.volumeUnit ?? "").trim() ||
      String(meta?.volume?.unit ?? "").trim() ||
      undefined;

    const apparelVariationLabel = [size, color].filter(Boolean).join(" / ");

    const alcoholVariationLabel =
      typeof volumeValue === "number" && Number.isFinite(volumeValue) && volumeUnit
        ? `${volumeValue}${volumeUnit}`
        : "";

    const variationLabel =
      kind === "alcohol"
        ? alcoholVariationLabel
        : kind === "apparel"
          ? apparelVariationLabel
          : alcoholVariationLabel || apparelVariationLabel;

    // displayOrder のみを表示順ロジックとして扱う
    const displayOrderNum =
      typeof m?.DisplayOrder === "number" ? m.DisplayOrder : Number(m?.DisplayOrder);

    const displayOrder = Number.isFinite(displayOrderNum)
      ? displayOrderNum
      : undefined;

    const vm: ProductionQuantityRowVM = {
      modelId,
      kind,

      modelNumber,

      size,
      color,
      rgb,

      volumeValue,
      volumeUnit,

      variationLabel,

      displayOrder,
      quantity,
    };

    return vm;
  });
}