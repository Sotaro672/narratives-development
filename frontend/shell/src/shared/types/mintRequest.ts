// frontend/shell/src/shared/types/mintRequest.ts

/**
 * 共通で利用する MintRequest (NFT ミント申請) の型定義。
 * backend/internal/domain/mintRequest/entity.go の構造を Mirror。
 */

/**
 * MintRequestStatus
 * - "planning"  : 申請前（数量や設計を検討中）
 * - "requested" : ミント申請済（実行待ち）
 * - "minted"    : ミント完了
 */
export type MintRequestStatus = "planning" | "requested" | "minted";

/**
 * MintRequest
 * - 各日付は ISO8601 文字列を採用（例: "2025-01-10T00:00:00Z"）
 * - burnDate は日付または日時文字列、null で未設定
 * - requestedBy/requestedAt/mintedAt はステータスに応じて null もしくは必須
 */
export interface MintRequest {
  id: string;
  tokenBlueprintId: string;
  productionId: string;
  mintQuantity: number;
  burnDate: string | null;
  status: MintRequestStatus;
  requestedBy: string | null;
  requestedAt: string | null;
  mintedAt?: string | null;
  createdAt: string;
  createdBy: string;
  updatedAt: string;
  updatedBy: string;
  deletedAt: string | null;
  deletedBy: string | null;
}

/* =========================================================
 * ヘルパ関数
 * =======================================================*/

/** ステータス妥当性チェック */
export function isValidMintRequestStatus(
  s: string,
): s is MintRequestStatus {
  return s === "planning" || s === "requested" || s === "minted";
}

/** ISO8601 / 日付文字列の簡易チェック */
export function isValidDateString(value: string | null | undefined): boolean {
  if (!value) return false;
  const v = value.trim();
  if (!v) return false;
  const t = Date.parse(v);
  return !Number.isNaN(t);
}

/** 2つの日時文字列の順序比較（a <= b のとき true） */
export function isDateOrderValid(
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
 * MintRequest の簡易バリデーション。
 * 問題がある場合はエラーメッセージ配列を返す。
 */
export function validateMintRequest(mr: MintRequest): string[] {
  const errors: string[] = [];

  // 基本必須チェック
  if (!mr.id?.trim()) errors.push("id is required");
  if (!mr.tokenBlueprintId?.trim()) errors.push("tokenBlueprintId is required");
  if (!mr.productionId?.trim()) errors.push("productionId is required");
  if (mr.mintQuantity == null || mr.mintQuantity <= 0)
    errors.push("mintQuantity must be > 0");

  // burnDate
  if (mr.burnDate && !isValidDateString(mr.burnDate))
    errors.push("burnDate must be a valid date or null");

  // ステータス妥当性
  if (!isValidMintRequestStatus(mr.status))
    errors.push("status must be 'planning' | 'requested' | 'minted'");

  // ステータス別の整合性
  const hasRequestedBy = !!mr.requestedBy?.trim();
  const hasRequestedAt = !!mr.requestedAt?.trim();
  const hasMintedAt = !!mr.mintedAt?.trim();

  if (mr.status === "planning") {
    if (hasRequestedBy || hasRequestedAt || hasMintedAt)
      errors.push("planning must not include requested/minted fields");
  }

  if (mr.status === "requested") {
    if (!hasRequestedBy) errors.push("requestedBy required for requested");
    if (!hasRequestedAt) errors.push("requestedAt required for requested");
    if (hasMintedAt) errors.push("mintedAt must be null in requested");
  }

  if (mr.status === "minted") {
    if (!hasRequestedBy) errors.push("requestedBy required for minted");
    if (!hasRequestedAt) errors.push("requestedAt required for minted");
    if (!hasMintedAt) errors.push("mintedAt required for minted");
    else if (!isDateOrderValid(mr.requestedAt, mr.mintedAt))
      errors.push("mintedAt must be >= requestedAt");
  }

  // 作成・更新情報
  if (!isValidDateString(mr.createdAt))
    errors.push("createdAt must be a valid datetime");
  if (!mr.createdBy?.trim()) errors.push("createdBy is required");
  if (!isValidDateString(mr.updatedAt))
    errors.push("updatedAt must be a valid datetime");
  if (!mr.updatedBy?.trim()) errors.push("updatedBy is required");
  if (!isDateOrderValid(mr.createdAt, mr.updatedAt))
    errors.push("updatedAt must be >= createdAt");

  // 削除情報
  const hasDeletedAt = !!mr.deletedAt?.trim();
  const hasDeletedBy = !!mr.deletedBy?.trim();
  if (hasDeletedAt !== hasDeletedBy)
    errors.push("deletedAt and deletedBy must be both set or both null");
  if (hasDeletedAt && !isDateOrderValid(mr.createdAt, mr.deletedAt))
    errors.push("deletedAt must be >= createdAt");

  return errors;
}

/**
 * MintRequest の正規化（トリム + 空文字→null）
 */
export function normalizeMintRequest(input: MintRequest): MintRequest {
  const norm = (v: string | null | undefined): string | null => {
    const t = v?.trim() ?? "";
    return t || null;
  };

  return {
    ...input,
    id: input.id.trim(),
    tokenBlueprintId: input.tokenBlueprintId.trim(),
    productionId: input.productionId.trim(),
    mintQuantity: input.mintQuantity,
    burnDate: norm(input.burnDate),
    status: input.status || "planning",
    requestedBy: norm(input.requestedBy),
    requestedAt: norm(input.requestedAt),
    mintedAt: norm(input.mintedAt ?? null),
    createdAt: input.createdAt.trim(),
    createdBy: input.createdBy.trim(),
    updatedAt: input.updatedAt.trim(),
    updatedBy: input.updatedBy.trim(),
    deletedAt: norm(input.deletedAt),
    deletedBy: norm(input.deletedBy),
  };
}
