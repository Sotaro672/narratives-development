// frontend/model/src/domain/entity/model.ts

/**
 * backend/internal/domain/model/entity.go に対応するフロントエンド用型定義＆ユーティリティ。
 *
 * 役割：
 * - ModelVariation / ModelData 構造の共通化
 * - バリデーション（Go側 validate() と整合）
 * - 軽量なファクトリ / 正規化ヘルパ
 */

/* =========================================================
 * 型定義 (mirror Go structs)
 * =======================================================*/

/**
 * Go: Color struct
 *   type Color struct {
 *     Name string
 *     RGB  int
 *   }
 */
export interface Color {
  /** カラー名（例: "グリーン"） */
  name: string;
  /** RGBを 0xRRGGBB などのintとして保持する想定 */
  rgb: number;
}

/**
 * 1つの具体的なバリエーション（サイズ・カラー・実測値等）
 * Go: ModelVariation
 */
export interface ModelVariation {
  modelId: string;
  productBlueprintId: string;
  modelNumber: string;
  size: string;

  /** Go: Color struct */
  color: Color;

  /**
   * 各種寸法等の数値マップ
   * Go: map[string]int を number で表現
   */
  measurements: Record<string, number>;

  /** 監査情報（任意, ISO8601） */
  createdAt?: string | null;
  createdBy?: string | null;
  updatedAt?: string | null;
  updatedBy?: string | null;
  deletedAt?: string | null;
  deletedBy?: string | null;
}

/**
 * ある ProductBlueprint の全バリエーション集合
 * Go: ModelData
 *
 *   type ModelData struct {
 *     ProductBlueprintID string
 *     Variations         []ModelVariation
 *     UpdatedAt          time.Time
 *   }
 */
export interface ModelData {
  /** ProductBlueprint.ID */
  productBlueprintId: string;

  /** バリエーション一覧 */
  variations: ModelVariation[];

  /** 最終更新日時 (ISO8601, 必須) */
  updatedAt: string;
}

/** Go 側 alias に合わせた別名 */
export type Model = ModelData;

/**
 * Go: ItemSpec
 * （UI で単一アイテム仕様を扱うときなどに利用）
 * Color は name だけあれば十分なケースが多いので string として扱う。
 */
export interface ItemSpec {
  modelNumber: string;
  size: string;
  /** カラー名（Color.name 相当） */
  color: string;
  measurements: Record<string, number>;
}

/**
 * Go: SizeVariation
 * サイズごとの代表寸法を扱う用途。
 */
export interface SizeVariation {
  id: string;
  size: string;
  measurements: Record<string, number>;
}

/**
 * Go: ModelNumber
 * サイズ＋カラーの組み合わせに対する型番。
 */
export interface ModelNumber {
  size: string;
  color: string;
  modelNumber: string;
}

/**
 * Go: ProductionQuantity
 * サイズ・カラーごとの生産数指定。
 */
export interface ProductionQuantity {
  size: string;
  color: string;
  quantity: number;
}

/* =========================================================
 * ポリシー (AllowedSizes / AllowedColors)
 * Go: AllowedSizes / AllowedColors と対応
 * - 空なら「何でもOK」
 * - 必要なら呼び出し側で埋める
 * =======================================================*/

export const AllowedSizes: Set<string> = new Set();
/**
 * Color.Name を対象にチェックする
 */
export const AllowedColors: Set<string> = new Set();

export function isSizeAllowed(size: string): boolean {
  if (!AllowedSizes.size) return !!size.trim();
  return AllowedSizes.has(size);
}

/**
 * Color.name を対象にチェックする
 */
export function isColorAllowed(colorName: string): boolean {
  if (!AllowedColors.size) return !!colorName.trim();
  return AllowedColors.has(colorName);
}

/* =========================================================
 * バリデーション (Go の validate() と整合イメージ)
 * =======================================================*/

export function validateModelVariation(mv: ModelVariation): string[] {
  const errors: string[] = [];

  if (!mv.modelId?.trim()) errors.push("modelId is required");
  if (!mv.productBlueprintId?.trim())
    errors.push("productBlueprintId is required");
  if (!mv.modelNumber?.trim()) errors.push("modelNumber is required");
  if (!mv.size?.trim()) errors.push("size is required");

  const colorName = mv.color?.name ?? "";
  if (!colorName.trim()) {
    errors.push("color.name is required");
  }

  if (!isSizeAllowed(mv.size)) {
    errors.push("size is not allowed");
  }

  if (!isColorAllowed(colorName)) {
    errors.push("color is not allowed");
  }

  // measurements: key 非空 & 値が有限数
  if (mv.measurements) {
    for (const [k, v] of Object.entries(mv.measurements)) {
      if (!k.trim()) {
        errors.push("measurements key must not be empty");
        break;
      }
      if (
        typeof v !== "number" ||
        !Number.isFinite(v) ||
        !Number.isInteger(v)
      ) {
        // Go 側は map[string]int 想定なので整数であることもチェック
        errors.push(
          `measurements['${k}'] must be a finite integer number`,
        );
        break;
      }
    }
  }

  // Go 側では createdAt / updatedAt の検証を削除しているので、
  // こちらも必須にはしない（整合のため）
  // 必要であれば UI 側の追加チェックとして拡張可能。

  return errors;
}

export function validateModelData(md: ModelData): string[] {
  const errors: string[] = [];

  if (!md.productBlueprintId?.trim())
    errors.push("productBlueprintId is required");
  if (!md.updatedAt?.trim()) errors.push("updatedAt is required");

  // variations
  const seen = new Set<string>();
  for (const v of md.variations || []) {
    errors.push(
      ...validateModelVariation(v).map(
        (e) => `variation(${v.modelId || "unknown"}): ${e}`,
      ),
    );
    if (v.productBlueprintId !== md.productBlueprintId) {
      errors.push(
        `variation(${v.modelId || "unknown"}): productBlueprintId mismatch`,
      );
    }
    if (seen.has(v.modelId)) {
      errors.push(`duplicate variation modelId: ${v.modelId}`);
    }
    seen.add(v.modelId);
  }

  return errors;
}

/* =========================================================
 * ヘルパ / ファクトリ
 * =======================================================*/

/** measurements をディープコピー */
export function cloneMeasurements(
  m: Record<string, number> | undefined | null,
): Record<string, number> {
  if (!m) return {};
  const out: Record<string, number> = {};
  for (const [k, v] of Object.entries(m)) {
    out[k] = v;
  }
  return out;
}

/** Color を正規化 */
export function normalizeColor(input: Color): Color {
  return {
    name: (input?.name ?? "").trim(),
    rgb: Number.isFinite(input?.rgb as number) ? input.rgb : 0,
  };
}

/** ModelVariation.from を意識した正規化ファクトリ */
export function createModelVariation(
  input: ModelVariation,
): ModelVariation {
  const normalized: ModelVariation = {
    ...input,
    modelId: input.modelId.trim(),
    productBlueprintId: input.productBlueprintId.trim(),
    modelNumber: input.modelNumber.trim(),
    size: input.size.trim(),
    color: normalizeColor(input.color),
    measurements: cloneMeasurements(input.measurements),
    createdAt: input.createdAt ?? null,
    createdBy: input.createdBy ?? null,
    updatedAt: input.updatedAt ?? null,
    updatedBy: input.updatedBy ?? null,
    deletedAt: input.deletedAt ?? null,
    deletedBy: input.deletedBy ?? null,
  };

  const errors = validateModelVariation(normalized);
  if (errors.length) {
    throw new Error(`Invalid ModelVariation: ${errors.join(", ")}`);
  }
  return normalized;
}

/** ModelData.from を意識した正規化ファクトリ */
export function createModelData(input: ModelData): ModelData {
  const normalized: ModelData = {
    ...input,
    productBlueprintId: input.productBlueprintId.trim(),
    updatedAt: input.updatedAt.trim(),
    variations: (input.variations || []).map(createModelVariation),
  };

  const errors = validateModelData(normalized);
  if (errors.length) {
    throw new Error(`Invalid ModelData: ${errors.join(", ")}`);
  }
  return normalized;
}

/** Go: ModelVariation.ToItemSpec 相当 */
export function toItemSpec(mv: ModelVariation): ItemSpec {
  return {
    modelNumber: mv.modelNumber,
    size: mv.size,
    color: mv.color.name,
    measurements: cloneMeasurements(mv.measurements),
  };
}
