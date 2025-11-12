// frontend\permission\src\infrastructure\mockdata\mockdata.tsx

import type {
  Permission,
  PermissionCategory,
} from "../../../../shell/src/shared/types/permission";

/**
 * ダミー権限一覧
 * backend/internal/domain/permission/entity.go および
 * frontend/shell/src/shared/types/permission.ts に準拠。
 */
export const ALL_PERMISSIONS: Permission[] = [
  {
    id: "perm_wallet_view",
    name: "wallet.view",
    category: "wallet",
    description: "ウォレット情報の閲覧",
  },
  {
    id: "perm_wallet_edit",
    name: "wallet.edit",
    category: "wallet",
    description: "ウォレット設定の編集",
  },
  {
    id: "perm_inquiry_view",
    name: "inquiry.view",
    category: "inquiry",
    description: "問い合わせの閲覧",
  },
  {
    id: "perm_inquiry_manage",
    name: "inquiry.manage",
    category: "inquiry",
    description: "問い合わせ対応・管理",
  },
  {
    id: "perm_org_admin",
    name: "organization.admin",
    category: "organization",
    description: "組織の完全な管理権限",
  },
  {
    id: "perm_brand_create",
    name: "brand.create",
    category: "brand",
    description: "ブランドの作成",
  },
  {
    id: "perm_brand_edit",
    name: "brand.edit",
    category: "brand",
    description: "ブランド情報の編集",
  },
  {
    id: "perm_brand_delete",
    name: "brand.delete",
    category: "brand",
    description: "ブランドの削除",
  },
  {
    id: "perm_token_create",
    name: "token.create",
    category: "token",
    description: "トークンの作成",
  },
  {
    id: "perm_token_manage",
    name: "token.manage",
    category: "token",
    description: "トークンの管理・配布",
  },
  {
    id: "perm_order_manage",
    name: "order.manage",
    category: "order",
    description: "注文の管理",
  },
  {
    id: "perm_member_view",
    name: "member.view",
    category: "member",
    description: "メンバー情報の閲覧",
  },
  {
    id: "perm_member_edit",
    name: "member.edit",
    category: "member",
    description: "メンバー情報の編集",
  },
  {
    id: "perm_inventory_view",
    name: "inventory.view",
    category: "inventory",
    description: "在庫情報の閲覧",
  },
  {
    id: "perm_production_manage",
    name: "production.manage",
    category: "production",
    description: "生産工程の管理",
  },
  {
    id: "perm_system_admin",
    name: "system.admin",
    category: "system",
    description: "システム管理全般",
  },
];

/**
 * カテゴリ別に権限をグループ化して返すヘルパ
 */
export function groupPermissionsByCategory(perms: Permission[]): Record<PermissionCategory, Permission[]> {
  return perms.reduce((acc, p) => {
    if (!acc[p.category]) acc[p.category] = [];
    acc[p.category].push(p);
    return acc;
  }, {} as Record<PermissionCategory, Permission[]>);
}
