// frontend/permission/src/domain/entity/permission.ts

/**
 * PermissionCategory
 * backend/internal/domain/permission/entity.go に対応するカテゴリ型。
 * フロント側では定義表示のみ使用し、編集は行わない。
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
 * backend/internal/domain/permission/entity.go の Permission 型に対応。
 * フロント側ではバックエンドから受け取って表示する用途のみ。
 */
export interface Permission {
  id: string;
  name: string;
  description: string;
  category: PermissionCategory;
}

/**
 * UI 表示用のカテゴリ一覧。
 * 利用者が権限を編集しない前提なので、UI のフィルタや分類用途のみで使用する。
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
