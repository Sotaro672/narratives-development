// frontend/console/model/src/infrastructure/api/modelCreateApi.ts
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
 * rgb を number(0xRRGGBB) に正規化する（DTO 正: rgb 必須）
 * - number: そのまま採用
 * - string: "#RRGGBB" / "RRGGBB" を許容
 * - それ以外: throw（省略不可のため）
 */
function normalizeRgbOrThrow(rgb: number | string | undefined): number {
  if (typeof rgb === "number" && Number.isFinite(rgb)) {
    return rgb;
  }

  if (typeof rgb === "string") {
    const hex = rgb.trim().replace(/^#/, "");
    if (/^[0-9a-fA-F]{6}$/.test(hex)) {
      return parseInt(hex, 16);
    }
  }

  throw new Error(
    `modelCreateApi: rgb が不正または未指定です (rgb=${String(rgb)})`,
  );
}

function dedupKeepOrder(xs: string[]): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const raw of xs) {
    const v = String(raw ?? "").trim();
    if (!v) continue;
    if (seen.has(v)) continue;
    seen.add(v);
    out.push(v);
  }
  return out;
}

/**
 * PB 作成後の variations JSON → CreateModelVariationRequest に変換して backend API を叩き、
 * 作成された modelId(docId) の配列を返す（append API の入力にするため）
 *
 * ※ repository(modelRepositoryHTTP.createModelVariations) の返り値は
 *    「modelIds: string[]」に統一済み、ここでは抽出ロジック不要。
 */
export async function createModelVariationsFromProductBlueprint(
  payload: ModelVariationsFromProductBlueprint,
): Promise<string[]> {
  const productBlueprintId = String(payload.productBlueprintId ?? "").trim();
  if (!productBlueprintId) {
    throw new Error("modelCreateApi: productBlueprintId が空です");
  }

  const requests: CreateModelVariationRequest[] = payload.variations.map((v) => {
    const measurements = normalizeMeasurements(v.measurements);

    // ✅ DTO 正: rgb 必須
    const rgbInt = normalizeRgbOrThrow(v.rgb);

    return {
      productBlueprintId,
      modelNumber: v.modelNumber,
      size: v.sizeLabel,
      color: v.color,
      rgb: rgbInt, // ✅ 必ず number
      measurements,
    };
  });

  // （必要なら残す。運用では env で抑制するのが推奨）
  console.group(
    "%c[modelCreateApi] POST /models payload preview",
    "color: #0077cc; font-weight: bold;",
  );
  console.log("productBlueprintId:", productBlueprintId);
  console.log("raw variations from screen:", payload.variations);
  console.log("normalized POST requests:", requests);
  console.groupEnd();

  // ✅ repo の返り値は string[]（modelIds）に統一
  const modelIds = await createModelVariations(productBlueprintId, requests);

  const out = dedupKeepOrder(modelIds);
  if (out.length === 0) {
    throw new Error("modelCreateApi: modelIds が空です");
  }

  return out;
}
