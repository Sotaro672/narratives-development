// frontend/permission/mockdata.tsx

export type Permission = {
  name: string;
  category: string;
  description: string;
};

export const ALL_PERMISSIONS: Permission[] = [
  { name: "wallet.view", category: "wallet", description: "ウォレット情報の閲覧" },
  { name: "wallet.edit", category: "wallet", description: "ウォレット設定の編集" },
  { name: "inquiry.view", category: "inquiry", description: "問い合わせの閲覧" },
  { name: "inquiry.manage", category: "inquiry", description: "問い合わせ対応・管理" },
  { name: "organization.admin", category: "organization", description: "組織の完全な管理権限" },
  { name: "brand.create", category: "brand", description: "ブランドの作成" },
  { name: "brand.edit", category: "brand", description: "ブランド情報の編集" },
  { name: "brand.delete", category: "brand", description: "ブランドの削除" },
  { name: "token.create", category: "token", description: "トークンの作成" },
  { name: "token.manage", category: "token", description: "トークンの管理・配布" },
  { name: "listing.view", category: "listing", description: "出品情報の閲覧" },
  { name: "order.manage", category: "order", description: "注文の管理" },
];
