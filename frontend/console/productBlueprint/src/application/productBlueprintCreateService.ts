// frontend/console/productBlueprint/src/application/productBlueprintCreateService.ts

import {
  isApparelCategoryCode,
  normalizeApparelMeasurements,
  type ApparelCategoryCode,
  type ApparelMeasurements,
  type ApparelModelVariationPayload,
  type ApparelSizeRow,
} from "../domain/entity/apparel";

import { isAlcoholCategoryCode } from "../domain/entity/alcohol";

import type {
  AlcoholModelNumber,
  Volume,
  VolumeRow,
} from "../../../model/src/application/modelCreateService";

import type {
  CategoryFieldValues,
  ProductBlueprintCategorySnapshot,
} from "../domain/entity/productBlueprintCategory";

import { hexToRgbInt } from "../../../shell/src/shared/util/color";

import type { ProductBlueprintDetailResponse } from "../infrastructure/api/productBlueprintDetailApi";

import { createProductBlueprintHTTP } from "../infrastructure/repository/productBlueprintRepositoryHTTP";

import {
  createModelVariations,
  type CreateModelVariationRequest,
} from "../../../model/src/infrastructure/api/modelCreateApi";

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

  /**
   * alcohol model variation 用。
   * volume は ProductBlueprint.categoryFields ではなく model domain 側で扱う。
   */
  volumes?: VolumeRow[];
  alcoholModelNumbers?: AlcoholModelNumber[];

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

function toCreateApparelModelVariationRequests(args: {
  productBlueprintId: string;
  variations: NewModelVariationPayload[];
}): CreateModelVariationRequest[] {
  const { productBlueprintId, variations } = args;

  return variations.map(
    (variation): CreateModelVariationRequest => ({
      kind: "apparel",
      productBlueprintId,
      modelNumber: String(variation.modelNumber ?? ""),
      size: String(variation.sizeLabel ?? ""),
      color: String(variation.color ?? ""),
      rgb: typeof variation.rgb === "number" ? variation.rgb : 0,
      measurements: normalizeMeasurementsForRequest(variation.measurements),
    }),
  );
}

function makeVolumeKey(volume: Volume): string {
  const value =
    typeof volume.value === "number" && Number.isFinite(volume.value)
      ? volume.value
      : 0;

  const unit = String(volume.unit ?? "").trim() || "ml";

  if (value <= 0) {
    return "";
  }

  return `${value}${unit}`;
}

function volumeRowToVolume(row: VolumeRow): Volume {
  return {
    value: row.volumeValue,
    unit: String(row.volumeUnit ?? "").trim() || "ml",
  };
}

function buildAlcoholModelNumberMap(
  modelNumbers: AlcoholModelNumber[] | undefined,
): Map<string, AlcoholModelNumber> {
  const map = new Map<string, AlcoholModelNumber>();

  if (!modelNumbers || modelNumbers.length === 0) {
    return map;
  }

  for (const modelNumber of modelNumbers) {
    const key = makeVolumeKey(modelNumber.volume);
    const code = String(modelNumber.code ?? "").trim();

    if (!key || !code) {
      continue;
    }

    map.set(key, {
      ...modelNumber,
      code,
    });
  }

  return map;
}

function toCreateAlcoholModelVariationRequests(args: {
  productBlueprintId: string;
  volumes: VolumeRow[];
  alcoholModelNumbers: AlcoholModelNumber[];
}): CreateModelVariationRequest[] {
  const { productBlueprintId, volumes, alcoholModelNumbers } = args;

  const modelNumberMap = buildAlcoholModelNumberMap(alcoholModelNumbers);
  const requests: CreateModelVariationRequest[] = [];
  const seen = new Set<string>();

  for (const row of volumes) {
    const volume = volumeRowToVolume(row);
    const key = makeVolumeKey(volume);

    if (!key || seen.has(key)) {
      continue;
    }

    seen.add(key);

    const modelNumber = modelNumberMap.get(key);

    if (!modelNumber) {
      continue;
    }

    requests.push({
      kind: "alcohol",
      productBlueprintId,
      modelNumber: modelNumber.code,
      volume,
    });
  }

  return requests;
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

function isAlcoholProductBlueprintCategory(
  params: CreateProductBlueprintParams,
): boolean {
  const code = String(params.productBlueprintCategory?.code ?? "").trim();
  return isAlcoholCategoryCode(code);
}

/**
 * apparel category code に応じて measurements を組み立てる。
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

async function createProductBlueprintWithModelRequests(
  params: CreateProductBlueprintParams,
  requests: CreateModelVariationRequest[],
): Promise<ProductBlueprintResponse> {
  assertProductBlueprintCategory(params);

  const created = await createProductBlueprintHTTP(params);
  const productBlueprintId = extractProductBlueprintId(created);

  if (!productBlueprintId) {
    throw new Error("createProductBlueprint: 作成後の id が空です");
  }

  if (requests.length === 0) {
    return created;
  }

  const normalizedRequests = requests.map((request) => ({
    ...request,
    productBlueprintId,
  })) as CreateModelVariationRequest[];

  await createModelVariations(productBlueprintId, normalizedRequests);

  return created;
}

// ------------------------------
// Service 本体（アプリケーション層）
// ------------------------------

export async function createProductBlueprint(
  params: CreateProductBlueprintParams,
): Promise<ProductBlueprintResponse> {
  const apparelCategoryCode = resolveApparelCategoryCode(params);

  if (apparelCategoryCode && shouldCreateApparelModelVariations(apparelCategoryCode)) {
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

        const payload = toApparelModelVariationPayload(apparelCategoryCode, sizeRow, {
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

    const requests = toCreateApparelModelVariationRequests({
      productBlueprintId: "",
      variations,
    });

    return await createProductBlueprintWithModelRequests(params, requests);
  }

  if (isAlcoholProductBlueprintCategory(params)) {
    const requests = toCreateAlcoholModelVariationRequests({
      productBlueprintId: "",
      volumes: params.volumes ?? [],
      alcoholModelNumbers: params.alcoholModelNumbers ?? [],
    });

    return await createProductBlueprintWithModelRequests(params, requests);
  }

  /**
   * modelFields を持たないカテゴリでは ModelVariation を作成しない。
   */
  return await createProductBlueprintWithModelRequests(params, []);
}