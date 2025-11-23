// frontend/productBlueprint/src/domain/entity/productBlueprint.ts

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
 * LogoDesignFile
 * backend/internal/domain/productBlueprint/entity.go の LogoDesignFile に対応。
 */
export interface LogoDesignFile {
  name: string;
  url: string;
}

/**
 * ProductIDTag
 * backend/internal/domain/productBlueprint/entity.go の ProductIDTag に対応。
 */
export interface ProductIDTag {
  type: ProductIDTagType;
  logoDesignFile?: LogoDesignFile | null;
}

/**
 * ModelVariation
 * backend/internal/domain/model/ModelVariation に対応するフロント側定義。
 * Go側では ProductBlueprint には VariationIDs（string 配列）のみを保持し、
 * 実体は Model 側で管理する想定。
 * フロントでは必要に応じて ModelVariation[] を別 API から取得して組み合わせる。
 */
export interface ModelVariation {
  id: string;
  name?: string;
  // 任意の追加プロパティ（色・サイズなど）は API に合わせて利用
  [key: string]: unknown;
}

/**
 * ProductBlueprint
 * backend/internal/domain/productBlueprint/entity.go の ProductBlueprint に対応。
 *
 * - 日付は ISO8601 文字列として扱う
 * - updatedAt / updatedBy は backend の UpdatedAt / UpdatedBy に対応
 * - deletedAt / deletedBy は backend の DeletedAt / DeletedBy に対応（ソフトデリート）
 */
export interface ProductBlueprint {
  id: string;
  productName: string;
  brandId: string;

  /** backend の ItemType に対応（tops / bottoms / other） */
  itemType: ItemType;

  /**
   * モデルIDの配列。
   * backend の VariationIDs ([]string) に対応。
   */
  variationIds: string[];

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

/** LogoDesignFile の簡易バリデーション */
export function validateLogoDesignFile(file: LogoDesignFile): string[] {
  const errors: string[] = [];
  if (!file.name?.trim()) {
    errors.push("logoDesignFile.name is required");
  }
  try {
    // URL コンストラクタでざっくり検証
    new URL(file.url);
  } catch {
    errors.push("logoDesignFile.url must be a valid URL");
  }
  return errors;
}

/** ProductIDTag の簡易バリデーション */
export function validateProductIDTag(tag: ProductIDTag): string[] {
  const errors: string[] = [];
  if (!isValidProductIDTagType(tag.type)) {
    errors.push("productIdTag.type must be 'qr' or 'nfc'");
  }
  if (tag.logoDesignFile) {
    errors.push(...validateLogoDesignFile(tag.logoDesignFile));
  }
  return errors;
}

/**
 * ModelVariation[] を id でユニーク化＆trim するユーティリティ。
 * （Model 側の UI 等で引き続き利用可能）
 */
export function normalizeVariations(
  vars: ModelVariation[],
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

/** variationIds の重複/空文字を排除（backend の dedupTrim 相当） */
export function normalizeVariationIds(ids: string[]): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const raw of ids || []) {
    const x = raw.trim();
    if (!x || seen.has(x)) continue;
    seen.add(x);
    out.push(x);
  }
  return out;
}

/** qualityAssurance の重複/空文字を排除（backend の dedupTrim 相当） */
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
 * backend の validate() ロジックと整合するようにチェック。
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

  // variationIds: 空文字でない & 重複なし（normalize前提だが念のためチェック）
  const seen = new Set<string>();
  for (const rawId of pb.variationIds || []) {
    const id = rawId.trim();
    if (!id) {
      errors.push("variationIds must not contain empty id");
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
 * （フロント側でモック生成やフォーム初期値に利用）
 */
export function createProductBlueprint(
  input: Omit<
    ProductBlueprint,
    | "variationIds"
    | "qualityAssurance"
    | "updatedAt"
    | "updatedBy"
    | "deletedAt"
    | "deletedBy"
  > & {
    variationIds?: string[];
    qualityAssurance?: string[];
    updatedAt?: string;
    updatedBy?: string | null;
    deletedAt?: string | null;
    deletedBy?: string | null;
  },
): ProductBlueprint {
  const variationIds = normalizeVariationIds(input.variationIds ?? []);
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
    variationIds,
    qualityAssurance,
    updatedAt,
    updatedBy,
    deletedAt,
    deletedBy,
  };
}
