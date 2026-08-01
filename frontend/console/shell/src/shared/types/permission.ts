// frontend/console/shell/src/shared/types/permission.ts

/**
 * PermissionCategory
 * backend/internal/domain/permission/entity.go に対応するカテゴリ一覧。
 *
 * カテゴリを追加・変更する場合は、この配列だけを変更する。
 */
export const PERMISSION_CATEGORIES = [
  "wallet",
  "inquiry",
  "organization",
  "brand",
  "member",
  "order",
  "product",
  "campaign",
  "token",
  "inventory",
  "production",
  "analytics",
  "system",
] as const;

/**
 * PERMISSION_CATEGORIESから生成されるカテゴリ型。
 */
export type PermissionCategory =
  (typeof PERMISSION_CATEGORIES)[number];

/**
 * backend/internal/domain/permission/entity.go の
 * Permission に対応する共通型。
 */
export interface Permission {
  id: string;
  name: string;
  description: string;
  category: PermissionCategory;
}

/**
 * 任意の文字列がPermissionCategoryか判定する。
 */
export function isPermissionCategory(
  value: string,
): value is PermissionCategory {
  return (
    PERMISSION_CATEGORIES as readonly string[]
  ).includes(value);
}