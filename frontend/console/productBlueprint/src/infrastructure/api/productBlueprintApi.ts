// frontend/console/productBlueprint/src/infrastructure/api/productBlueprintApi.ts

import type {
  ApparelMeasurements,
  ApparelModelNumberRow,
  ApparelModelVariationPayload,
  ApparelSizeRow,
  Fit,
} from "../../domain/entity/apparel";

import type {
  CategoryFieldValues,
  ProductBlueprintCategorySnapshot,
} from "../../domain/entity/productBlueprintCategory";

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

import type { ProductBlueprintDetailResponse } from "./productBlueprintDetailApi";

// 日付表示は shared util に統一
export const formatProductBlueprintDate = (iso?: string | null): string =>
  safeDateLabelJa(iso, "");

// ------------------------------------------------------
// Product ID Tag
// ------------------------------------------------------

export type ProductIDTagType = "qr" | "nfc";

export type ProductIDTag = {
  type: ProductIDTagType;
};

// ------------------------------------------------------
// 一覧表示用のUI行モデル
// ------------------------------------------------------

export type ProductBlueprintListRow = {
  id: string;
  productName: string;
  brandLabel: string;
  assigneeLabel: string;
  tagLabel: string;
  createdAt: string;
  lastModifiedAt: string;
};

// ProductBlueprint 作成用：apparel サイズ行モデル
export type ProductBlueprintSizeRow = ApparelSizeRow;

// 詳細画面用：apparel モデルナンバー行モデル
export type ModelNumberRow = ApparelModelNumberRow;

/* =========================================================
 * 作成系 API（createProductBlueprint + variations 作成 + modelRefs append）
 * =======================================================*/

export type CreateProductBlueprintParams = {
  productName: string;
  brandId: string;

  productBlueprintCategoryId: string;
  productBlueprintCategory: ProductBlueprintCategorySnapshot;

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

  categoryFields?: CategoryFieldValues | null;
};

export type ProductBlueprintResponse = ProductBlueprintDetailResponse;

export type NewModelVariationPayload = ApparelModelVariationPayload;

function assertProductBlueprintCategory(params: CreateProductBlueprintParams): void {
  if (!params.productBlueprintCategoryId?.trim()) {
    throw new Error("createProductBlueprintApi: productBlueprintCategoryId が空です");
  }

  if (!params.productBlueprintCategory?.id?.trim()) {
    throw new Error("createProductBlueprintApi: productBlueprintCategory.id が空です");
  }

  if (params.productBlueprintCategoryId !== params.productBlueprintCategory.id) {
    throw new Error(
      "createProductBlueprintApi: productBlueprintCategoryId と productBlueprintCategory.id が一致しません",
    );
  }
}

/**
 * ProductBlueprint の ID 抽出
 * backend 正: id
 */
function extractProductBlueprintId(json: ProductBlueprintDetailResponse): string {
  return typeof json.id === "string" ? json.id : "";
}

function dedupKeepOrder(xs: string[]): string[] {
  const seen = new Set<string>();
  const out: string[] = [];

  for (const raw of xs ?? []) {
    const v = String(raw ?? "").trim();
    if (!v) continue;
    if (seen.has(v)) continue;
    seen.add(v);
    out.push(v);
  }

  return out;
}

function normalizeMeasurements(
  measurements: ApparelMeasurements | undefined,
): Record<string, number> {
  return Object.fromEntries(
    Object.entries(measurements ?? {}).filter(([, value]) => value != null),
  ) as Record<string, number>;
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
    measurements: normalizeMeasurements(v.measurements),
  }));
}

function shouldCreateApparelModelVariations(
  params: CreateProductBlueprintParams,
  variations: NewModelVariationPayload[],
): boolean {
  if (params.productBlueprintCategory.kind !== "apparel") {
    return false;
  }

  return variations.length > 0;
}

/**
 * ProductBlueprint + apparel ModelVariations をまとめて作成し、
 * さらに modelRefs（modelIds）を append する API 呼び出し。
 *
 * - ProductBlueprint 自体の作成は createProductBlueprintHTTP に委譲
 * - apparel category の場合だけ variations を作成
 * - variations 作成で得られた modelIds を順序付きで append
 * - append の返り値（detail）を最終結果として返す
 */
export async function createProductBlueprintApi(
  params: CreateProductBlueprintParams,
  variations: NewModelVariationPayload[],
): Promise<ProductBlueprintResponse> {
  assertProductBlueprintCategory(params);

  const created = await createProductBlueprintHTTP(params);
  const productBlueprintId = extractProductBlueprintId(created);

  if (!productBlueprintId) {
    throw new Error("createProductBlueprintApi: 作成後の id が空です");
  }

  if (!shouldCreateApparelModelVariations(params, variations)) {
    return created;
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
  return detail;
}