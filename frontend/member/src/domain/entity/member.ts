// frontend/shell/src/shared/types/member.ts

/**
 * MemberRole
 * backend/internal/domain/member/entity.go の MemberRole / RoleXXX と対応
 */
export const MEMBER_ROLES = [
  "admin",
  "brand-manager",
  "token-manager",
  "inquiry-handler",
  "production-designer",
] as const;

export type MemberRole = (typeof MEMBER_ROLES)[number];

/**
 * Member
 * backend の Member 構造体に対応したフロントエンド用モデル
 *
 * - 日付は ISO8601 文字列として扱う想定
 * - フロント側では camelCase を採用
 */
export interface Member {
  id: string;

  firstName?: string;
  lastName?: string;
  firstNameKana?: string;
  lastNameKana?: string;

  /**
   * 空文字 or undefined の場合は「未設定」とみなす
   */
  email?: string;

  role: MemberRole;

  /**
   * backend: Permissions []string
   * Permission.Name の配列
   */
  permissions: string[];

  /**
   * backend: AssignedBrands []string
   * ブランドID（やスラッグ）配列
   */
  assignedBrands?: string[];

  createdAt: string;          // ISO8601
  updatedAt?: string | null;  // ISO8601 or null
  updatedBy?: string | null;
  deletedAt?: string | null;  // ISO8601 or null
  deletedBy?: string | null;
}

/**
 * MemberPatch
 * 部分更新用（backend の MemberPatch に対応）
 * - undefined: 変更なし
 * - null: サーバ側実装次第だが、多くは「クリア」を意味させる場合に使用
 */
export interface MemberPatch {
  firstName?: string | null;
  lastName?: string | null;
  firstNameKana?: string | null;
  lastNameKana?: string | null;
  email?: string | null;
  role?: MemberRole;
  permissions?: string[] | null;
  assignedBrands?: string[] | null;

  createdAt?: string | null;
  updatedAt?: string | null;
  updatedBy?: string | null;
  deletedAt?: string | null;
  deletedBy?: string | null;
}

/**
 * ユーティリティ
 */

/** 文字列が定義済み MemberRole か判定 */
export function isMemberRole(value: string): value is MemberRole {
  return (MEMBER_ROLES as readonly string[]).includes(value);
}

/** メンバーが指定の Permission.Name を保持しているか簡易チェック */
export function hasPermission(member: Member, permissionName: string): boolean {
  const target = permissionName.trim().toLowerCase();
  if (!target) return false;
  return member.permissions.some(
    (p) => p.trim().toLowerCase() === target,
  );
}
