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
 * - Account は Company 配下の Stripe Connect 受取口座 
 * - 1 Company は複数 Account を持てる 
 * - 1 Account を複数 Brand が共有できる 
 * - Brand 側が accountId を保持して Account を参照する 
 * - stripeAccountId は Stripe Connected Account の acct_xxx 
 * - Stripe onboarding 中は銀行口座情報の未取得を許容する 
 * - 日付は ISO8601 文字列（例: "2025-01-10T00:00:00Z"）を想定 
 * - *_by 系は省略可能 
 * - deletedAt は論理削除時のみ設定 
 */ 
export interface Account { 
  id: string; 
  companyId: string; 
  stripeAccountId: string; 
  memberId: string; 
  bankName: string; 
  branchName: string; 
  accountNumber: number; // 0..99,999,999 
  accountType: AccountType | ""; 
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
export const MAX_COMPANY_ID_LENGTH = 100; 
export const MAX_STRIPE_ACCOUNT_ID_LENGTH = 255; 
export const MAX_BANK_NAME_LENGTH = 50; 
export const MAX_BRANCH_NAME_LENGTH = 50; 
 
// accountNumber: 0..99,999,999 
export const MIN_ACCOUNT_NUMBER = 0; 
export const MAX_ACCOUNT_NUMBER = 99_999_999; 
 
// MemberID length limit（backend と揃える） 
export const MAX_MEMBER_ID_LENGTH = 100; 
 
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
 
  // companyId 
  if (!a.companyId) return false; 
  if ( 
    MAX_COMPANY_ID_LENGTH > 0 && 
    [...a.companyId].length > MAX_COMPANY_ID_LENGTH 
  ) { 
    return false; 
  } 
 
  // stripeAccountId 
  if (!a.stripeAccountId) return false; 
  if (!a.stripeAccountId.startsWith("acct_")) return false; 
  if ( 
    MAX_STRIPE_ACCOUNT_ID_LENGTH > 0 && 
    [...a.stripeAccountId].length > MAX_STRIPE_ACCOUNT_ID_LENGTH 
  ) { 
    return false; 
  } 
 
  // memberId 
  if (!a.memberId) return false; 
  if ( 
    MAX_MEMBER_ID_LENGTH > 0 && 
    [...a.memberId].length > MAX_MEMBER_ID_LENGTH 
  ) { 
    return false; 
  } 
 
  // bankName 
  // Stripe onboarding 中は未取得を許容する 
  if ( 
    a.bankName && 
    MAX_BANK_NAME_LENGTH > 0 && 
    [...a.bankName].length > MAX_BANK_NAME_LENGTH 
  ) { 
    return false; 
  } 
 
  // branchName 
  // Stripe onboarding 中は未取得を許容する 
  if ( 
    a.branchName && 
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
  // Stripe onboarding 中は未取得を許容する 
  if (a.accountType && !isValidAccountType(a.accountType)) { 
    return false; 
  } 
 
  // currency 
  if (!a.currency) { 
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
 
  const createdAt = Date.parse(a.createdAt); 
  const updatedAt = Date.parse(a.updatedAt); 
 
  if (updatedAt < createdAt) { 
    return false; 
  } 
 
  // deletedAt がある場合は形式と createdAt 以降であることを確認 
  if ( 
    a.deletedAt != null && 
    a.deletedAt !== "" 
  ) { 
    const deletedAt = Date.parse(a.deletedAt); 
 
    if (Number.isNaN(deletedAt)) { 
      return false; 
    } 
 
    if (deletedAt < createdAt) { 
      return false; 
    } 
  } 
 
  return true; 
} 
 
/** 
 * フォーム入力用 DTO 
 * Account 新規作成・更新時に利用する軽量型。 
 * companyId は認証中ユーザーから Backend 側で決定するため含めない。 
 */ 
export interface AccountInput { 
  id?: string; 
  stripeAccountId: string; 
  memberId: string; 
  bankName: string; 
  branchName: string; 
  accountNumber: number; 
  accountType: AccountType | ""; 
  currency?: string; // 未指定時は DEFAULT_CURRENCY 
  status?: AccountStatus; // 未指定時は backend 側デフォルトに委譲 
} 