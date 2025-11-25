// frontend/console/productBlueprint/src/application/productBlueprintCreateService.ts 

import type { ItemType, Fit } from "../domain/entity/catalog";
import type { ProductIDTag } from "../domain/entity/productBlueprint";

// SizeRow / ModelNumber は model 側の型を利用
import type {
  SizeRow,
  MeasurementKey,
} from "../../../model/src/domain/entity/catalog";
import type { ModelNumber } from "../../../model/src/application/modelCreateService";

// HTTP 呼び出しは infrastructure 層に委譲
import {
  createProductBlueprintHTTP,
  // 将来的に直接 HTTP で ModelVariation を作成したくなった場合に利用
  // createModelVariationHTTP,
} from "../infrastructure/repository/productBlueprintRepositoryHTTP";

// ProductBlueprint 作成後の JSON を受け取るアプリケーション層サービス
import { createModelVariationsFromProductBlueprint } from "../../../model/src/application/modelCreateService";

// ------------------------------
// 型定義
// ------------------------------

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

  // 商品設計画面から渡されるバリエーション情報
  colors: string[];
  sizes: SizeRow[];
  modelNumbers: ModelNumber[];

  // ★ ColorVariationCard から渡される color 名 → HEX(RGB) のマップ
  //   例: { "グリーン": "#417505" }
  colorRgbMap?: Record<string, string>;
};

export type ProductBlueprintResponse = {
  ID?: string;
  id?: string;
  productBlueprintId?: string;
  [key: string]: unknown;
};

/**
 * measurements 部分の型
 * - modelCreateService.tsx 側と同じく、MeasurementKey をキーにしたマップ
 */
export type NewModelVariationMeasurements = Partial<
  Record<MeasurementKey, number | null>
>;

/**
 * ModelVariation 用 Payload
 *
 * - modelCreateService.tsx 側の NewModelVariationPayload と構造互換
 */
export type NewModelVariationPayload = {
  sizeLabel: string;
  color: string;
  rgb?: number; // ★ 色の RGB 値（0xRRGGBB）
  modelNumber: string;
  createdBy: string;
  measurements: NewModelVariationMeasurements;
};

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

export async function createProductBlueprint(
  params: CreateProductBlueprintParams,
): Promise<ProductBlueprintResponse> {
  // 1. ProductBlueprint の作成（HTTP）
  const json = await createProductBlueprintHTTP(params);

  // 2. productBlueprintId 抽出（backend がどのキーで返してもある程度吸収する）
  const anyJson = json as any;
  const productBlueprintIdRaw =
    anyJson.productBlueprintId ??
    anyJson.productBlueprintID ??
    anyJson.id ??
    anyJson.ID;

  const productBlueprintId =
    typeof productBlueprintIdRaw === "string"
      ? productBlueprintIdRaw.trim()
      : "";

  if (!productBlueprintId) {
    console.warn(
      "[productBlueprintCreateService] productBlueprintId not found in response; skip ModelVariation creation",
      json,
    );
    return json;
  }

  // 3. color / size / modelNumber / measurements から
  //    modelCreateService.tsx に渡す JSON を組み立てる
  const variations: NewModelVariationPayload[] = [];

  const colorRgbMap = params.colorRgbMap ?? {};

  if (params.modelNumbers && params.sizes) {
    for (const v of params.modelNumbers) {
      // 該当サイズの SizeRow を取得
      const sizeRow = params.sizes.find((s) => s.sizeLabel === v.size);
      if (!sizeRow) {
        // サイズ行が見つからない場合はスキップ
        console.warn(
          "[productBlueprintCreateService] SizeRow not found for modelNumber; skip one variation",
          v,
        );
        continue;
      }

      // ★ color 名から HEX を取得し、RGB(int) に変換
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

  // 🔍 backend（/models/{productBlueprintId}/variations）に渡す直前の payload 全体をログ出力
  console.log(
    "[productBlueprintCreateService] variations payload for backend",
    {
      productBlueprintId,
      variations,
    },
  );

  // 4. modelCreateService.tsx へ JSON を渡す
  //    - ここでは「productBlueprint を Create した結果」を元に
  //      model 作成（variations 作成）の起点となる payload を組み立てて渡す。
  if (variations.length > 0) {
    await createModelVariationsFromProductBlueprint({
      productBlueprintId,
      variations,
    });
  } else {
    console.log(
      "[productBlueprintCreateService] no variations to create; variations array is empty",
    );
  }

  return json;
}
