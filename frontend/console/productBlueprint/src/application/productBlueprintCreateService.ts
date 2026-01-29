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

import { hexToRgbInt } from "../../../shell/src/shared/util/color";

export type {
  CreateProductBlueprintParams,
  ProductBlueprintResponse,
} from "../infrastructure/api/productBlueprintApi";

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
  result["身幅"] = size.width ?? null;
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

/**
 * modelNumbers 配列を (size,color) -> code の Map に変換する。
 * - trim
 * - 空は除外
 * - 重複キーは「後勝ち」にする（同一 key が来た場合に最新入力を採用）
 */
function buildModelNumberMap(
  modelNumbers: Array<{ size: string; color: string; code: string }> | undefined,
): Map<string, string> {
  const m = new Map<string, string>();
  if (!modelNumbers || modelNumbers.length === 0) return m;

  for (const mn of modelNumbers) {
    const size = (mn.size ?? "").trim();
    const color = (mn.color ?? "").trim();
    const code = (mn.code ?? "").trim();
    if (!size || !color || !code) continue;
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
  const variations: NewModelVariationPayload[] = [];

  // displayOrder の採番元（将来 modelIds を組むときの順序）を
  // 「色登録順 → サイズ登録順」に固定するため、variations 生成もその順序に揃える。
  const colors = (params.colors ?? []).map((c) => String(c).trim()).filter(Boolean);
  const sizes = (params.sizes ?? []).filter(
    (s: SizeRow) => String((s as any).sizeLabel ?? "").trim() !== "",
  );

  const colorRgbMap = params.colorRgbMap ?? {};
  const modelNumberMap = buildModelNumberMap(params.modelNumbers as any);

  for (const color of colors) {
    for (const sizeRow of sizes) {
      const sizeLabel = String((sizeRow as any).sizeLabel ?? "").trim();
      if (!sizeLabel) continue;

      const code = modelNumberMap.get(`${sizeLabel}__${color}`)?.trim();
      if (!code) continue;

      const hex = colorRgbMap[color];
      const rgbInt = hexToRgbInt(hex);

      variations.push(
        toNewModelVariationPayload(params.itemType, sizeRow, {
          sizeLabel,
          color,
          modelNumber: code,
          createdBy: params.createdBy ?? "",
          rgb: rgbInt,
        }),
      );
    }
  }

  // NOTE:
  // - productIdTag の DTO 形状はバックエンドは { productIdTag: { type } }。
  // - 現行 API が productIdTagType を期待している可能性があるため、ここでは後方互換として残す。
  // - 次手順で infrastructure/api 側を改修し、最終的にここから productIdTagType の「型詐称」を排除する。
  return await createProductBlueprintApi(
    {
      ...params,
      productIdTagType: params.productIdTag?.type ?? null,
    } as CreateProductBlueprintParams,
    variations,
  );
}
