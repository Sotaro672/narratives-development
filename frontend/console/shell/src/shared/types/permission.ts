// frontend/shell/src/shared/types/permission.ts

/**
 * PermissionCategory
 * backend/internal/domain/permission/entity.go に対応。
 * 各カテゴリは権限の対象ドメイン（機能領域）を表す。
 */
export type PermissionCategory =
  | "wallet"
  | "inquiry"
  | "organization"
  | "brand"
  | "member"
  | "order"
  | "product"
  | "campaign"
  | "token"
  | "inventory"
  | "production"
  | "analytics"
  | "system";

/**
 * Permission
 * backend/internal/domain/permission/entity.go の Permission に対応する共通型。
 * - name は「brand.create」や「member.view」などの形式。
 * - category は PermissionCategory のいずれか。
 */
export interface Permission {
  id: string;
  name: string;
  description: string;
  category: PermissionCategory;
}

/**
 * 定義済みカテゴリ一覧（選択UIやバリデーションに使用）
 */
export const PERMISSION_CATEGORIES: PermissionCategory[] = [
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
];

/**
 * 権限名の形式バリデーション
 * backend 側の正規表現 `^[a-z][a-z0-9.-]*\.[a-z][a-z0-9.-]*$` に対応。
 * 例: "brand.create", "member.update", "system.view-all"
 */
export function isValidPermissionName(name: string): boolean {
  return /^[a-z][a-z0-9.-]*\.[a-z][a-z0-9.-]*$/.test(name);
}

/**
 * Permission オブジェクトの簡易バリデーション
 * backend の validate() と整合性を持つ。
 */
export function validatePermission(p: Permission): string[] {
  const errors: string[] = [];

  if (!p.id?.trim()) errors.push("IDは必須です");
  if (!p.name?.trim() || !isValidPermissionName(p.name))
    errors.push("権限名の形式が不正です（例: brand.create）");
  if (!p.description?.trim()) errors.push("説明は必須です");
  if (!PERMISSION_CATEGORIES.includes(p.category))
    errors.push("カテゴリが不正です");

  return errors;
}

/**
 * Permission インスタンス生成のユーティリティ
 * バリデーション済みオブジェクトを構築する。
 */
export function createPermission(
  id: string,
  name: string,
  description: string,
  category: PermissionCategory
): Permission {
  return { id, name, description, category };
}
