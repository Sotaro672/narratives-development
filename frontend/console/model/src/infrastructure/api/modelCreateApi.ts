import type { MeasurementKey } from "../../domain/entity/catalog";

// ★ HTTP リポジトリ（CreateModelVariation 用）
import {
  createModelVariations,
  type CreateModelVariationRequest,
} from "../repository/modelRepositoryHTTP";

/* =========================================================
 * ProductBlueprint Create 後に受け取る JSON 用の型
 * =======================================================*/

export type NewModelVariationMeasurements = Partial<
  Record<MeasurementKey, number | null>
>;

export type NewModelVariationPayload = {
  sizeLabel: string;
  color: string;
  rgb?: number | string;
  modelNumber: string;
  createdBy: string;
  measurements: NewModelVariationMeasurements;
};

export type ModelVariationsFromProductBlueprint = {
  productBlueprintId: string;
  variations: NewModelVariationPayload[];
};

/**
 * NewModelVariationMeasurements を backend API 用の Record<string, number> に正規化
 */
function normalizeMeasurements(
  raw: NewModelVariationMeasurements | undefined,
): Record<string, number> {
  const result: Record<string, number> = {};
  if (!raw) return result;

  for (const [key, value] of Object.entries(raw)) {
    if (value == null) continue;
    const n = Number(value);
    if (Number.isNaN(n)) continue;
    result[key] = n;
  }

  return result;
}

/**
 * PB 作成後の variations JSON → CreateModelVariationRequest に変換して
 * backend API を叩く
 */
export async function createModelVariationsFromProductBlueprint(
  payload: ModelVariationsFromProductBlueprint,
): Promise<void> {
  const requests: CreateModelVariationRequest[] = payload.variations.map(
    (v) => {
      const measurements = normalizeMeasurements(v.measurements);

      let rgbInt: number | undefined = undefined;
      if (typeof v.rgb === "string") {
        const hex = v.rgb.replace("#", "");
        if (/^[0-9a-fA-F]{6}$/.test(hex)) {
          rgbInt = parseInt(hex, 16);
        }
      } else if (typeof v.rgb === "number") {
        rgbInt = v.rgb;
      }

      return {
        productBlueprintId: payload.productBlueprintId,
        modelNumber: v.modelNumber,
        size: v.sizeLabel,
        color: v.color,
        rgb: rgbInt,
        measurements,
        // version: 1,  ← ★ 削除
      };
    },
  );

  /* ============================================
   * ★★ ここにログを追加 — POST直前の中身を全部出力
   * ============================================ */
  console.group(
    "%c[modelCreateApi] POST /models payload preview",
    "color: #0077cc; font-weight: bold;",
  );
  console.log("productBlueprintId:", payload.productBlueprintId);
  console.log("raw variations from screen:", payload.variations);
  console.log("normalized POST requests:", requests);
  console.groupEnd();

  await createModelVariations(payload.productBlueprintId, requests);
}
