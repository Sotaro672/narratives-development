/**
 * Remote route manifest
 * 各マイクロフロントエンド (MFE) のルーティングエントリを管理。
 * shell 側の router.tsx で動的 import に使用される。
 */
export const remoteRouteModules = {
  // ─────────── 組織関連
  member: "member/routes",
  brand: "brand/routes",
  permission: "permission/routes",
  company: "company/routes",

  // ─────────── 問い合わせ / 商品 / 在庫 / 生産
  inquiries: "inquiries/routes",
  listings: "listings/routes",
  inventory: "inventory/routes",
  preview: "preview/routes",
  production: "production/routes",

  // ─────────── 設計・ブループリント系
  tokenBlueprint: "tokenBlueprint/routes",
  productBlueprint: "productBlueprint/routes",
  sales: "sales/routes",

  // ✅ 追加: レビュー（ProductBlueprintReview / TokenBlueprintReview）
  productBlueprintReview: "productBlueprintReview/routes",
  tokenBlueprintReview: "tokenBlueprintReview/routes",

  // ─────────── 取引・注文・財務
  mint: "mint/routes",
  orders: "orders/routes",
  accounts: "accounts/routes",
  transactions: "transactions/routes",
} as const;