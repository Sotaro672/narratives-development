// frontend/console/productBlueprint/src/infrastructure/api/productBlueprintApi.ts 

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
import { createModelVariationsFromProductBlueprint } from "../../../../model/src/infrastructure/api/modelCreateApi";

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
// ★ model ドメインの SizeRow をそのまま使う
export type SizeRow = CatalogSizeRow;

// 詳細画面用：モデルナンバー行モデル
export type ModelNumberRow = {
  size: string;
  color: string;
  code: string;
};

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
  /** 新規作成時の version （基本 1 から開始） */
  version?: number;
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
    // ID が取れない場合は ModelVariation 作成をスキップ
    return json;
  }

  // 3. variations がある場合のみ ModelVariation を作成
  if (variations.length > 0) {
    await createModelVariationsFromProductBlueprint({
      productBlueprintId,
      variations,
    });
  }

  return json;
}
