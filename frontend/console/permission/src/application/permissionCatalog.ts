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
 * → front 側のカテゴリ判定 & 日本語表示名の基準として使用する。
 */
type PermissionCatalogItem = {
  name: string;
  category: PermissionCategory;
  description: string; // 日本語表示名（backend catalog.go の Description と同じ）
};

const permissionCatalog: PermissionCatalogItem[] = [
  // Wallet
  { name: "wallet.view", category: "wallet", description: "ウォレット閲覧" },
  {
    name: "wallet.settings.view",
    category: "wallet",
    description: "ウォレット設定閲覧",
  },

  // Inquiry
  {
    name: "inquiry.view",
    category: "inquiry",
    description: "問い合わせ一覧閲覧",
  },
  {
    name: "inquiry.detail.view",
    category: "inquiry",
    description: "問い合わせ詳細・履歴閲覧",
  },

  // Organization
  {
    name: "organization.settings.view",
    category: "organization",
    description: "組織設定・構成情報閲覧",
  },

  // Brand
  {
    name: "brand.view",
    category: "brand",
    description: "ブランド一覧閲覧",
  },
  {
    name: "brand.detail.view",
    category: "brand",
    description: "ブランド詳細閲覧",
  },
  {
    name: "brand.archive.view",
    category: "brand",
    description: "アーカイブ済みブランド閲覧",
  },

  // Token
  {
    name: "token.view",
    category: "token",
    description: "トークン一覧閲覧",
  },
  {
    name: "token.distribution.view",
    category: "token",
    description: "トークン配布・割当状況閲覧",
  },

  // Order
  {
    name: "order.view",
    category: "order",
    description: "注文情報閲覧",
  },

  // Member
  {
    name: "member.view",
    category: "member",
    description: "メンバー一覧閲覧",
  },
  {
    name: "member.roles.view",
    category: "member",
    description: "メンバー権限・ロール設定閲覧",
  },

  // Inventory
  {
    name: "inventory.view",
    category: "inventory",
    description: "在庫情報閲覧",
  },

  // Production
  {
    name: "production.status.view",
    category: "production",
    description: "生産工程ステータス閲覧",
  },

  // System
  {
    name: "system.admin.view",
    category: "system",
    description: "システム設定・管理情報閲覧",
  },
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

/**
 * 単一の permission name から日本語説明を取得
 */
export function getPermissionDescriptionJa(name: string): string {
  const key = name.trim();
  if (!key) return "";

  const hit = permissionCatalog.find((p) => p.name === key);
  if (!hit) {
    // 見つからない場合は元の name をそのまま返す
    return key;
  }
  return hit.description;
}

/**
 * 複数 permission name を日本語説明リストに変換
 */
export function mapPermissionNamesToDescriptionsJa(
  permissions: string[],
): string[] {
  return permissions
    .map((p) => p.trim())
    .filter((p) => p.length > 0)
    .map((p) => getPermissionDescriptionJa(p));
}
