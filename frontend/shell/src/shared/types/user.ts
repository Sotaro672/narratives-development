// frontend/shell/src/shared/types/user.ts

/**
 * User (shared type)
 *
 * Mirrors:
 * - backend/internal/domain/user/entity.go
 * - frontend/order/src/domain/entity/user.ts
 *
 * TS source-of-truth fields:
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
 * Domain error messages (aligned with Go)
 */
export const USER_ERRORS = {
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
 * Policy (sync with backend)
 */
export const USER_MAX_NAME_LENGTH = 100;

// E.164 (+xxxxxxxxxxxx up to 15 digits)
const E164_RE = /^\+[1-9]\d{1,14}$/;
// Local format: digits / spaces / hyphen / parens
const LOCAL_TEL_RE = /^[0-9\-\s()]{7,20}$/;

/**
 * Normalize Date | string → Date (UTC)
 */
function normalizeToDate(input: Date | string, errorMsg: string): Date {
  if (input instanceof Date) {
    if (Number.isNaN(input.getTime())) {
      throw new Error(errorMsg);
    }
    return new Date(input.toISOString());
  }

  const s = input.trim();
  if (!s) {
    throw new Error(errorMsg);
  }
  const d = new Date(s);
  if (Number.isNaN(d.getTime())) {
    throw new Error(errorMsg);
  }
  return new Date(d.toISOString());
}

/**
 * Normalize optional string (trim, empty → undefined)
 */
function normalizeOptionalString(
  v: string | null | undefined
): string | undefined {
  if (v == null) return undefined;
  const t = `${v}`.trim();
  return t === "" ? undefined : t;
}

/**
 * Name length check
 */
function validateNamePart(v: string | undefined, errorMsg: string): void {
  if (!v) return;
  if ([...v].length > USER_MAX_NAME_LENGTH) {
    throw new Error(errorMsg);
  }
}

/**
 * Email validation (lightweight, aligns roughly with net/mail-based check)
 */
function emailValid(email: string): boolean {
  const t = email.trim();
  if (!t) return false;

  const parts = t.split("@");
  if (parts.length !== 2) return false;
  if (!parts[0] || !parts[1]) return false;
  if (!parts[1].includes(".")) return false;

  return true;
}

/**
 * Phone validation (E.164 or relaxed local)
 */
function phoneValid(phone: string): boolean {
  const t = phone.trim();
  if (!t) return false;
  return E164_RE.test(t) || LOCAL_TEL_RE.test(t);
}

/**
 * Validate a User object according to backend rules.
 * Throws Error on violation.
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
    throw new Error(USER_ERRORS.invalidID);
  }

  // names
  validateNamePart(first_name, USER_ERRORS.invalidFirstName);
  validateNamePart(first_name_kana, USER_ERRORS.invalidFirstNameKana);
  validateNamePart(last_name, USER_ERRORS.invalidLastName);
  validateNamePart(last_name_kana, USER_ERRORS.invalidLastNameKana);

  // email
  if (email !== undefined) {
    if (!emailValid(email)) {
      throw new Error(USER_ERRORS.invalidEmail);
    }
  }

  // phone
  if (phone_number !== undefined) {
    if (!phoneValid(phone_number)) {
      throw new Error(USER_ERRORS.invalidPhone);
    }
  }

  // dates (backend: all 3 are required & deletedAt >= createdAt, updatedAt >= createdAt)
  const created = normalizeToDate(createdAt, USER_ERRORS.invalidCreatedAt);
  const updated = normalizeToDate(updatedAt, USER_ERRORS.invalidUpdatedAt);
  const deleted = normalizeToDate(deletedAt, USER_ERRORS.invalidDeletedAt);

  if (updated.getTime() < created.getTime()) {
    throw new Error(USER_ERRORS.invalidUpdatedAt);
  }
  if (deleted.getTime() < created.getTime()) {
    throw new Error(USER_ERRORS.invalidDeletedAt);
  }
}

/**
 * Factory helper:
 * - trims & normalizes optional fields
 * - runs domain validation
 */
export function createUser(input: User): User {
  const normalized: User = {
    id: input.id.trim(),
    first_name: normalizeOptionalString(input.first_name),
    first_name_kana: normalizeOptionalString(input.first_name_kana),
    last_name: normalizeOptionalString(input.last_name),
    last_name_kana: normalizeOptionalString(input.last_name_kana),
    email: normalizeOptionalString(input.email),
    phone_number: normalizeOptionalString(input.phone_number),
    createdAt: input.createdAt,
    updatedAt: input.updatedAt,
    deletedAt: input.deletedAt,
  };

  validateUser(normalized);
  return normalized;
}
