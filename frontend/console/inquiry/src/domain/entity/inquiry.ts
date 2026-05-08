// frontend/inquiry/src/domain/entity/inquiry.ts

/**
 * InquiryStatus / InquiryType
 * backend/internal/domain/inquiry/entity.go と対応する型。
 *
 * 現時点では Go 側も「非空であれば OK」という運用のため、
 * ここでは string エイリアスとして定義し、バリデーションで非空チェックのみ行う。
 */
export type InquiryStatus = string;
export type InquiryType = string;

/**
 * Inquiry
 * backend/internal/domain/inquiry/entity.go の Inquiry 構造体を Mirror。
 *
 * - 日付は ISO8601 文字列（例: "2025-01-10T00:00:00Z"）
 * - Optional フィールドは null/undefined を許容
 * - imageId は inquiryImage の primary key (= inquiryId) を指す想定
 */
export interface Inquiry {
  id: string;
  avatarId: string;
  subject: string;
  content: string;
  status: InquiryStatus;
  inquiryType: InquiryType;

  productBlueprintId?: string | null;
  tokenBlueprintId?: string | null;
  assigneeId?: string | null;
  imageId?: string | null;

  createdAt: string;
  updatedAt: string;
  updatedBy?: string | null;
  deletedAt?: string | null;
  deletedBy?: string | null;
}

/* =========================================================
 * Validation helpers
 * =======================================================*/

/** 簡易な日時文字列チェック（ISO8601 / Date.parse ベース） */
export function isValidDateTimeString(
  value: string | null | undefined,
): boolean {
  if (!value) return false;
  const v = value.trim();
  if (!v) return false;
  const t = Date.parse(v);
  return !Number.isNaN(t);
}

/** a <= b の順序であれば true（両方パース可能な場合のみ） */
export function isDateTimeOrderValid(
  a: string | null | undefined,
  b: string | null | undefined,
): boolean {
  if (!a || !b) return false;
  const ta = Date.parse(a);
  const tb = Date.parse(b);
  if (Number.isNaN(ta) || Number.isNaN(tb)) return false;
  return ta <= tb;
}

/**
 * Inquiry の妥当性チェック（Go 側 validate() と対応）
 * 問題があればエラーメッセージ配列を返す。
 */
export function validateInquiry(inq: Inquiry): string[] {
  const errors: string[] = [];

  // 必須文字列
  if (!inq.id?.trim()) errors.push("id is required");
  if (!inq.avatarId?.trim()) errors.push("avatarId is required");
  if (!inq.subject?.trim()) errors.push("subject is required");
  if (!inq.content?.trim()) errors.push("content is required");

  // status / inquiryType: 非空のみチェック（Go と同じ）
  if (!String(inq.status || "").trim()) {
    errors.push("status is required");
  }
  if (!String(inq.inquiryType || "").trim()) {
    errors.push("inquiryType is required");
  }

  // createdAt / updatedAt
  if (!isValidDateTimeString(inq.createdAt)) {
    errors.push("createdAt must be a valid datetime");
  }
  if (!isValidDateTimeString(inq.updatedAt)) {
    errors.push("updatedAt must be a valid datetime");
  }
  if (
    isValidDateTimeString(inq.createdAt) &&
    isValidDateTimeString(inq.updatedAt) &&
    !isDateTimeOrderValid(inq.createdAt, inq.updatedAt)
  ) {
    errors.push("updatedAt must be >= createdAt");
  }

  // updatedBy: セットされている場合は非空
  if (
    inq.updatedBy != null &&
    inq.updatedBy.trim() === ""
  ) {
    errors.push("updatedBy must not be empty when set");
  }

  // deletedAt / deletedBy: createdAt 以上 & セット時は非空
  if (inq.deletedAt != null) {
    if (!isValidDateTimeString(inq.deletedAt)) {
      errors.push("deletedAt must be a valid datetime when set");
    } else if (
      isValidDateTimeString(inq.createdAt) &&
      !isDateTimeOrderValid(inq.createdAt, inq.deletedAt)
    ) {
      errors.push("deletedAt must be >= createdAt");
    }
  }
  if (
    inq.deletedBy != null &&
    inq.deletedBy.trim() === ""
  ) {
    errors.push("deletedBy must not be empty when set");
  }

  return errors;
}

/* =========================================================
 * Normalization helpers
 * =======================================================*/

/**
 * 文字列を trim し、空文字は null に正規化。
 */
function normOpt(v: string | null | undefined): string | null {
  const t = v?.trim() ?? "";
  return t || null;
}

/**
 * Inquiry の正規化
 * - 必須文字列は trim のみ
 * - optional な文字列は trim + 空文字→null
 */
export function normalizeInquiry(input: Inquiry): Inquiry {
  return {
    ...input,
    id: input.id.trim(),
    avatarId: input.avatarId.trim(),
    subject: input.subject.trim(),
    content: input.content.trim(),
    status: (input.status as string).trim(),
    inquiryType: (input.inquiryType as string).trim(),
    productBlueprintId: normOpt(input.productBlueprintId),
    tokenBlueprintId: normOpt(input.tokenBlueprintId),
    assigneeId: normOpt(input.assigneeId),
    imageId: normOpt(input.imageId),
    createdAt: input.createdAt.trim(),
    updatedAt: input.updatedAt.trim(),
    updatedBy: normOpt(input.updatedBy),
    deletedAt: normOpt(input.deletedAt),
    deletedBy: normOpt(input.deletedBy),
  };
}

/* =========================================================
 * Behavior helpers (TS-side)
 * =======================================================*/

/**
 * Go の (*Inquiry).Touch(now) に対応するユーティリティ。
 * - now が不正な場合はエラーを投げる。
 */
export function touchInquiry(
  inq: Inquiry,
  now: Date = new Date(),
): Inquiry {
  if (!(now instanceof Date) || Number.isNaN(now.getTime())) {
    throw new Error("invalid updatedAt");
  }
  return {
    ...inq,
    updatedAt: now.toISOString(),
  };
}
