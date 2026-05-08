// frontend/console/productBlueprint/src/infrastructure/api/productBlueprintApi.ts

import type { ItemType, Fit } from "../../domain/entity/catalog";
import type { ProductIDTag } from "../../../../productBlueprint/src/domain/entity/productBlueprint";
import type { MeasurementKey } from "../../../../model/src/domain/entity/catalog";
import type { ModelNumber } from "../../../../model/src/application/modelCreateService";

import { safeDateLabelJa } from "../../../../shell/src/shared/util/dateJa";

import {
  createProductBlueprintHTTP,
  appendModelRefsHTTP,
} from "../repository/productBlueprintRepositoryHTTP";

import {
  createModelVariations,
  type CreateModelVariationRequest,
} from "../../../../model/src/infrastructure/repository/modelRepositoryHTTP";

// 日付表示は shared util に統一
export const formatProductBlueprintDate = (iso?: string | null): string =>
  safeDateLabelJa(iso, "");

// 一覧表示用のUI行モデル（API が返す形）
export type ProductBlueprintListRow = {
  id: string;
  productName: string;
  brandLabel: string;
  assigneeLabel: string;
  tagLabel: string;
  createdAt: string;
  lastModifiedAt: string;
};

// ProductBlueprint 作成用：サイズ行モデル
export type ProductBlueprintSizeRow = {
  id: string;
  sizeLabel: string;

  // tops
  length?: number;
  width?: number;
  chest?: number;
  shoulder?: number;
  sleeveLength?: number;

  // bottoms
  waist?: number;
  hip?: number;
  rise?: number;
  inseam?: number;
  thigh?: number;
  hemWidth?: number;
};

// 詳細画面用：モデルナンバー行モデル
export type ModelNumberRow = {
  size: string;
  color: string;
  code: string;
};

/* =========================================================
 * 作成系 API（createProductBlueprint + variations 作成 + modelRefs append）
 * =======================================================*/

export type CreateProductBlueprintParams = {
  productName: string;
  brandId: string;
  itemType: ItemType;
  fit: Fit;
  material: string;
  weight: number;
  qualityAssurance: string[];

  productIdTag: ProductIDTag;

  companyId: string;
  assigneeId?: string;
  createdBy?: string;

  colors: string[];
  sizes: ProductBlueprintSizeRow[];
  modelNumbers: ModelNumber[];
  colorRgbMap?: Record<string, string>;
};

export type ProductBlueprintResponse = {
  id?: string;
  productBlueprintId?: string;
  [key: string]: unknown;
};

/**
 * measurements 部分の型
 */
export type NewModelVariationMeasurements = Partial<
  Record<MeasurementKey, number | null>
>;

/**
 * ModelVariation 用 Payload
 */
export type NewModelVariationPayload = {
  sizeLabel: string;
  color: string;
  rgb?: number;
  modelNumber: string;
  createdBy: string;
  measurements: NewModelVariationMeasurements;
};

/**
 * ProductBlueprint の ID 抽出
 * backend 正: id を第一優先、互換で productBlueprintId も許容
 */
function extractProductBlueprintId(json: unknown): string {
  const anyJson = json as Record<string, unknown>;
  const raw = anyJson?.id ?? anyJson?.productBlueprintId;
  return typeof raw === "string" ? raw : "";
}

function dedupKeepOrder(xs: string[]): string[] {
  const seen = new Set<string>();
  const out: string[] = [];

  for (const raw of xs ?? []) {
    const v = String(raw ?? "");
    if (!v) continue;
    if (seen.has(v)) continue;
    seen.add(v);
    out.push(v);
  }

  return out;
}

function toCreateModelVariationRequests(args: {
  productBlueprintId: string;
  variations: NewModelVariationPayload[];
}): CreateModelVariationRequest[] {
  const { productBlueprintId, variations } = args;

  return variations.map((v) => ({
    productBlueprintId,
    modelNumber: String(v.modelNumber ?? ""),
    size: String(v.sizeLabel ?? ""),
    color: String(v.color ?? ""),
    rgb: typeof v.rgb === "number" ? v.rgb : 0,
    measurements: Object.fromEntries(
      Object.entries(v.measurements ?? {}).filter(([, value]) => value != null),
    ) as Record<string, number>,
  }));
}

/**
 * ProductBlueprint + ModelVariations をまとめて作成し、
 * さらに modelRefs（modelIds）を append する API 呼び出し。
 *
 * - ProductBlueprint 自体の作成は createProductBlueprintHTTP に委譲
 * - 生成された productBlueprintId を使って variations を作成
 * - variations 作成で得られた modelIds を順序付きで append
 * - append の返り値（detail）を最終結果として返す
 */
export async function createProductBlueprintApi(
  params: CreateProductBlueprintParams,
  variations: NewModelVariationPayload[],
): Promise<ProductBlueprintResponse> {
  const created = await createProductBlueprintHTTP(params);
  const productBlueprintId = extractProductBlueprintId(created);

  if (!productBlueprintId) {
    return created as ProductBlueprintResponse;
  }

  if (variations.length === 0) {
    return created as ProductBlueprintResponse;
  }

  const requests = toCreateModelVariationRequests({
    productBlueprintId,
    variations,
  });

  const modelIds = await createModelVariations(productBlueprintId, requests);
  const cleaned = dedupKeepOrder(modelIds);

  if (cleaned.length === 0) {
    throw new Error("createProductBlueprintApi: modelIds が空です");
  }

  const detail = await appendModelRefsHTTP(productBlueprintId, cleaned);
  return detail as ProductBlueprintResponse;
}