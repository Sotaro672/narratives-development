// frontend/console/productBlueprint/src/application/productBlueprintCreateService.ts

import type { ItemType, Fit } from "../domain/entity/catalog";
import type { ProductIDTag } from "../domain/entity/productBlueprint";

// ★ measurements を共通化するユーティリティ
import { buildMeasurements } from "../../../model/src/application/buildMeasurements";
import type { SizeRow } from "../../../model/src/domain/entity/catalog";

// HTTP 呼び出しは infrastructure 層に委譲
import {
  createProductBlueprintHTTP,
  // future: createModelVariationHTTP,
} from "../infrastructure/repository/productBlueprintRepositoryHTTP";

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

  // ※ 実際の実装では colors / sizes / modelNumbers なども
  //    ここに追加されている前提（useProductBlueprintCreate の apiParams と対応）
  // colors?: string[];
  // sizes?: SizeRow[];
  // modelNumbers?: { size: string; color: string; code: string }[];
};

export type ProductBlueprintResponse = {
  ID?: string;
  id?: string;
  productId?: string;
  productID?: string;
  [key: string]: unknown;
};

/**
 * ModelVariation 用 Payload
 *
 * ★ createdBy を追加
 * ★ itemType がトップス / ボトムス どちらでも対応できる柔軟な measurements 形式
 */
export type NewModelVariationPayload = {
  sizeLabel: string;
  color: string;
  modelNumber: string;
  createdBy: string; // 🔥 追加

  measurements: {
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
};

/**
 * itemType / SizeRow / 各種コードから NewModelVariationPayload を組み立てる共通ヘルパー
 * measurements 部分は buildMeasurements() を使って一元管理する。
 */
export function toNewModelVariationPayload(
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

  // 2. productId 抽出
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

  // ★ ここで modelVariation を作るためのデータを組み立てる予定
  //    measurements の構築には toNewModelVariationPayload / buildMeasurements を利用する。
  //
  // if (params.modelNumbers && params.sizes) {
  //   for (const v of params.modelNumbers) {
  //     const sizeRow = params.sizes.find((s) => s.sizeLabel === v.size);
  //     if (!sizeRow) continue;
  //
  //     const payload = toNewModelVariationPayload(params.itemType, sizeRow, {
  //       sizeLabel: v.size,
  //       color: v.color,
  //       modelNumber: v.code,
  //       createdBy: params.createdBy ?? "",
  //     });
  //
  //     await createModelVariationHTTP(productId, payload);
  //   }
  // }

  return json;
}
