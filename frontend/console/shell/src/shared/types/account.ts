// frontend/shell/src/shared/types/account.ts

/**
 * AccountStatus
 * backend/internal/domain/account/entity.go の AccountStatus に対応。
 *
 * - "active"    : 利用中
 * - "inactive"  : 未利用 / 一時未使用
 * - "suspended" : 利用停止
 * - "deleted"   : 論理削除
 */
export type AccountStatus = "active" | "inactive" | "suspended" | "deleted";

/** AccountStatus の妥当性チェック */
export function isValidAccountStatus(s: string): s is AccountStatus {
  return (
    s === "active" ||
    s === "inactive" ||
    s === "suspended" ||
    s === "deleted"
  );
}

/**
 * AccountType
 * backend/internal/domain/account/entity.go の AccountType に対応。
 *
 * - "普通"
 * - "当座"
 */
export type AccountType = "普通" | "当座";

/** AccountType の妥当性チェック */
export function isValidAccountType(t: string): t is AccountType {
  return t === "普通" || t === "当座";
}

/**
 * Account
 * backend/internal/domain/account/entity.go の Account に対応。
 *
 * - 日付は ISO8601 文字列（例: "2025-01-10T00:00:00Z"）を想定
 * - *_by 系は省略可能
 * - deletedAt は論理削除時のみ設定
 */
export interface Account {
  id: string;
  memberId: string;
  bankName: string;
  branchName: string;
  accountNumber: number; // 0..99,999,999
  accountType: AccountType;
  currency: string; // デフォルト "円"
  status: AccountStatus;
  createdAt: string;
  createdBy?: string | null;
  updatedAt: string;
  updatedBy?: string | null;
  deletedAt?: string | null;
  deletedBy?: string | null;
}

/**
 * Policy (backend と同期させる定数群)
 * backend/internal/domain/account/entity.go の Policy 相当。
 */
export const ACCOUNT_ID_PREFIX = "account_";
export const DEFAULT_CURRENCY = "円";
export const MAX_BANK_NAME_LENGTH = 50;
export const MAX_BRANCH_NAME_LENGTH = 50;

// accountNumber: 0..99,999,999
export const MIN_ACCOUNT_NUMBER = 0;
export const MAX_ACCOUNT_NUMBER = 99_999_999;

// MemberID length limit（backend と揃える）
export const MAX_MEMBER_ID_LENGTH = 100;
// 後方互換 alias
export const MAX_BRAND_NAME_LENGTH = MAX_MEMBER_ID_LENGTH;

/**
 * 表示用の口座名義
 * backend の Account.AccountHolderName() と同様に memberId をそのまま利用。
 */
export function getAccountHolderName(
  account: Pick<Account, "memberId">
): string {
  return account.memberId;
}

/**
 * Account の簡易バリデーション
 * backend/internal/domain/account/entity.go の validate() と整合する範囲で
 * フロントエンド側チェックを行う。
 */
export function validateAccount(a: Account): boolean {
  // id
  if (!a.id) return false;
  if (!a.id.startsWith(ACCOUNT_ID_PREFIX)) return false;

  // memberId
  if (!a.memberId) return false;
  if (
    MAX_MEMBER_ID_LENGTH > 0 &&
    [...a.memberId].length > MAX_MEMBER_ID_LENGTH
  ) {
    return false;
  }

  // bankName
  if (!a.bankName) return false;
  if (
    MAX_BANK_NAME_LENGTH > 0 &&
    [...a.bankName].length > MAX_BANK_NAME_LENGTH
  ) {
    return false;
  }

  // branchName
  if (!a.branchName) return false;
  if (
    MAX_BRANCH_NAME_LENGTH > 0 &&
    [...a.branchName].length > MAX_BRANCH_NAME_LENGTH
  ) {
    return false;
  }

  // accountNumber
  if (
    typeof a.accountNumber !== "number" ||
    !Number.isInteger(a.accountNumber) ||
    a.accountNumber < MIN_ACCOUNT_NUMBER ||
    a.accountNumber > MAX_ACCOUNT_NUMBER
  ) {
    return false;
  }

  // accountType
  if (!isValidAccountType(a.accountType)) {
    return false;
  }

  // currency
  if (!a.currency || !a.currency.toString().trim()) {
    return false;
  }

  // status
  if (!isValidAccountStatus(a.status)) {
    return false;
  }

  // createdAt / updatedAt
  if (!a.createdAt || Number.isNaN(Date.parse(a.createdAt))) {
    return false;
  }
  if (!a.updatedAt || Number.isNaN(Date.parse(a.updatedAt))) {
    return false;
  }

  // deletedAt がある場合は形式のみ確認
  if (
    a.deletedAt != null &&
    a.deletedAt !== "" &&
    Number.isNaN(Date.parse(a.deletedAt))
  ) {
    return false;
  }

  return true;
}

/**
 * GraphQL / フォーム入力用 DTO
 * 新規作成・更新時に利用する軽量型
 */
export interface AccountInput {
  id?: string;
  memberId: string;
  bankName: string;
  branchName: string;
  accountNumber: number;
  accountType: AccountType;
  currency?: string; // 未指定時は DEFAULT_CURRENCY
  status?: AccountStatus; // 未指定時は backend 側デフォルトに委譲
}
