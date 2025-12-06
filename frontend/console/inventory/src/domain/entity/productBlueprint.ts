// frontend/console/inventory/src/domain/entity/productBlueprint.ts

/**
 * ItemType
 * backend/internal/domain/productBlueprint/entity.go の ItemType に対応。
 */
export type ItemType = "tops" | "bottoms" | "other";

/**
 * ProductIDTagType
 * backend/internal/domain/productBlueprint/entity.go の ProductIDTagType (= string alias) に対応。
 */
export type ProductIDTagType = "qr" | "nfc";

/**
 * printed 状態
 * backend/internal/domain/productBlueprint/entity.go の PrintedStatus に対応。
 * ""（未設定）はフロントでは扱わず、"notYet" / "printed" のみを想定。
 */
export type PrintedStatus = "notYet" | "printed";

/**
 * ProductIDTag
 * backend/internal/domain/productBlueprint/entity.go の ProductIDTag に対応。
 *
 */
export interface ProductIDTag {
  type: ProductIDTagType;
}

/**
 * ModelVariation
 * backend/internal/domain/model/ModelVariation に対応するフロント側定義。
 */
export interface ModelVariation {
  id: string;
  name?: string;
  [key: string]: unknown;
}

/**
 * ProductBlueprint
 * backend/internal/domain/productBlueprint/entity.go の ProductBlueprint に対応。
 */
export interface ProductBlueprint {
  id: string;
  productName: string;
  brandId: string;

  /** backend の ItemType に対応（tops / bottoms / other） */
  itemType: ItemType;

  /** variationIds 削除に伴い該当要素も削除 */

  fit: string;
  material: string;

  /** 重量(kg等)。0以上 */
  weight: number;

  /** 品質保証に関するメモ／タグ一覧（空文字なし） */
  qualityAssurance: string[];

  /** 製品IDタグ情報（必須, type は qr/nfc） */
  productIdTag: ProductIDTag;

  /** 会社 ID （backend: CompanyID） */
  companyId: string;

  /** 担当者 Member ID（必須） */
  assigneeId: string;

  /** printed フラグ ("notYet" | "printed") */
  printed?: PrintedStatus;

  /** 作成者 Member ID（任意, backend: CreatedBy） */
  createdBy?: string | null;

  /** 作成日時 (ISO8601, backend: CreatedAt) */
  createdAt: string;

  /** 最終更新者 Member ID（任意, backend: UpdatedBy） */
  updatedBy?: string | null;

  /** 最終更新日時 (ISO8601, backend: UpdatedAt) */
  updatedAt: string;

  /** 削除者 Member ID（任意, backend: DeletedBy, 未削除時は null/undefined） */
  deletedBy?: string | null;

  /** 削除日時 (ISO8601, backend: DeletedAt, 未削除時は null/undefined) */
  deletedAt?: string | null;
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
  value: string,
): value is ProductIDTagType {
  return value === "qr" || value === "nfc";
}

/** PrintedStatus の妥当性チェック */
export function isValidPrintedStatus(
  value: string,
): value is PrintedStatus {
  return value === "notYet" || value === "printed";
}

/** ProductIDTag の簡易バリデーション（LogoDesignFile 削除対応） */
export function validateProductIDTag(tag: ProductIDTag): string[] {
  const errors: string[] = [];
  if (!isValidProductIDTagType(tag.type)) {
    errors.push("productIdTag.type must be 'qr' or 'nfc'");
  }
  return errors;
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
 * variationIds 削除済みのため関連チェックも削除。
 */
export function validateProductBlueprint(
  pb: ProductBlueprint,
): string[] {
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

  if (!pb.companyId?.trim()) {
    errors.push("companyId is required");
  }

  errors.push(...validateProductIDTag(pb.productIdTag));

  if (!pb.assigneeId?.trim()) {
    errors.push("assigneeId is required");
  }

  if (!pb.createdAt?.trim()) {
    errors.push("createdAt is required");
  }

  // printed が渡されている場合だけ妥当性チェック
  if (pb.printed !== undefined && pb.printed !== null) {
    if (!isValidPrintedStatus(pb.printed)) {
      errors.push("printed must be 'notYet' or 'printed'");
    }
  }

  return errors;
}

/**
 * ファクトリ: 入力値を正規化しつつ ProductBlueprint を生成。
 * variationIds 削除に伴いロジックから除外。
 */
export function createProductBlueprint(
  input: Omit<
    ProductBlueprint,
    "qualityAssurance" | "updatedAt" | "updatedBy" | "deletedAt" | "deletedBy"
  > & {
    qualityAssurance?: string[];
    updatedAt?: string;
    updatedBy?: string | null;
    deletedAt?: string | null;
    deletedBy?: string | null;
  },
): ProductBlueprint {
  const qualityAssurance = normalizeQualityAssurance(
    input.qualityAssurance ?? [],
  );

  const updatedAt =
    input.updatedAt && input.updatedAt.trim()
      ? input.updatedAt
      : input.createdAt;

  const updatedBy =
    input.updatedBy !== undefined
      ? input.updatedBy
      : input.createdBy ?? null;

  const deletedAt =
    input.deletedAt !== undefined ? input.deletedAt : null;

  const deletedBy =
    input.deletedBy !== undefined ? input.deletedBy : null;

  return {
    ...input,
    qualityAssurance,
    updatedAt,
    updatedBy,
    deletedAt,
    deletedBy,
  };
}
