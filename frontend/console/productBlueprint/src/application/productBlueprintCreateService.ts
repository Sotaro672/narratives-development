// frontend/console/productBlueprint/src/application/productBlueprintCreateService.ts

import {
  isApparelCategoryCode,
  normalizeApparelMeasurements,
  type ApparelCategoryCode,
  type ApparelMeasurements,
  type ApparelModelVariationPayload,
  type ApparelSizeRow,
} from "../domain/entity/apparel";

import { createProductBlueprintApi } from "../infrastructure/api/productBlueprintApi";

import type {
  CreateProductBlueprintParams,
  ProductBlueprintResponse,
} from "../infrastructure/api/productBlueprintApi";

import { hexToRgbInt } from "../../../shell/src/shared/util/color";

export type {
  CreateProductBlueprintParams,
  ProductBlueprintResponse,
} from "../infrastructure/api/productBlueprintApi";

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
 * itemType は廃止。
 * productBlueprintCategory.code を正として利用する。
 */
function buildApparelMeasurements(
  categoryCode: ApparelCategoryCode,
  size: ApparelSizeRow,
): ApparelMeasurements {
  const result: ApparelMeasurements = {};

  switch (categoryCode) {
    case "apparel.bottoms": {
      result.waist = size.waist ?? null;
      result.hip = size.hip ?? null;
      result.rise = size.rise ?? null;
      result.inseam = size.inseam ?? null;
      result.thighWidth = size.thighWidth ?? null;
      result.hemWidth = size.hemWidth ?? null;
      result.totalLength = size.totalLength ?? null;
      return result;
    }

    case "apparel.outerwear": {
      result.shoulderWidth = size.shoulderWidth ?? null;
      result.bodyWidth = size.bodyWidth ?? null;
      result.bodyLength = size.bodyLength ?? null;
      result.sleeveLength = size.sleeveLength ?? null;
      return result;
    }

    case "apparel.dress": {
      result.shoulderWidth = size.shoulderWidth ?? null;
      result.bodyWidth = size.bodyWidth ?? null;
      result.bodyLength = size.bodyLength ?? null;
      result.sleeveLength = size.sleeveLength ?? null;
      result.waist = size.waist ?? null;
      result.hip = size.hip ?? null;
      result.totalLength = size.totalLength ?? null;
      return result;
    }

    case "apparel.shoes": {
      result.heelHeight = size.heelHeight ?? null;
      return result;
    }

    case "apparel.bag": {
      result.width = size.width ?? null;
      result.height = size.height ?? null;
      result.depth = size.depth ?? null;
      return result;
    }

    case "apparel.accessory": {
      result.width = size.width ?? null;
      result.height = size.height ?? null;
      return result;
    }

    case "apparel.tops":
    default: {
      result.shoulderWidth = size.shoulderWidth ?? null;
      result.bodyWidth = size.bodyWidth ?? null;
      result.bodyLength = size.bodyLength ?? null;
      result.sleeveLength = size.sleeveLength ?? null;
      result.neckWidth = size.neckWidth ?? null;
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
  modelNumbers: Array<{ size: string; color: string; code: string }> | undefined,
): Map<string, string> {
  const m = new Map<string, string>();

  if (!modelNumbers || modelNumbers.length === 0) {
    return m;
  }

  for (const mn of modelNumbers) {
    const size = String(mn.size ?? "").trim();
    const color = String(mn.color ?? "").trim();
    const code = String(mn.code ?? "").trim();

    if (!size || !color || !code) {
      continue;
    }

    m.set(`${size}__${color}`, code);
  }

  return m;
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
    return await createProductBlueprintApi(params, []);
  }

  const variations: ApparelModelVariationPayload[] = [];

  // displayOrder の採番元を「色登録順 → サイズ登録順」に固定
  const colors = (params.colors ?? [])
    .map((c) => String(c).trim())
    .filter(Boolean);

  const sizes = (params.sizes ?? []).filter(
    (s) => String(s.sizeLabel ?? "").trim() !== "",
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

  return await createProductBlueprintApi(params, variations);
}