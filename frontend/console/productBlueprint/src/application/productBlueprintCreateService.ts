// frontend/console/productBlueprint/src/application/productBlueprintCreateService.ts 

import type { ItemType } from "../domain/entity/catalog";
import type { SizeRow } from "../../../model/src/domain/entity/catalog";
import { createProductBlueprintApi } from "../infrastructure/api/productBlueprintApi";
import type {
  CreateProductBlueprintParams,
  ProductBlueprintResponse,
  NewModelVariationPayload,
  NewModelVariationMeasurements,
} from "../infrastructure/api/productBlueprintApi";

export type {
  CreateProductBlueprintParams,
  ProductBlueprintResponse,
} from "../infrastructure/api/productBlueprintApi";

// ------------------------------
// HEX → number(RGB) 変換ヘルパー
// ------------------------------

function hexToRgbInt(hex?: string): number | undefined {
  if (!hex) return undefined;

  const trimmed = hex.trim();
  if (!trimmed) return undefined;

  const withoutHash = trimmed.startsWith("#")
    ? trimmed.slice(1)
    : trimmed;

  if (!/^[0-9a-fA-F]{6}$/.test(withoutHash)) {
    return undefined;
  }

  const parsed = parseInt(withoutHash, 16);
  if (Number.isNaN(parsed)) {
    return undefined;
  }

  return parsed;
}

// ------------------------------
// buildMeasurements をこのファイルに集約
// ------------------------------

/**
 * itemType に応じて measurements を組み立てるユーティリティ
 *
 * - MeasurementKey（catalog.ts）をキーにしたマップを返す。
 * - SizeVariationCard.tsx の mapLabelToField と対応させる。
 */
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
    result["わたり幅"] = size.thigh ?? null;
    result["裾幅"] = size.hemWidth ?? null;
    return result;
  }

  // トップス（デフォルト）
  result["着丈"] = size.length ?? null;
  result["身幅"] = size.bodyWidth ?? null;
  result["胸囲"] = size.chest ?? null;
  result["肩幅"] = size.shoulder ?? null;
  result["袖丈"] = size.sleeveLength ?? null;

  return result;
}

/**
 * itemType / SizeRow / 各種コードから NewModelVariationPayload を組み立てる共通ヘルパー
 * measurements 部分は buildMeasurements() を使って一元管理する。
 */
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

// ------------------------------
// Service 本体（アプリケーション層）
// ------------------------------

export async function createProductBlueprint(
  params: CreateProductBlueprintParams,
): Promise<ProductBlueprintResponse> {
  const variations: NewModelVariationPayload[] = [];

  const colorRgbMap = params.colorRgbMap ?? {};

  if (params.modelNumbers && params.sizes) {
    for (const v of params.modelNumbers) {
      const sizeRow = params.sizes.find(
        (s: SizeRow) => s.sizeLabel === v.size,
      );
      if (!sizeRow) {
        continue;
      }

      const hex = colorRgbMap[v.color];
      const rgbInt = hexToRgbInt(hex);

      const payload = toNewModelVariationPayload(params.itemType, sizeRow, {
        sizeLabel: v.size,
        color: v.color,
        modelNumber: v.code,
        createdBy: params.createdBy ?? "",
        rgb: rgbInt,
      });

      variations.push(payload);
    }
  }

  return await createProductBlueprintApi(
    {
      ...params,
      productIdTagType: params.productIdTag?.type ?? null,
    } as CreateProductBlueprintParams,
    variations,
  );
}
