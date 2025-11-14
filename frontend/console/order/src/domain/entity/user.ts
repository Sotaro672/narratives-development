// frontend/order/src/domain/entity/user.ts

/**
 * User domain entity (frontend)
 *
 * Mirrors backend/internal/domain/user/entity.go and
 * web-app/src/shared/types/user.ts (TS is the source of truth).
 *
 * TS fields:
 * - id: string
 * - first_name?: string
 * - first_name_kana?: string
 * - last_name_kana?: string
 * - last_name?: string
 * - email?: string
 * - phone_number?: string
 * - createdAt: Date | string
 * - updatedAt: Date | string
 * - deletedAt: Date | string
 */

/**
 * Runtime representation used across the app.
 * Date 型/ISO文字列の両方を許容しておき、必要に応じて正規化して利用する想定。
 */
export interface User {
  id: string;
  first_name?: string;
  first_name_kana?: string;
  last_name_kana?: string;
  last_name?: string;
  email?: string;
  phone_number?: string;
  createdAt: Date | string;
  updatedAt: Date | string;
  deletedAt: Date | string;
}

/**
 * ドメインエラー定義（メッセージは Go 実装に揃える）
 */
export const UserErrors = {
  invalidID: "user: invalid id",
  invalidFirstName: "user: invalid first_name",
  invalidFirstNameKana: "user: invalid first_name_kana",
  invalidLastNameKana: "user: invalid last_name_kana",
  invalidLastName: "user: invalid last_name",
  invalidEmail: "user: invalid email",
  invalidPhone: "user: invalid phone_number",
  invalidCreatedAt: "user: invalid createdAt",
  invalidUpdatedAt: "user: invalid updatedAt",
  invalidDeletedAt: "user: invalid deletedAt",
} as const;

/**
 * ポリシー (Go 実装と概ね同期)
 */
export const USER_MAX_NAME_LENGTH = 100;

// E.164 (+xxxxxxxxxxxx up to 15 digits)
const E164_RE = /^\+[1-9]\d{1,14}$/;
// 緩めの国内/ローカル表記 (数字/空白/ハイフン/括弧)
const LOCAL_TEL_RE = /^[0-9\-\s()]{7,20}$/;

/**
 * ヘルパー: 文字列 or Date を UTC ISO8601 文字列へ正規化
 */
function normalizeDate(input: Date | string): Date {
  if (input instanceof Date) {
    if (Number.isNaN(input.getTime())) {
      throw new Error(UserErrors.invalidCreatedAt);
    }
    return new Date(input.toISOString());
  }

  const trimmed = input.trim();
  if (!trimmed) {
    throw new Error(UserErrors.invalidCreatedAt);
  }
  const d = new Date(trimmed);
  if (Number.isNaN(d.getTime())) {
    throw new Error(UserErrors.invalidCreatedAt);
  }
  return new Date(d.toISOString());
}

/**
 * ヘルパー: optional string 正規化（空文字なら undefined）
 */
function normalizeOptionalString(v: string | undefined | null): string | undefined {
  if (v == null) return undefined;
  const t = `${v}`.trim();
  return t === "" ? undefined : t;
}

/**
 * ヘルパー: 名前長さチェック
 */
function validateNamePart(
  v: string | undefined,
  errorMessage: string
): void {
  if (v && [...v].length > USER_MAX_NAME_LENGTH) {
    throw new Error(errorMessage);
  }
}

/**
 * ヘルパー: メール形式チェック（net/mail 相当の緩め判定）
 */
function emailValid(email: string): boolean {
  const t = email.trim();
  if (!t) return false;
  // 簡易: 「@を1つ含み、ローカル/ドメインが空でない」
  const parts = t.split("@");
  if (parts.length !== 2) return false;
  if (!parts[0] || !parts[1]) return false;
  if (!parts[1].includes(".")) return false;
  return true;
}

/**
 * ヘルパー: 電話番号形式チェック
 */
function phoneValid(phone: string): boolean {
  const t = phone.trim();
  if (!t) return false;
  return E164_RE.test(t) || LOCAL_TEL_RE.test(t);
}

/**
 * ドメインバリデーション
 * - backend/internal/domain/user/entity.go の validate() と同等の制約を再現
 */
export function validateUser(user: User): void {
  const {
    id,
    first_name,
    first_name_kana,
    last_name,
    last_name_kana,
    email,
    phone_number,
    createdAt,
    updatedAt,
    deletedAt,
  } = user;

  // id
  if (!id || !id.trim()) {
    throw new Error(UserErrors.invalidID);
  }

  // names
  validateNamePart(first_name, UserErrors.invalidFirstName);
  validateNamePart(first_name_kana, UserErrors.invalidFirstNameKana);
  validateNamePart(last_name, UserErrors.invalidLastName);
  validateNamePart(last_name_kana, UserErrors.invalidLastNameKana);

  // email
  if (email !== undefined) {
    if (!emailValid(email)) {
      throw new Error(UserErrors.invalidEmail);
    }
  }

  // phone
  if (phone_number !== undefined) {
    if (!phoneValid(phone_number)) {
      throw new Error(UserErrors.invalidPhone);
    }
  }

  // dates (Go 実装に合わせ deletedAt も必須 & createdAt 以降)
  const created = normalizeDate(createdAt);
  const updated = normalizeDate(updatedAt);
  const deleted = normalizeDate(deletedAt);

  if (updated.getTime() < created.getTime()) {
    throw new Error(UserErrors.invalidUpdatedAt);
  }

  if (deleted.getTime() < created.getTime()) {
    throw new Error(UserErrors.invalidDeletedAt);
  }
}

/**
 * コンストラクタ的ヘルパー:
 * 生データを受け取り、トリムや undefined 正規化を行った User を返却。
 * バリデーションに失敗した場合は Error を投げます。
 */
export function createUser(input: User): User {
  const normalized: User = {
    id: input.id.trim(),
    first_name: normalizeOptionalString(input.first_name),
    first_name_kana: normalizeOptionalString(input.first_name_kana),
    last_name_kana: normalizeOptionalString(input.last_name_kana),
    last_name: normalizeOptionalString(input.last_name),
    email: normalizeOptionalString(input.email),
    phone_number: normalizeOptionalString(input.phone_number),
    createdAt: input.createdAt,
    updatedAt: input.updatedAt,
    deletedAt: input.deletedAt,
  };

  validateUser(normalized);
  return normalized;
}
