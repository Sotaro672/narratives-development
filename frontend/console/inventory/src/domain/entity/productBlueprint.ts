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
 * backend は Printed: bool のため、フロントでは boolean として扱う。
 * - true  => 印刷済み
 * - false => 未印刷
 *
 * ※ 旧 PrintedStatus("notYet" | "printed") は廃止
 */
export type PrintedStatus = boolean;

/**
 * ProductIDTag
 * backend/internal/domain/productBlueprint/entity.go の ProductIDTag に対応。
 */
export interface ProductIDTag {
  type: ProductIDTagType;
}

/**
 * ModelRef
 * backend/internal/domain/productBlueprint/entity.go の ModelRef に対応。
 * - modelId: model テーブル docId
 * - displayOrder: 表示順 (1..N)
 */
export interface ModelRef {
  modelId: string;
  displayOrder: number;
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
  companyId: string;
  brandId: string;

  /** backend の ItemType に対応（tops / bottoms / other） */
  itemType: ItemType;

  fit: string;
  material: string;

  /** 重量(kg等)。0以上 */
  weight: number;

  /** 品質保証に関するメモ／タグ一覧（空文字なし、重複なし） */
  qualityAssurance: string[];

  /** 製品IDタグ情報（必須, type は qr/nfc） */
  productIdTag: ProductIDTag;

  /** 担当者 Member ID（必須） */
  assigneeId: string;

  /**
   * model 参照（表示順つき）
   * backend: ModelRefs []ModelRef
   * - 空は許容
   */
  modelRefs?: ModelRef[];

  /**
   * 印刷状態
   * backend: Printed bool
   */
  printed: boolean;

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

  /**
   * 物理削除予定日時 (ISO8601, backend: ExpireAt, 未設定時は null/undefined)
   * Firestore TTL 対象フィールド
   */
  expireAt?: string | null;
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

/** Printed (bool) の妥当性チェック */
export function isValidPrintedStatus(value: unknown): value is PrintedStatus {
  return typeof value === "boolean";
}

/** ProductIDTag の簡易バリデーション */
export function validateProductIDTag(tag: ProductIDTag): string[] {
  const errors: string[] = [];
  if (!tag) {
    errors.push("productIdTag is required");
    return errors;
  }
  if (!isValidProductIDTagType(String((tag as any).type ?? ""))) {
    errors.push("productIdTag.type must be 'qr' or 'nfc'");
  }
  return errors;
}

/** modelRefs の簡易バリデーション（空は許容） */
export function validateModelRefs(modelRefs?: ModelRef[] | null): string[] {
  const errors: string[] = [];
  if (!modelRefs || modelRefs.length === 0) return errors;

  for (let i = 0; i < modelRefs.length; i++) {
    const r = modelRefs[i];
    const id = String(r?.modelId ?? "").trim();
    const order = Number((r as any)?.displayOrder);

    if (!id) errors.push(`modelRefs[${i}].modelId is required`);
    if (!Number.isFinite(order) || order <= 0) {
      errors.push(`modelRefs[${i}].displayOrder must be > 0`);
    }
  }
  return errors;
}

/** qualityAssurance の重複/空文字を排除 */
export function normalizeQualityAssurance(xs: string[]): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const raw of xs || []) {
    const x = String(raw ?? "").trim();
    if (!x || seen.has(x)) continue;
    seen.add(x);
    out.push(x);
  }
  return out;
}

/**
 * ProductBlueprint の簡易バリデーション
 */
export function validateProductBlueprint(pb: ProductBlueprint): string[] {
  const errors: string[] = [];

  if (!pb?.id?.trim()) errors.push("id is required");
  if (!pb?.productName?.trim()) errors.push("productName is required");
  if (!pb?.brandId?.trim()) errors.push("brandId is required");

  if (!isValidItemType(String(pb?.itemType ?? ""))) {
    errors.push("itemType must be one of 'tops', 'bottoms', 'other'");
  }

  if (Number(pb?.weight ?? 0) < 0) {
    errors.push("weight must be >= 0");
  }

  if (!pb?.companyId?.trim()) {
    errors.push("companyId is required");
  }

  errors.push(...validateProductIDTag(pb.productIdTag));

  if (!pb?.assigneeId?.trim()) {
    errors.push("assigneeId is required");
  }

  if (!pb?.createdAt?.trim()) {
    errors.push("createdAt is required");
  }

  // printed は必須（backend bool）
  if (!isValidPrintedStatus((pb as any)?.printed)) {
    errors.push("printed must be boolean");
  }

  // modelRefs（任意）
  errors.push(...validateModelRefs(pb.modelRefs));

  return errors;
}

/**
 * ファクトリ: 入力値を正規化しつつ ProductBlueprint を生成。
 * - qualityAssurance: dedup/trim
 * - updatedAt: 未指定なら createdAt
 * - updatedBy: 未指定なら createdBy (なければ null)
 * - deletedAt/deletedBy/expireAt: 未指定なら null
 * - printed: 未指定なら false（backend create 時は常に false）
 */
export function createProductBlueprint(
  input: Omit<
    ProductBlueprint,
    | "qualityAssurance"
    | "updatedAt"
    | "updatedBy"
    | "deletedAt"
    | "deletedBy"
    | "expireAt"
    | "printed"
  > & {
    qualityAssurance?: string[];
    updatedAt?: string;
    updatedBy?: string | null;
    deletedAt?: string | null;
    deletedBy?: string | null;
    expireAt?: string | null;
    printed?: boolean;
    modelRefs?: ModelRef[] | null;
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
    input.updatedBy !== undefined ? input.updatedBy : input.createdBy ?? null;

  const deletedAt = input.deletedAt !== undefined ? input.deletedAt : null;
  const deletedBy = input.deletedBy !== undefined ? input.deletedBy : null;
  const expireAt = input.expireAt !== undefined ? input.expireAt : null;

  const printed = input.printed !== undefined ? !!input.printed : false;

  const modelRefs =
    input.modelRefs !== undefined && input.modelRefs !== null
      ? input.modelRefs
      : undefined;

  return {
    ...(input as any),
    qualityAssurance,
    updatedAt,
    updatedBy,
    deletedAt,
    deletedBy,
    expireAt,
    printed,
    modelRefs,
  };
}
