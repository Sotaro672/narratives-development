// frontend/mall/src/features/howToUse/application/howToUseSteps.ts

export type HowToUseCategory = "console" | "mall";

export type HowToUseItem = {
  category: HowToUseCategory;
  slug: string;
  title: string;
  description: string;
};

export const consoleItems: HowToUseItem[] = [
  {
    category: "console",
    slug: "brand-registration",
    title: "ブランド登録",
    description: "ブランドの名前、ロゴ、基本情報を入力し、ブランド専用ブロックチェーンウォレットを開設します。",
  },
  {
    category: "console",
    slug: "member-invite",
    title: "メンバー招待",
    description: "ブランド運営に関わるメンバーを招待し、管理画面へのアクセス権を付与します。",
  },
  {
    category: "console",
    slug: "product-design",
    title: "商品設計",
    description: "商品の名前、型番、採寸、色などの基本情報を登録します。",
  },
  {
    category: "console",
    slug: "token-design",
    title: "トークン設計",
    description: "ブロックチェーントークンのアイコン画像とコンテンツを登録します。",
  },
  {
    category: "console",
    slug: "production",
    title: "生産",
    description: "型番毎の生産数を記入し、商品毎に固有のQRコードを印刷します。",
  },
  {
    category: "console",
    slug: "inspection",
    title: "検品",
    description: "検品スキャナーでQRコードをスキャンして検品結果を入力します。",
  },
  {
    category: "console",
    slug: "mint",
    title: "ミント",
    description: "検品合格した商品にブロックチェーントークンを連携します。",
  },
  {
    category: "console",
    slug: "inventory",
    title: "在庫",
    description: "在庫保管場所の住所を設定します。",
  },
  {
    category: "console",
    slug: "shipping",
    title: "配送",
    description: "配送料金体系を設定します。",
  },
  {
    category: "console",
    slug: "listing",
    title: "出品",
    description: "すべての準備が完了した商品を購入者Mallに出品し、販売を開始します。",
  },
  {
    category: "console",
    slug: "orders",
    title: "注文",
    description: "Mallからの注文内容を確認します。",
  },
  {
    category: "console",
    slug: "inquiry",
    title: "お問い合わせ",
    description: "購入者から送信された商品や取引内容についてのお問い合わせを確認します。",
  },
  {
    category: "console",
    slug: "comment-reply",
    title: "コメント返信",
    description: "トークンコンテンツに投稿されたコメントを確認し、ブランドから返信します。",
  },
  {
    category: "console",
    slug: "reviews",
    title: "レビュー",
    description: "購入者から投稿されたレビューを確認します。",
  },
  {
    category: "console",
    slug: "announcement",
    title: "告知",
    description: "発行したトークンを所有するアバターへお知らせを一斉送信します。",
  },
];

export const mallItems: HowToUseItem[] = [
  {
    category: "mall",
    slug: "avatar-registration",
    title: "登録",
    description: "購入者Mallでアカウントを作成し、アバター情報を登録します。",
  },
  {
    category: "mall",
    slug: "shipping-address",
    title: "配送先住所",
    description: "購入した商品の配送先として使用する住所を登録・管理します。",
  },
  {
    category: "mall",
    slug: "purchase",
    title: "購入",
    description: "商品を購入し、届いた商品のQRコードをスキャンしてブロックチェーントークンを受け取ります。",
  },
  {
    category: "mall",
    slug: "comment",
    title: "コメント",
    description: "所有しているトークンのコンテンツにコメントを投稿します。",
  },
  {
    category: "mall",
    slug: "cancel",
    title: "キャンセル",
    description: "購入した商品の注文をキャンセルする手順を確認します。",
  },
  {
    category: "mall",
    slug: "return",
    title: "返品",
    description: "購入した商品の返品手続きと返送方法を確認します。",
  },
  {
    category: "mall",
    slug: "review",
    title: "レビュー投稿",
    description: "購入後に商品の体験や評価をレビューとして投稿します。",
  },
  {
    category: "mall",
    slug: "payout-account",
    title: "売上受取口座",
    description: "フリマの売上を受け取るための銀行口座を登録・管理します。",
  },
  {
    category: "mall",
    slug: "resale",
    title: "フリマ",
    description: "所有しているトークンをフリマへ出品できます。",
  },
];

export const howToUseItems: HowToUseItem[] = [
  ...consoleItems,
  ...mallItems,
];

export function isHowToUseCategory(
  value: string | undefined,
): value is HowToUseCategory {
  return value === "console" || value === "mall";
}

export function findHowToUseItem(
  category: HowToUseCategory,
  slug: string,
): HowToUseItem | undefined {
  return howToUseItems.find(
    (item) =>
      item.category === category &&
      item.slug === slug,
  );
}