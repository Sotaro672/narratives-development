// frontend/console/productBlueprint/src/application/productBlueprintCreateService.ts

import type { ItemType, Fit } from "../domain/entity/catalog";
import type { ProductIDTag } from "../domain/entity/productBlueprint";

// SizeRow / ModelNumber は model 側の型を利用
import type { SizeRow } from "../../../model/src/domain/entity/catalog";
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
};

export type ProductBlueprintResponse = {
  ID?: string;
  id?: string;
  productId?: string;
  productID?: string;
  [key: string]: unknown;
};

/**
 * measurements 部分の型
 * - modelCreateService.tsx 側の NewModelVariationMeasurements と同じ構造
 */
export type NewModelVariationMeasurements = {
  // Top
  chest?: number | null;
  shoulder?: number | null;

  // Bottom
  waist?: number | null;
  length?: number | null;

  // 共通で他項目を追加したい場合はここに拡張可能
  hip?: number | null;
  thigh?: number | null;
};

/**
 * ModelVariation 用 Payload
 *
 * - modelCreateService.tsx 側の NewModelVariationPayload と構造互換
 */
export type NewModelVariationPayload = {
  sizeLabel: string;
  color: string;
  modelNumber: string;
  createdBy: string;
  measurements: NewModelVariationMeasurements;
};

// ------------------------------
// buildMeasurements をこのファイルに集約
// ------------------------------

/**
 * itemType に応じて measurements を組み立てるユーティリティ
 *
 * chest / shoulder / waist / length の 4 項目だけを返す。
 * （hip / thigh は呼び出し側で null を詰める）
 */
function buildMeasurements(
  itemType: ItemType,
  size: SizeRow,
): Omit<NewModelVariationMeasurements, "hip" | "thigh"> {
  // ボトムスの場合: ウエスト / 丈 を優先して埋める
  if (itemType === "ボトムス") {
    return {
      // ボトムスでは胸囲・肩幅は使わないので null
      chest: null,
      shoulder: null,
      waist: size.waist ?? null,
      length: size.length ?? null,
    };
  }

  // デフォルト（トップス想定）
  return {
    chest: size.chest ?? null,
    shoulder: size.shoulder ?? null,
    waist: size.waist ?? null,
    length: size.length ?? null,
  };
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
  },
): NewModelVariationPayload {
  const baseMeasurements = buildMeasurements(itemType, sizeRow);

  // 🔍 buildMeasurements で組み立てた値をここでログ出力
  console.log(
    "[productBlueprintCreateService] buildMeasurements result",
    {
      itemType,
      sizeRow,
      base,
      measurements: baseMeasurements,
    },
  );

  return {
    sizeLabel: base.sizeLabel,
    color: base.color,
    modelNumber: base.modelNumber,
    createdBy: base.createdBy,
    measurements: {
      // chest / shoulder / waist / length は buildMeasurements に委譲
      ...baseMeasurements,
      // まだ未対応の採寸は null で固定
      hip: null,
      thigh: null,
    },
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

  // 2. productId 抽出（backend がどのキーで返してもある程度吸収する）
  const anyJson = json as any;
  const productIdRaw =
    anyJson.productId ??
    anyJson.productID ??
    anyJson.id ??
    anyJson.ID;

  const productId =
    typeof productIdRaw === "string" ? productIdRaw.trim() : "";

  if (!productId) {
    console.warn(
      "[productBlueprintCreateService] productId not found in response; skip ModelVariation creation",
      json,
    );
    return json;
  }

  // 3. color / size / modelNumber / measurements から
  //    modelCreateService.tsx に渡す JSON を組み立てる
  const variations: NewModelVariationPayload[] = [];

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

      const payload = toNewModelVariationPayload(params.itemType, sizeRow, {
        sizeLabel: v.size,
        color: v.color,
        modelNumber: v.code,
        createdBy: params.createdBy ?? "",
      });

      variations.push(payload);
    }
  }

  // 🔍 backend（/models/{productId}/variations）に渡す直前の payload 全体をログ出力
  console.log(
    "[productBlueprintCreateService] variations payload for backend",
    {
      productId,
      variations,
    },
  );

  // 4. modelCreateService.tsx へ JSON を渡す
  //    - ここでは「productBlueprint を Create した結果」を元に
  //      model 作成（variations 作成）の起点となる payload を組み立てて渡す。
  if (variations.length > 0) {
    await createModelVariationsFromProductBlueprint({
      productId,
      variations,
    });
  } else {
    console.log(
      "[productBlueprintCreateService] no variations to create; variations array is empty",
    );
  }

  return json;
}
