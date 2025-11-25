// frontend/console/productBlueprint/src/application/productBlueprintCreateService.ts 

import type { ItemType } from "../domain/entity/catalog";

// SizeRow は model 側の型を利用
import type { SizeRow } from "../../../model/src/domain/entity/catalog";

// API 呼び出しは infrastructure 層（api）に委譲
import { createProductBlueprintApi } from "../infrastructure/api/productBlueprintApi";
import type {
  CreateProductBlueprintParams,
  ProductBlueprintResponse,
  NewModelVariationPayload,
  NewModelVariationMeasurements,
} from "../infrastructure/api/productBlueprintApi";

// 他モジュールからも型を引き続きここ経由で参照できるように re-export
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

  // 6桁以外は無視（ログだけ出す）
  if (!/^[0-9a-fA-F]{6}$/.test(withoutHash)) {
    console.warn(
      "[productBlueprintCreateService] invalid rgb hex format",
      { hex },
    );
    return undefined;
  }

  const parsed = parseInt(withoutHash, 16);
  if (Number.isNaN(parsed)) {
    console.warn(
      "[productBlueprintCreateService] failed to parse rgb hex",
      { hex },
    );
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
 */
function buildMeasurements(
  itemType: ItemType,
  size: SizeRow,
): NewModelVariationMeasurements {
  const result: NewModelVariationMeasurements = {};

  if (itemType === "ボトムス") {
    // ボトムス用の採寸マッピング
    result["ウエスト"] = size.waist ?? null;
    result["ヒップ"] = size.hip ?? null;
    result["股上"] = size.rise ?? null;
    result["股下"] = size.inseam ?? null;
    result["わたり幅"] = size.thighWidth ?? null;
    result["裾幅"] = size.hemWidth ?? null;
    return result;
  }

  // デフォルト（トップス想定）
  result["着丈"] = size.lengthTop ?? null;
  result["身幅"] = size.bodyWidth ?? null;
  result["肩幅"] = size.shoulderWidth ?? null;
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

  // 🔍 buildMeasurements & rgb をここでログ出力
  console.log("[productBlueprintCreateService] buildMeasurements result", {
    itemType,
    sizeRow,
    base,
    measurements,
  });

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

/**
 * アプリケーション層の createProductBlueprint
 *
 * - variations の計算（SizeRow / ModelNumber / itemType / colorRgbMap から構築）
 * - その結果を infrastructure/api の createProductBlueprintApi に委譲
 */
export async function createProductBlueprint(
  params: CreateProductBlueprintParams,
): Promise<ProductBlueprintResponse> {
  // 1. color / size / modelNumber / measurements から
  //    modelCreateService.ts に渡す JSON を組み立てる
  const variations: NewModelVariationPayload[] = [];

  const colorRgbMap = params.colorRgbMap ?? {};

  if (params.modelNumbers && params.sizes) {
    for (const v of params.modelNumbers) {
      // 該当サイズの SizeRow を取得（コールバック引数に型を明示）
      const sizeRow = params.sizes.find(
        (s: SizeRow) => s.sizeLabel === v.size,
      );
      if (!sizeRow) {
        // サイズ行が見つからない場合はスキップ
        console.warn(
          "[productBlueprintCreateService] SizeRow not found for modelNumber; skip one variation",
          v,
        );
        continue;
      }

      // color 名から HEX を取得し、RGB(int) に変換
      const hex = colorRgbMap[v.color];
      const rgbInt = hexToRgbInt(hex);

      // rgb を含めて payload を組み立て
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

  // 🔍 backend へ渡す variations 全体をログ出力（id 抽出前段階）
  console.log(
    "[productBlueprintCreateService] variations payload (before API call)",
    {
      variations,
    },
  );

  // 2. API モジュールに委譲（ProductBlueprint 作成 + ModelVariations 作成）
  return await createProductBlueprintApi(params, variations);
}
