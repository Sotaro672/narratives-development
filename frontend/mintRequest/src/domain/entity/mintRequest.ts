// frontend/mintRequest/src/domain/entity/mintRequest.ts

/**
 * MintRequestStatus
 * backend/internal/domain/mintRequest/entity.go の MintRequestStatus に対応。
 *
 * - "planning"  : 申請前の計画状態
 * - "requested" : ミント申請済み（実行待ち）
 * - "minted"    : ミント実行済み
 */
export type MintRequestStatus = "planning" | "requested" | "minted";

/**
 * MintRequest
 * backend/internal/domain/mintRequest/entity.go の MintRequest に対応。
 *
 * - 日付は ISO8601 文字列として表現（例: "2025-01-01T00:00:00Z"）
 * - burnDate は日付 or 日時文字列を許容（Go 側は日付/日時混在を parseTime で吸収）
 * - requestedBy / requestedAt / mintedAt / deletedAt / deletedBy は null で未設定
 */
export interface MintRequest {
  id: string;
  tokenBlueprintId: string;
  productionId: string;
  mintQuantity: number;

  /** 焼却予定日（"YYYY-MM-DD" または ISO8601）null で未設定 */
  burnDate: string | null;

  status: MintRequestStatus;

  /** 申請者 Member ID（requested 状態以降で必須） */
  requestedBy: string | null;
  /** 申請日時（requested 状態以降で必須） */
  requestedAt: string | null;

  /** ミント実行日時（minted 状態のみ必須） */
  mintedAt?: string | null;

  /** 作成情報（必須） */
  createdAt: string;
  createdBy: string;

  /** 更新情報（必須） */
  updatedAt: string;
  updatedBy: string;

  /** 論理削除情報（両方 null か、両方非 null のペア） */
  deletedAt: string | null;
  deletedBy: string | null;
}

/* =========================================================
 * ユーティリティ
 * =======================================================*/

/** ステータス妥当性チェック（Go の IsValidStatus に対応） */
export function isValidMintRequestStatus(
  s: string,
): s is MintRequestStatus {
  return s === "planning" || s === "requested" || s === "minted";
}

/** ISO8601/日付文字列の簡易チェック（空文字は非許容） */
function isValidDateTimeString(value: string | null | undefined): boolean {
  if (value == null) return false;
  const v = value.trim();
  if (!v) return false;
  const t = Date.parse(v);
  return !Number.isNaN(t);
}

/** 2つの日時文字列の順序比較（a <= b であれば true, どちらか不正なら false） */
function isDateTimeOrderValid(
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
 * MintRequest の簡易バリデーション
 * backend/internal/domain/mintRequest/entity.go の validate() ロジックと概ね対応。
 *
 * 問題があればエラーメッセージ配列を返す。
 */
export function validateMintRequest(
  mr: MintRequest,
): string[] {
  const errors: string[] = [];

  // 基本必須
  if (!mr.id?.trim()) errors.push("id is required");
  if (!mr.tokenBlueprintId?.trim()) {
    errors.push("tokenBlueprintId is required");
  }
  if (!mr.productionId?.trim()) {
    errors.push("productionId is required");
  }
  if (mr.mintQuantity == null || mr.mintQuantity <= 0) {
    errors.push("mintQuantity must be > 0");
  }

  // burnDate（存在する場合のみチェック）
  if (mr.burnDate !== null) {
    if (!isValidDateTimeString(mr.burnDate)) {
      errors.push("burnDate must be a valid date/datetime string or null");
    }
  }

  // ステータス
  if (!isValidMintRequestStatus(mr.status)) {
    errors.push("status must be one of 'planning' | 'requested' | 'minted'");
  }

  // ステータスごとの整合性
  const hasRequestedBy = !!mr.requestedBy?.trim();
  const hasRequestedAt = !!mr.requestedAt?.trim();
  const hasMintedAt =
    mr.mintedAt != null && mr.mintedAt.toString().trim() !== "";

  if (mr.status === "planning") {
    if (hasRequestedBy || hasRequestedAt || hasMintedAt) {
      errors.push(
        "planning status must not have requestedBy/requestedAt/mintedAt",
      );
    }
  }

  if (mr.status === "requested") {
    if (!hasRequestedBy) {
      errors.push(
        "requested status requires requestedBy to be non-empty",
      );
    }
    if (!hasRequestedAt) {
      errors.push(
        "requested status requires requestedAt to be non-empty",
      );
    } else if (!isValidDateTimeString(mr.requestedAt)) {
      errors.push("requestedAt must be a valid datetime string");
    }
    if (hasMintedAt) {
      errors.push(
        "requested status must not have mintedAt (use 'minted' status)",
      );
    }
  }

  if (mr.status === "minted") {
    if (!hasRequestedBy) {
      errors.push(
        "minted status requires requestedBy to be non-empty",
      );
    }
    if (!hasRequestedAt) {
      errors.push(
        "minted status requires requestedAt to be non-empty",
      );
    } else if (!isValidDateTimeString(mr.requestedAt)) {
      errors.push("requestedAt must be a valid datetime string");
    }
    if (!hasMintedAt) {
      errors.push(
        "minted status requires mintedAt to be non-empty",
      );
    } else if (!isValidDateTimeString(mr.mintedAt!)) {
      errors.push("mintedAt must be a valid datetime string");
    } else if (
      !isDateTimeOrderValid(mr.requestedAt!, mr.mintedAt!)
    ) {
      errors.push("mintedAt must be >= requestedAt");
    }
  }

  // 作成・更新必須
  if (!mr.createdBy?.trim()) {
    errors.push("createdBy is required");
  }
  if (!isValidDateTimeString(mr.createdAt)) {
    errors.push("createdAt is required and must be a valid datetime");
  }
  if (!mr.updatedBy?.trim()) {
    errors.push("updatedBy is required");
  }
  if (!isValidDateTimeString(mr.updatedAt)) {
    errors.push("updatedAt is required and must be a valid datetime");
  }
  if (
    isValidDateTimeString(mr.createdAt) &&
    isValidDateTimeString(mr.updatedAt) &&
    !isDateTimeOrderValid(mr.createdAt, mr.updatedAt)
  ) {
    errors.push("updatedAt must be >= createdAt");
  }

  // deletedAt / deletedBy のペア整合性
  const hasDeletedAt =
    mr.deletedAt != null && mr.deletedAt.toString().trim() !== "";
  const hasDeletedBy =
    mr.deletedBy != null && mr.deletedBy.toString().trim() !== "";

  if (hasDeletedAt !== hasDeletedBy) {
    errors.push(
      "deletedAt and deletedBy must be both set or both null",
    );
  }
  if (hasDeletedAt && !isValidDateTimeString(mr.deletedAt)) {
    errors.push("deletedAt must be a valid datetime string");
  }
  if (
    hasDeletedAt &&
    isValidDateTimeString(mr.createdAt) &&
    !isDateTimeOrderValid(mr.createdAt, mr.deletedAt)
  ) {
    errors.push("deletedAt must be >= createdAt");
  }

  return errors;
}

/**
 * MintRequest の正規化用ヘルパ
 * - 文字列を trim
 * - 空文字の任意フィールドは null に丸める
 * - status 未指定時は "planning"
 * - バリデーションエラー時は例外を投げる
 */
export function createMintRequest(input: MintRequest): MintRequest {
  const normalizeOpt = (
    v: string | null | undefined,
  ): string | null => {
    const t = v?.trim() ?? "";
    return t ? t : null;
  };

  const normalized: MintRequest = {
    ...input,
    id: input.id.trim(),
    tokenBlueprintId: input.tokenBlueprintId.trim(),
    productionId: input.productionId.trim(),
    mintQuantity: input.mintQuantity,
    burnDate: normalizeOpt(input.burnDate),

    status: input.status || "planning",

    requestedBy: normalizeOpt(input.requestedBy),
    requestedAt: normalizeOpt(input.requestedAt),
    mintedAt:
      input.mintedAt === undefined
        ? undefined
        : normalizeOpt(input.mintedAt),

    createdAt: input.createdAt.trim(),
    createdBy: input.createdBy.trim(),
    updatedAt: input.updatedAt.trim(),
    updatedBy: input.updatedBy.trim(),

    deletedAt: normalizeOpt(input.deletedAt),
    deletedBy: normalizeOpt(input.deletedBy),
  };

  const errors = validateMintRequest(normalized);
  if (errors.length > 0) {
    throw new Error(
      `Invalid MintRequest: ${errors.join(", ")}`,
    );
  }

  return normalized;
}
