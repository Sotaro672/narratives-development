// frontend/shell/src/shared/types/productBlueprint.ts

/**
 * ItemType
 * backend/internal/domain/productBlueprint/entity.go の ItemType に対応。
 */
export type ItemType = "tops" | "bottoms" | "other";

/**
 * ProductIDTagType
 * backend/internal/domain/productBlueprint/entity.go の ProductIDTagType に対応。
 */
export type ProductIDTagType = "qr" | "nfc";

/**
 * ModelVariation
 * backend/internal/domain/model/ModelVariation に対応する共通定義。
 * 最低限 id が必須。それ以外は API スキーマに応じて拡張。
 */
export interface ModelVariation {
  id: string;
  name?: string;
  // 任意の追加プロパティ（色・サイズなど）は実装側で利用
  [key: string]: unknown;
}

/**
 * ProductBlueprint
 * backend/internal/domain/productBlueprint/entity.go に対応する共通型。
 *
 * - 日付は ISO8601 文字列
 * - updatedAt / updatedBy は backend の UpdatedAt / UpdatedBy に対応
 */
export interface ProductBlueprint {
  id: string;
  productName: string;
  brandId: string;

  itemType: ItemType;

  /** バリエーション（色・サイズ等） */
  variations: ModelVariation[];

  fit: string;
  material: string;

  /** 重量(kg等)。0以上 */
  weight: number;

  /** 品質保証に関するメモ／タグ一覧（空文字なし） */
  qualityAssurance: string[];

  /** 製品IDタグタイプ（qr / nfc） */
  productIdTagType: ProductIDTagType;

  /** 担当者 Member ID（必須） */
  assigneeId: string;

  /** 作成者 Member ID（任意） */
  createdBy?: string | null;

  /** 作成日時 (ISO8601) */
  createdAt: string;

  /** 最終更新者 Member ID（任意） */
  updatedBy?: string | null;

  /** 最終更新日時 (ISO8601) */
  updatedAt: string;
}

/* =========================================================
 * ユーティリティ / バリデーション
 * =======================================================*/

/** ItemType の妥当性チェック */
export function isValidItemType(value: string): value is ItemType {
  return value === "tops" || value === "bottoms" || value === "other";
}

/** ProductIDTagType の妥当性チェック */
export function isValidProductIDTagType(
  value: string
): value is ProductIDTagType {
  return value === "qr" || value === "nfc";
}

/** variations 配列を id でユニーク化＆trim する */
export function normalizeVariations(
  vars: ModelVariation[]
): ModelVariation[] {
  const seen = new Set<string>();
  const out: ModelVariation[] = [];
  for (const v of vars || []) {
    const id = (v.id ?? "").trim();
    if (!id || seen.has(id)) continue;
    seen.add(id);
    out.push({ ...v, id });
  }
  return out;
}

/** qualityAssurance の重複/空文字を排除 */
export function normalizeQualityAssurance(xs: string[]): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const raw of xs || []) {
    const x = raw.trim();
    if (!x || seen.has(x)) continue;
    seen.add(x);
    out.push(x);
  }
  return out;
}

/**
 * ProductBlueprint の簡易バリデーション
 * backend の validate() ロジックと整合させたフロント側チェック。
 */
export function validateProductBlueprint(pb: ProductBlueprint): string[] {
  const errors: string[] = [];

  if (!pb.id?.trim()) errors.push("id is required");
  if (!pb.productName?.trim()) errors.push("productName is required");
  if (!pb.brandId?.trim()) errors.push("brandId is required");

  if (!isValidItemType(pb.itemType)) {
    errors.push("itemType must be one of 'tops', 'bottoms', 'other'");
  }

  if (pb.weight < 0) {
    errors.push("weight must be >= 0");
  }

  if (!isValidProductIDTagType(pb.productIdTagType)) {
    errors.push("productIdTagType must be 'qr' or 'nfc'");
  }

  if (!pb.assigneeId?.trim()) {
    errors.push("assigneeId is required");
  }

  if (!pb.createdAt?.trim()) {
    errors.push("createdAt is required");
  }

  // variations: id が空でない & 重複なし
  const seen = new Set<string>();
  for (const v of pb.variations || []) {
    const id = (v.id ?? "").trim();
    if (!id) {
      errors.push("variation.id must not be empty");
      continue;
    }
    if (seen.has(id)) {
      errors.push(`duplicate variation id: ${id}`);
    }
    seen.add(id);
  }

  return errors;
}

/**
 * ファクトリ: 入力値を正規化しつつ ProductBlueprint を生成。
 * （モックデータ生成やフォーム初期値に利用）
 */
export function createProductBlueprint(
  input: Omit<
    ProductBlueprint,
    "variations" | "qualityAssurance" | "updatedAt" | "updatedBy"
  > & {
    variations?: ModelVariation[];
    qualityAssurance?: string[];
    updatedAt?: string;
    updatedBy?: string | null;
  }
): ProductBlueprint {
  const variations = normalizeVariations(input.variations ?? []);
  const qualityAssurance = normalizeQualityAssurance(
    input.qualityAssurance ?? []
  );

  const updatedAt =
    input.updatedAt && input.updatedAt.trim()
      ? input.updatedAt
      : input.createdAt;

  const updatedBy =
    input.updatedBy !== undefined
      ? input.updatedBy
      : input.createdBy ?? null;

  return {
    ...input,
    variations,
    qualityAssurance,
    updatedAt,
    updatedBy,
  };
}
