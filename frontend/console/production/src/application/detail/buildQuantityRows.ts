//frontend\console\production\src\application\detail\buildQuantityRows.ts
import type { ModelVariationSummary, ProductionQuantityRow } from "./types";

/* ---------------------------------------------------------
 * モデル別 生産数行を生成（pure function）
 * --------------------------------------------------------- */
export function buildQuantityRowsFromModels(
  models: any[],
  modelIndex: Record<string, ModelVariationSummary>,
): ProductionQuantityRow[] {
  const safeModels = Array.isArray(models) ? models : [];

  const rows: ProductionQuantityRow[] = safeModels.map((m: any, index) => {
    // ✅ camelCase / PascalCase 両対応で modelId を解決する
    const rawModelId =
      m.modelId ??
      m.ModelID ??
      m.modelID ??
      m.model_id ??
      m.id ??
      m.ID ??
      null;

    // modelId が取れない場合でも UI が壊れないように index fallback
    const id = rawModelId ? String(rawModelId) : String(index);

    // ✅ camelCase / PascalCase 両対応で quantity を解決する
    const quantityRaw = m.quantity ?? m.Quantity ?? 0;

    const quantity = Number.isFinite(Number(quantityRaw))
      ? Math.max(0, Math.floor(Number(quantityRaw)))
      : 0;

    // ✅ backend の詳細 DTO が modelNumber/color/size/rgb を返している場合はそれを優先する
    const modelNumberFromModel = m.modelNumber ?? m.ModelNumber ?? "";
    const sizeFromModel = m.size ?? m.Size ?? "";
    const colorFromModel = m.color ?? m.Color ?? "";
    const rgbFromModel = m.rgb ?? m.RGB ?? null;

    // ✅ 取れなかった分は modelIndex で補完する
    const meta = id ? modelIndex[id] : undefined;

    const row: ProductionQuantityRow = {
      id,
      modelNumber: modelNumberFromModel || meta?.modelNumber || "",
      size: sizeFromModel || meta?.size || "",
      color: colorFromModel || meta?.color || "",
      rgb: rgbFromModel ?? meta?.rgb ?? null,
      quantity,
    };

    return row;
  });

  return rows;
}
