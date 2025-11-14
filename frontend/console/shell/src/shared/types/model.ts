// frontend/shell/src/shared/types/model.ts

/**
 * 共通で利用する Model (製品バリエーション情報) の型定義。
 * backend/internal/domain/model/entity.go の Mirror。
 */

export interface ModelVariation {
  id: string;
  productBlueprintId: string;
  modelNumber: string;
  size: string;
  color: string;
  measurements: Record<string, number>;

  createdAt?: string | null;
  createdBy?: string | null;
  updatedAt?: string | null;
  updatedBy?: string | null;
  deletedAt?: string | null;
  deletedBy?: string | null;
}

/**
 * 1つの Product に紐づく全てのバリエーション集合。
 */
export interface ModelData {
  productId: string;
  productBlueprintId: string;
  variations: ModelVariation[];
  updatedAt: string;
}

/** alias: Model */
export type Model = ModelData;

/**
 * 単一アイテム仕様（size/color/測定値）
 */
export interface ItemSpec {
  modelNumber: string;
  size: string;
  color: string;
  measurements: Record<string, number>;
}

/**
 * サイズ単位の仕様。
 */
export interface SizeVariation {
  id: string;
  size: string;
  measurements: Record<string, number>;
}

/**
 * サイズ＋カラーごとの型番。
 */
export interface ModelNumber {
  size: string;
  color: string;
  modelNumber: string;
}
