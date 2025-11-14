// frontend/permission/src/domain/entity/permission.ts

/**
 * PermissionCategory
 * backend/internal/domain/permission/entity.go の PermissionCategory に対応。
 * 各カテゴリはシステム機能の分類を示す。
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
 * backend/internal/domain/permission/entity.go の Permission に対応する型。
 *
 * - name は「brand.create」などの形式を取る
 * - category は PermissionCategory のいずれか
 */
export interface Permission {
  id: string;
  name: string;
  description: string;
  category: PermissionCategory;
}

/**
 * 定義済みカテゴリ一覧（UI用など）
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
 * name 構文の簡易バリデーション（Go側の正規表現に準拠: ^[a-z][a-z0-9.-]*\.[a-z][a-z0-9.-]*$）
 */
export function isValidPermissionName(name: string): boolean {
  return /^[a-z][a-z0-9.-]*\.[a-z][a-z0-9.-]*$/.test(name);
}

/**
 * Permission オブジェクトの簡易検証
 * backend の validate() と整合する。
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
 * Permission を生成するファクトリ関数（軽量版）
 */
export function createPermission(
  id: string,
  name: string,
  description: string,
  category: PermissionCategory
): Permission {
  return { id, name, description, category };
}
