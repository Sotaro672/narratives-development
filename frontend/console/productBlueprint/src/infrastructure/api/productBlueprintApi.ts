// frontend/console/productBlueprint/src/infrastructure/api/productBlueprintApi.ts

import { PRODUCT_BLUEPRINTS } from "../mockdata/productBlueprint_mockdata";
import {
  MODEL_NUMBERS,
  SIZE_VARIATIONS,
} from "../../../../model/src/infrastructure/mockdata/mockdata";

// 一覧・詳細表示で利用する ProductBlueprint（モック用）
import type { ProductBlueprint } from "../../../../shell/src/shared/types/productBlueprint";

// ─────────────────────────────────────────────
// 作成系 API 用の型・依存
// ─────────────────────────────────────────────
import type { ItemType, Fit } from "../../domain/entity/catalog";
import type { ProductIDTag } from "../../../../productBlueprint/src/domain/entity/productBlueprint";
import type {
  SizeRow as CatalogSizeRow,
  MeasurementKey,
} from "../../../../model/src/domain/entity/catalog";
import type { ModelNumber } from "../../../../model/src/application/modelCreateService";

import { createProductBlueprintHTTP } from "../repository/productBlueprintRepositoryHTTP";
import { createModelVariationsFromProductBlueprint } from "../../../../model/src/application/modelCreateService";

// ISO8601 → "YYYY/MM/DD"（壊れてたらそのまま返す） ※一覧用
const toDisplayDate = (iso?: string | null): string => {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}/${m}/${day}`;
};

// ISO8601 → "YYYY/M/D" 表示 ※詳細画面用（元の挙動を維持）
export const formatProductBlueprintDate = (iso?: string | null): string => {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  const y = d.getFullYear();
  const m = d.getMonth() + 1;
  const day = d.getDate();
  return `${y}/${m}/${day}`;
};

// 一覧表示用のUI行モデル（API が返す形）
export type ProductBlueprintListRow = {
  id: string;
  productName: string;
  brandLabel: string;
  assigneeLabel: string;
  tagLabel: string;
  createdAt: string; // YYYY/MM/DD
  lastModifiedAt: string; // YYYY/MM/DD
};

// 詳細画面用：サイズ行モデル
export type SizeRow = {
  id: string;
  sizeLabel: string;
  chest: number;
  waist: number;
  length: number;
  shoulder: number;
};

// 詳細画面用：モデルナンバー行モデル
export type ModelNumberRow = {
  size: string;
  color: string;
  code: string;
};

/**
 * ID から ProductBlueprint を取得（現在はモック配列を探索）
 * - ソフトデリート済み（deletedAt が truthy）のものは取得対象外
 */
export function fetchProductBlueprintById(
  blueprintId?: string,
): ProductBlueprint | undefined {
  if (!blueprintId) return undefined;
  return (PRODUCT_BLUEPRINTS as ProductBlueprint[]).find(
    (pb) => pb.id === blueprintId && !pb.deletedAt,
  );
}

/**
 * 詳細画面用：サイズ行データを取得（現在は SIZE_VARIATIONS から復元）
 */
export function fetchProductBlueprintSizeRows(): SizeRow[] {
  return SIZE_VARIATIONS.map((v, i) => ({
    id: String(i + 1),
    sizeLabel: v.size,
    width: v.measurements["身幅"] ?? 0,
    chest: v.measurements["胸囲"] ?? 0,
    waist: v.measurements["ウエスト"] ?? 0,
    length: v.measurements["着丈"] ?? 0,
    shoulder: v.measurements["肩幅"] ?? 0,
  }));
}

/**
 * 詳細画面用：モデルナンバー行データを取得（現在は MODEL_NUMBERS から復元）
 */
export function fetchProductBlueprintModelNumberRows(): ModelNumberRow[] {
  return MODEL_NUMBERS.map((m) => ({
    size: m.size,
    color: m.color,
    code: m.modelNumber,
  }));
}

/* =========================================================
 * 作成系 API（createProductBlueprint + variations 作成）
 * =======================================================*/

// ProductBlueprint 作成時の入力パラメータ
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
  sizes: CatalogSizeRow[];
  modelNumbers: ModelNumber[];

  // ColorVariationCard から渡される color 名 → HEX(RGB) のマップ
  // 例: { "グリーン": "#417505" }
  colorRgbMap?: Record<string, string>;
};

// backend から返ってくる ProductBlueprint 作成レスポンス
export type ProductBlueprintResponse = {
  ID?: string;
  id?: string;
  productBlueprintId?: string;
  [key: string]: unknown;
};

/**
 * measurements 部分の型
 * - modelCreateService.ts 側と同じく、MeasurementKey をキーにしたマップ
 */
export type NewModelVariationMeasurements = Partial<
  Record<MeasurementKey, number | null>
>;

/**
 * ModelVariation 用 Payload
 *
 * - modelCreateService.ts 側の NewModelVariationPayload と構造互換
 */
export type NewModelVariationPayload = {
  sizeLabel: string;
  color: string;
  rgb?: number; // 色の RGB 値（0xRRGGBB）
  modelNumber: string;
  createdBy: string;
  measurements: NewModelVariationMeasurements;
};

/**
 * ProductBlueprint + ModelVariations をまとめて作成する API 呼び出し
 *
 * - ProductBlueprint 自体の作成は createProductBlueprintHTTP に委譲
 * - 生成された productBlueprintId を使って
 *   createModelVariationsFromProductBlueprint を呼び出す
 */
export async function createProductBlueprintApi(
  params: CreateProductBlueprintParams,
  variations: NewModelVariationPayload[],
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
      "[productBlueprintApi] productBlueprintId not found in response; skip ModelVariation creation",
      json,
    );
    return json;
  }

  // 🔍 backend（/models/{productBlueprintId}/variations）に渡す直前の payload 全体をログ出力
  console.log(
    "[productBlueprintApi] variations payload for backend",
    {
      productBlueprintId,
      variations,
    },
  );

  // 3. variations がある場合のみ ModelVariation を作成
  if (variations.length > 0) {
    await createModelVariationsFromProductBlueprint({
      productBlueprintId,
      variations,
    });
  } else {
    console.log(
      "[productBlueprintApi] no variations to create; variations array is empty",
    );
  }

  return json;
}
