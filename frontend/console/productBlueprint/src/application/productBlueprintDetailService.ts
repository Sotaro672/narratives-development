// frontend/console/productBlueprint/src/application/productBlueprintDetailService.ts

import type { ItemType } from "../domain/entity/catalog";
import type { SizeRow } from "../../../model/src/domain/entity/catalog";

import {
  getProductBlueprintDetailApi,
  updateProductBlueprintApi,
  type ProductBlueprintDetailResponse,
  type UpdateProductBlueprintParams,
  type NewModelVariationMeasurements,
  type NewModelVariationPayload,
} from "../infrastructure/api/productBlueprintDetailApi";

// -----------------------------------------
// HEX -> number(RGB) 変換
// -----------------------------------------
function hexToRgbInt(hex?: string): number | undefined {
  if (!hex) return undefined;
  const trimmed = hex.trim();
  const h = trimmed.startsWith("#") ? trimmed.slice(1) : trimmed;

  if (!/^[0-9a-fA-F]{6}$/.test(h)) return undefined;

  const parsed = parseInt(h, 16);
  if (Number.isNaN(parsed)) return undefined;

  return parsed;
}

// -----------------------------------------
// itemType → measurements 組み立て
// -----------------------------------------
function buildMeasurements(
  itemType: ItemType,
  size: SizeRow,
): NewModelVariationMeasurements {
  const result: NewModelVariationMeasurements = {};

  if (itemType === "ボトムス") {
    result["ウエスト"] = size.waist ?? null;
    result["ヒップ"] = size.hip ?? null;
    result["股上"] = size.rise ?? null;
    result["股下"] = size.inseam ?? null;
    result["わたり幅"] = size.thighWidth ?? null;
    result["裾幅"] = size.hemWidth ?? null;
    return result;
  }

  // トップス
  result["着丈"] = size.lengthTop ?? null;
  result["身幅"] = size.bodyWidth ?? null;
  result["肩幅"] = size.shoulderWidth ?? null;
  result["袖丈"] = size.sleeveLength ?? null;

  return result;
}

// -----------------------------------------
// variations payload builder
// -----------------------------------------
function toNewModelVariationPayload(
  itemType: ItemType,
  sizeRow: SizeRow,
  base: {
    sizeLabel: string;
    color: string;
    modelNumber: string;
    createdBy: string;
    rgb?: number;
  },
): NewModelVariationPayload {
  const measurements = buildMeasurements(itemType, sizeRow);

  return {
    sizeLabel: base.sizeLabel,
    color: base.color,
    modelNumber: base.modelNumber,
    createdBy: base.createdBy,
    rgb: base.rgb,
    measurements,
  };
}

// -----------------------------------------
// GET: 商品設計 詳細
// -----------------------------------------
export async function getProductBlueprintDetail(
  id: string,
): Promise<ProductBlueprintDetailResponse> {
  return await getProductBlueprintDetailApi(id);
}

// -----------------------------------------
// UPDATE: 商品設計 更新
// -----------------------------------------
export async function updateProductBlueprint(
  params: UpdateProductBlueprintParams,
): Promise<ProductBlueprintDetailResponse> {
  const variations: NewModelVariationPayload[] = [];

  const colorRgbMap = params.colorRgbMap ?? {};
  const itemType = params.itemType as ItemType; // ★ string → ItemType にキャスト

  if (params.modelNumbers && params.sizes) {
    for (const v of params.modelNumbers) {
      const sizeRow = params.sizes.find((s: SizeRow) => s.sizeLabel === v.size);
      if (!sizeRow) continue;

      const hex = colorRgbMap[v.color];
      const rgbInt = hexToRgbInt(hex);

      variations.push(
        toNewModelVariationPayload(itemType, sizeRow, {
          sizeLabel: v.size,
          color: v.color,
          modelNumber: v.code,
          createdBy: params.updatedBy ?? "",
          rgb: rgbInt,
        }),
      );
    }
  }

  console.log("[productBlueprintDetailService] update variations:", variations);

  return await updateProductBlueprintApi(params, variations);
}
