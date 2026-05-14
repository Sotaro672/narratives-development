// frontend/console/productBlueprint/src/application/productBlueprintCreateService.ts

import {
  isApparelCategoryCode,
  normalizeApparelMeasurements,
  type ApparelCategoryCode,
  type ApparelMeasurements,
  type ApparelModelVariationPayload,
  type ApparelSizeRow,
} from "../domain/entity/apparel";

import type {
  CategoryFieldValues,
  ProductBlueprintCategorySnapshot,
} from "../domain/entity/productBlueprintCategory";

import { hexToRgbInt } from "../../../shell/src/shared/util/color";

import type { ProductBlueprintDetailResponse } from "../infrastructure/api/productBlueprintDetailApi";

import {
  appendModelRefsHTTP,
  createProductBlueprintHTTP,
} from "../infrastructure/repository/productBlueprintRepositoryHTTP";

import {
  createModelVariations,
  type CreateModelVariationRequest,
} from "../../../model/src/infrastructure/repository/modelRepositoryHTTP";

// ------------------------------------------------------
// Product ID Tag
// ------------------------------------------------------

export type ProductIDTagType = "qr" | "nfc";

export type ProductIDTag = {
  type: ProductIDTagType;
};

// ------------------------------------------------------
// 作成用型
// ------------------------------------------------------

export type CreateProductBlueprintModelNumber = {
  size: string;
  color: string;
  code: string;
};

export type CreateProductBlueprintParams = {
  productName: string;
  brandId: string;

  productBlueprintCategoryId: string;
  productBlueprintCategory: ProductBlueprintCategorySnapshot;

  fit?: string | null;
  material?: string | null;
  weight?: number | null;
  qualityAssurance?: string[] | null;

  productIdTag: ProductIDTag;

  companyId: string;
  assigneeId?: string;
  createdBy?: string;

  colors?: string[];
  sizes?: ApparelSizeRow[];
  modelNumbers?: CreateProductBlueprintModelNumber[];
  colorRgbMap?: Record<string, string>;

  categoryFields?: CategoryFieldValues | null;
};

export type ProductBlueprintResponse = ProductBlueprintDetailResponse;

export type NewModelVariationPayload = ApparelModelVariationPayload;

// ------------------------------
// validation helpers
// ------------------------------

function assertProductBlueprintCategory(
  params: CreateProductBlueprintParams,
): void {
  if (!params.productBlueprintCategoryId?.trim()) {
    throw new Error("createProductBlueprint: productBlueprintCategoryId が空です");
  }

  if (!params.productBlueprintCategory?.id?.trim()) {
    throw new Error("createProductBlueprint: productBlueprintCategory.id が空です");
  }

  if (params.productBlueprintCategoryId !== params.productBlueprintCategory.id) {
    throw new Error(
      "createProductBlueprint: productBlueprintCategoryId と productBlueprintCategory.id が一致しません",
    );
  }
}

function extractProductBlueprintId(json: ProductBlueprintDetailResponse): string {
  return typeof json.id === "string" ? json.id : "";
}

function dedupKeepOrder(values: string[]): string[] {
  const seen = new Set<string>();
  const out: string[] = [];

  for (const raw of values) {
    const value = String(raw ?? "").trim();

    if (!value) {
      continue;
    }

    if (seen.has(value)) {
      continue;
    }

    seen.add(value);
    out.push(value);
  }

  return out;
}

function normalizeMeasurementsForRequest(
  measurements: ApparelMeasurements | undefined,
): Record<string, number> {
  const out: Record<string, number> = {};

  for (const [key, value] of Object.entries(measurements ?? {})) {
    if (typeof value === "number" && Number.isFinite(value)) {
      out[key] = value;
    }
  }

  return out;
}

function toCreateModelVariationRequests(args: {
  productBlueprintId: string;
  variations: NewModelVariationPayload[];
}): CreateModelVariationRequest[] {
  const { productBlueprintId, variations } = args;

  return variations.map((variation): CreateModelVariationRequest => ({
    productBlueprintId,
    modelNumber: String(variation.modelNumber ?? ""),
    size: String(variation.sizeLabel ?? ""),
    color: String(variation.color ?? ""),
    rgb: typeof variation.rgb === "number" ? variation.rgb : 0,
    measurements: normalizeMeasurementsForRequest(variation.measurements),
  }));
}

function shouldAppendModelRefs(
  params: CreateProductBlueprintParams,
  variations: NewModelVariationPayload[],
): boolean {
  if (params.productBlueprintCategory.kind !== "apparel") {
    return false;
  }

  return variations.length > 0;
}

// ------------------------------
// apparel measurements builder
// ------------------------------

function resolveApparelCategoryCode(
  params: CreateProductBlueprintParams,
): ApparelCategoryCode | null {
  const code = String(params.productBlueprintCategory?.code ?? "").trim();

  if (!isApparelCategoryCode(code)) {
    return null;
  }

  return code;
}

/**
 * apparel category code に応じて measurements を組み立てる。
 *
 * model/src/domain/entity/catalog.ts の MeasurementKey / SizeRow を正とする。
 *
 * category code:
 * - apparel.tops
 * - apparel.bottoms
 * - apparel.dress
 *
 * measurement key:
 * - 着丈
 * - 身幅
 * - 胸囲
 * - 肩幅
 * - 袖丈
 * - ウエスト
 * - ヒップ
 * - 股上
 * - 股下
 * - わたり幅
 * - 裾幅
 */
function buildApparelMeasurements(
  categoryCode: ApparelCategoryCode,
  size: ApparelSizeRow,
): ApparelMeasurements {
  const result: ApparelMeasurements = {};

  switch (categoryCode) {
    case "apparel.bottoms": {
      result["ウエスト"] = size.waist ?? null;
      result["ヒップ"] = size.hip ?? null;
      result["股上"] = size.rise ?? null;
      result["股下"] = size.inseam ?? null;
      result["わたり幅"] = size.thigh ?? null;
      result["裾幅"] = size.hemWidth ?? null;
      return result;
    }

    case "apparel.dress": {
      result["着丈"] = size.length ?? null;
      result["身幅"] = size.width ?? null;
      result["胸囲"] = size.chest ?? null;
      result["肩幅"] = size.shoulder ?? null;
      result["袖丈"] = size.sleeveLength ?? null;
      result["ウエスト"] = size.waist ?? null;
      result["ヒップ"] = size.hip ?? null;
      return result;
    }

    case "apparel.tops": {
      result["着丈"] = size.length ?? null;
      result["身幅"] = size.width ?? null;
      result["胸囲"] = size.chest ?? null;
      result["肩幅"] = size.shoulder ?? null;
      result["袖丈"] = size.sleeveLength ?? null;
      return result;
    }

    case "apparel.outerwear":
    case "apparel.shoes":
    case "apparel.bag":
    case "apparel.accessory":
    default: {
      return result;
    }
  }
}

/**
 * ProductBlueprintCategory / SizeRow / 各種コードから
 * apparel ModelVariation payload を組み立てる共通ヘルパー。
 */
function toApparelModelVariationPayload(
  categoryCode: ApparelCategoryCode,
  sizeRow: ApparelSizeRow,
  base: {
    sizeLabel: string;
    color: string;
    modelNumber: string;
    createdBy: string;
    rgb?: number;
  },
): ApparelModelVariationPayload {
  const measurements = buildApparelMeasurements(categoryCode, sizeRow);

  return {
    sizeLabel: base.sizeLabel,
    color: base.color,
    modelNumber: base.modelNumber,
    createdBy: base.createdBy,
    rgb: base.rgb,
    measurements,
  };
}

/**
 * modelNumbers 配列を (size,color) -> code の Map に変換する。
 *
 * - trim
 * - 空は除外
 * - 重複キーは後勝ち
 */
function buildModelNumberMap(
  modelNumbers: CreateProductBlueprintModelNumber[] | undefined,
): Map<string, string> {
  const map = new Map<string, string>();

  if (!modelNumbers || modelNumbers.length === 0) {
    return map;
  }

  for (const modelNumber of modelNumbers) {
    const size = String(modelNumber.size ?? "").trim();
    const color = String(modelNumber.color ?? "").trim();
    const code = String(modelNumber.code ?? "").trim();

    if (!size || !color || !code) {
      continue;
    }

    map.set(`${size}__${color}`, code);
  }

  return map;
}

function shouldCreateApparelModelVariations(
  categoryCode: ApparelCategoryCode,
): boolean {
  return (
    categoryCode === "apparel.tops" ||
    categoryCode === "apparel.bottoms" ||
    categoryCode === "apparel.dress" ||
    categoryCode === "apparel.outerwear" ||
    categoryCode === "apparel.shoes"
  );
}

async function createProductBlueprintWithVariations(
  params: CreateProductBlueprintParams,
  variations: NewModelVariationPayload[],
): Promise<ProductBlueprintResponse> {
  assertProductBlueprintCategory(params);

  const created = await createProductBlueprintHTTP(params);
  const productBlueprintId = extractProductBlueprintId(created);

  if (!productBlueprintId) {
    throw new Error("createProductBlueprint: 作成後の id が空です");
  }

  if (!shouldAppendModelRefs(params, variations)) {
    return created;
  }

  const requests = toCreateModelVariationRequests({
    productBlueprintId,
    variations,
  });

  const modelIds = await createModelVariations(productBlueprintId, requests);
  const cleanedModelIds = dedupKeepOrder(modelIds);

  if (cleanedModelIds.length === 0) {
    throw new Error("createProductBlueprint: modelIds が空です");
  }

  return await appendModelRefsHTTP(productBlueprintId, cleanedModelIds);
}

// ------------------------------
// Service 本体（アプリケーション層）
// ------------------------------

export async function createProductBlueprint(
  params: CreateProductBlueprintParams,
): Promise<ProductBlueprintResponse> {
  const categoryCode = resolveApparelCategoryCode(params);

  /**
   * apparel 以外は size/color/measurements 前提の ModelVariation を作成しない。
   * productBlueprint 本体のみ作成する。
   */
  if (!categoryCode) {
    return await createProductBlueprintWithVariations(params, []);
  }

  /**
   * apparel.bag / apparel.accessory は入力表上 model field が無いため、
   * ModelVariation は作成しない。
   */
  if (!shouldCreateApparelModelVariations(categoryCode)) {
    return await createProductBlueprintWithVariations(params, []);
  }

  const variations: ApparelModelVariationPayload[] = [];

  // displayOrder の採番元を「色登録順 → サイズ登録順」に固定
  const colors: string[] = (params.colors ?? [])
    .map((color: string) => String(color).trim())
    .filter((color: string) => color.length > 0);

  const sizes: ApparelSizeRow[] = (params.sizes ?? []).filter(
    (size: ApparelSizeRow) => String(size.sizeLabel ?? "").trim() !== "",
  );

  const colorRgbMap = params.colorRgbMap ?? {};
  const modelNumberMap = buildModelNumberMap(params.modelNumbers);

  for (const color of colors) {
    for (const sizeRow of sizes) {
      const sizeLabel = String(sizeRow.sizeLabel ?? "").trim();

      if (!sizeLabel) {
        continue;
      }

      const code = modelNumberMap.get(`${sizeLabel}__${color}`)?.trim();

      if (!code) {
        continue;
      }

      const hex = colorRgbMap[color];
      const rgbInt = hexToRgbInt(hex);

      const payload = toApparelModelVariationPayload(categoryCode, sizeRow, {
        sizeLabel,
        color,
        modelNumber: code,
        createdBy: params.createdBy ?? "",
        rgb: rgbInt,
      });

      variations.push({
        ...payload,
        measurements: normalizeApparelMeasurements(payload.measurements),
      });
    }
  }

  return await createProductBlueprintWithVariations(params, variations);
}