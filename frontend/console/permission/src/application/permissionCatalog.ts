// frontend/console/permission/src/application/permissionCatalog.ts

/**
 * backend の PermissionCategory と同じ分類
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
 * backend の static allPermissions と同じ一覧（最終的な真実）
 * → front 側のカテゴリ判定の基準として使用する。
 */
const permissionCatalog: { name: string; category: PermissionCategory }[] = [
  // Wallet
  { name: "wallet.view", category: "wallet" },
  { name: "wallet.settings.view", category: "wallet" },

  // Inquiry
  { name: "inquiry.view", category: "inquiry" },
  { name: "inquiry.detail.view", category: "inquiry" },

  // Organization
  { name: "organization.settings.view", category: "organization" },

  // Brand
  { name: "brand.view", category: "brand" },
  { name: "brand.detail.view", category: "brand" },
  { name: "brand.archive.view", category: "brand" },

  // Token
  { name: "token.view", category: "token" },
  { name: "token.distribution.view", category: "token" },

  // Order
  { name: "order.view", category: "order" },

  // Member
  { name: "member.view", category: "member" },
  { name: "member.roles.view", category: "member" },

  // Inventory
  { name: "inventory.view", category: "inventory" },

  // Production
  { name: "production.status.view", category: "production" },

  // System
  { name: "system.admin.view", category: "system" },
];

/**
 * CategoryFromPermissionName
 *
 * - 権限名 → カテゴリを取得
 * - 1) static catalog に完全一致すればそのカテゴリ
 * - 2) なければ "wallet.edit" の prefix = "wallet" から推論
 * - Firestore に古いデータが入っている状況でも正常動作する
 */
export function CategoryFromPermissionName(
  name: string,
): PermissionCategory | null {
  const n = name.trim();
  if (!n) return null;

  // 1) カタログと完全一致
  const hit = permissionCatalog.find((p) => p.name === n);
  if (hit) return hit.category;

  // 2) 旧データ ("wallet.edit" など) は prefix 推論
  const firstDot = n.indexOf(".");
  if (firstDot <= 0) return null;

  const prefix = n.substring(0, firstDot) as PermissionCategory;

  const validCategories: PermissionCategory[] = [
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

  if (validCategories.includes(prefix)) {
    return prefix;
  }

  return null;
}

/**
 * 権限名一覧を Category → 権限配列 にグルーピングする
 *
 * 例:
 * ["wallet.view","brand.edit","brand.view"]
 * →
 * {
 *   wallet: ["wallet.view"],
 *   brand: ["brand.edit","brand.view"]
 * }
 */
export function groupPermissionsByCategory(
  permissionNames: string[],
): Record<PermissionCategory, string[]> {
  const grouped: Record<PermissionCategory, string[]> = {} as any;

  for (const perm of permissionNames) {
    const cat = CategoryFromPermissionName(perm);
    if (!cat) continue;
    if (!grouped[cat]) grouped[cat] = [];
    grouped[cat].push(perm);
  }

  return grouped;
}
