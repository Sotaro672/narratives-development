// frontend/console/production/src/application/detail/buildQuantityRows.ts
import type { ModelVariationSummary, ProductionQuantityRow } from "./types";

/* ---------------------------------------------------------
 * モデル別 生産数行を生成（pure function）
 * --------------------------------------------------------- */
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

  const rows: ProductionQuantityRow[] = safeModels.map((m, index) => {
    // dto/detail.go を正: modelId がキー
    const modelId = (m.modelId ?? "").trim() || String(index);

    // dto を正: quantity は number 前提だが、UI 安全のため clamp のみ実施
    const quantity = Number.isFinite(m.quantity)
      ? Math.max(0, Math.floor(m.quantity))
      : 0;

    // dto を正: 詳細 DTO が modelNumber/color/size/rgb/displayOrder を返す場合はそれを優先
    const modelNumberFromModel = (m.modelNumber ?? "").trim();
    const sizeFromModel = (m.size ?? "").trim();
    const colorFromModel = (m.color ?? "").trim();
    const rgbFromModel = typeof m.rgb === "number" ? m.rgb : undefined;

    const displayOrderFromModel =
      Number.isFinite(m.displayOrder) ? (m.displayOrder as number) : undefined;

    // 足りない分は modelIndex で補完する（キーは modelId）
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

  return rows;
}
