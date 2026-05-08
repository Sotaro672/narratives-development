// frontend/console/production/src/application/detail/buildQuantityRows.ts
import type { ModelVariationSummary, ProductionQuantityRow } from "./types";

export function buildQuantityRowsFromModels(
  models: {
    modelId: string;
    quantity: number;
    modelNumber?: string;
    size?: string;
    color?: string;
    rgb?: number;
    displayOrder?: number;
  }[],
  modelIndex: Record<string, ModelVariationSummary>,
): ProductionQuantityRow[] {
  const safeModels = Array.isArray(models) ? models : [];

  return safeModels.map((m, index) => {
    const modelId = (m.modelId ?? "").trim() || String(index);

    const quantity = Number.isFinite(m.quantity)
      ? Math.max(0, Math.floor(m.quantity))
      : 0;

    const modelNumberFromModel = (m.modelNumber ?? "").trim();
    const sizeFromModel = (m.size ?? "").trim();
    const colorFromModel = (m.color ?? "").trim();

    const rgbFromModel =
      typeof m.rgb === "number" || typeof m.rgb === "string" ? m.rgb : undefined;

    const displayOrderNum =
      typeof m.displayOrder === "number" ? m.displayOrder : Number(m.displayOrder);
    const displayOrderFromModel = Number.isFinite(displayOrderNum)
      ? displayOrderNum
      : undefined;

    const meta = modelId ? modelIndex[modelId] : undefined;

    const row: ProductionQuantityRow = {
      modelId,
      modelNumber: modelNumberFromModel || meta?.modelNumber || "",
      size: sizeFromModel || meta?.size || "",
      color: colorFromModel || meta?.color || "",
      rgb: rgbFromModel ?? meta?.rgb ?? null,
      displayOrder: displayOrderFromModel ?? meta?.displayOrder,
      quantity,
    };

    return row;
  });
}
