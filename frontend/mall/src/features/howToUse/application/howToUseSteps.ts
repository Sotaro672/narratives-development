// frontend/mall/src/features/howToUse/application/howToUseSteps.ts

export type HowToUseCategory = "console" | "mall";

export type HowToUseSectionItem = {
  id: string;
  title: string;
};

export type HowToUseItem = {
  category: HowToUseCategory;
  slug: string;
  title: string;
  description: string;
  sections: HowToUseSectionItem[];
};

export const consoleItems: HowToUseItem[] = [
  {
    category: "console",
    slug: "brand-registration",
    title: "ブランド登録",
    description: "ブランドの名前、ロゴ、基本情報を入力し、ブランド専用ブロックチェーンウォレットを開設します。",
    sections: [
      { id: "brand-registration-open", title: "ブランド登録画面の開け方" },
      { id: "brand-registration-create", title: "ブランド登録手順" },
    ],
  },
  {
    category: "console",
    slug: "member-invite",
    title: "メンバー招待",
    description: "ブランド運営に関わるメンバーを招待し、管理画面へのアクセス権を付与します。",
    sections: [
      { id: "member-invite-open", title: "メンバー登録画面の開け方" },
      { id: "member-invite-send", title: "メンバー招待手順" },
      { id: "member-invite-accept", title: "招待の受諾" },
      { id: "member-invite-cancel", title: "招待の取り消し" },
    ],
  },
  {
    category: "console",
    slug: "product-design",
    title: "商品設計",
    description: "商品の名前、型番、採寸、色などの基本情報を登録します。",
    sections: [
      { id: "product-design-open", title: "商品設計画面の開け方" },
      { id: "product-design-category", title: "商品カテゴリの選択" },
      { id: "product-design-category-fields", title: "カテゴリ固有情報の入力（衣類）" },
      { id: "product-design-colors", title: "カラーの登録（衣類）" },
      { id: "product-design-sizes", title: "サイズ・採寸の登録（衣類）" },
      { id: "product-design-model-numbers", title: "型番の登録" },
      { id: "product-design-shipping-package", title: "配送時の梱包情報の登録" },
      { id: "product-design-save", title: "商品設計の保存" },
    ],
  },
  {
    category: "console",
    slug: "token-design",
    title: "トークン設計",
    description: "ブロックチェーントークンのアイコン画像とコンテンツを登録します。",
    sections: [
      { id: "token-design-open", title: "トークン設計画面の開け方" },
      { id: "token-design-create", title: "トークン設計手順" },
    ],
  },
  {
    category: "console",
    slug: "production",
    title: "生産",
    description: "型番毎の生産数を記入し、商品毎に固有のQRコードを印刷します。",
    sections: [
      { id: "production-open", title: "生産計画作成画面の開け方" },
      { id: "production-quantity", title: "生産数の入力" },
      { id: "production-print", title: "商品の印刷" },
      { id: "production-print-result", title: "印刷結果の確認" },
    ],
  },
  {
    category: "console",
    slug: "inspection",
    title: "検品",
    description: "検品スキャナーでQRコードをスキャンして検品結果を入力します。",
    sections: [
      { id: "inspection-open", title: "検品スキャナーの開き方" },
      { id: "inspection-process", title: "検品手順" },
      { id: "inspection-complete", title: "検品の完了" },
    ],
  },
  {
    category: "console",
    slug: "mint",
    title: "ミント",
    description: "検品合格した商品にブロックチェーントークンを連携します。",
    sections: [
      { id: "mint-request", title: "ミント申請" },
    ],
  },
  {
    category: "console",
    slug: "inventory",
    title: "在庫",
    description: "在庫保管場所の住所を設定します。",
    sections: [
      { id: "inventory-location", title: "在庫の保管場所を設定します。" },
    ],
  },
  {
    category: "console",
    slug: "shipping",
    title: "配送",
    description: "配送料金体系を設定します。",
    sections: [
      { id: "shipping-fee", title: "配送料金体系の設定" },
    ],
  },
  {
    category: "console",
    slug: "listing",
    title: "出品",
    description: "すべての準備が完了した商品を購入者Mallに出品し、販売を開始します。",
    sections: [
      { id: "listing-settings", title: "在庫の保管場所と配送料金を設定" },
      { id: "listing-create", title: "出品作成" },
      { id: "listing-stop", title: "出品停止" },
    ],
  },
  {
    category: "console",
    slug: "orders",
    title: "注文",
    description: "Mallからの注文内容を確認します。",
    sections: [
      { id: "orders-dispatch", title: "商品の発送" },
    ],
  },
  {
    category: "console",
    slug: "inquiry",
    title: "お問い合わせ",
    description: "購入者から送信された商品や取引内容についてのお問い合わせを確認します。",
    sections: [
      { id: "inquiry-return-response", title: "返品申請への対応" },
      { id: "inquiry-return-received", title: "返品受領後の手順" },
    ],
  },
  {
    category: "console",
    slug: "comment-reply",
    title: "コメント返信",
    description: "トークンコンテンツに投稿されたコメントを確認し、ブランドから返信します。",
    sections: [
      { id: "comment-reply", title: "コメント返信" },
    ],
  },
  {
    category: "console",
    slug: "reviews",
    title: "レビュー",
    description: "購入者から投稿されたレビューを確認します。",
    sections: [
      { id: "reviews-view", title: "レビュー確認" },
    ],
  },
  {
    category: "console",
    slug: "announcement",
    title: "告知",
    description: "発行したトークンを所有するアバターへお知らせを一斉送信します。",
    sections: [
      { id: "announcement-send", title: "告知の一斉送信" },
      { id: "announcement-preview", title: "アバターへの見え方" },
    ],
  },
];

export const mallItems: HowToUseItem[] = [
  {
    category: "mall",
    slug: "avatar-registration",
    title: "登録",
    description: "購入者Mallでアカウントを作成し、アバター情報を登録・編集します。",
    sections: [
      { id: "avatar-registration-account", title: "Mallアカウントの作成" },
      { id: "avatar-registration-avatar", title: "アバターの作成" },
      { id: "avatar-registration-edit", title: "アバターを編集する" },
    ],
  },
  {
    category: "mall",
    slug: "shipping-address",
    title: "配送先住所",
    description: "購入した商品の配送先として使用する住所を登録・管理します。",
    sections: [
      { id: "shipping-address-open", title: "配送先住所の登録画面を開く" },
    ],
  },
  {
    category: "mall",
    slug: "purchase",
    title: "購入",
    description: "商品を購入し、届いた商品のQRコードをスキャンしてブロックチェーントークンを受け取ります。",
    sections: [
      { id: "purchase-order", title: "商品の購入" },
      { id: "purchase-token-transfer", title: "トークンの受け取り" },
      { id: "purchase-resale-receipt", title: "フリマ取引での受取" },
    ],
  },
  {
    category: "mall",
    slug: "comment",
    title: "コメント",
    description: "所有しているトークンのコンテンツにコメントを投稿します。",
    sections: [
      { id: "comment-post", title: "コメント投稿" },
    ],
  },
  {
    category: "mall",
    slug: "cancel",
    title: "キャンセル",
    description: "購入した商品の注文をキャンセルする手順を確認します。",
    sections: [
      { id: "cancel-order", title: "注文のキャンセル" },
      { id: "cancel-resale-trade", title: "フリマ取引のキャンセル" },
    ],
  },
  {
    category: "mall",
    slug: "return",
    title: "返品",
    description: "購入した商品の返品手続きと返送方法を確認します。",
    sections: [
      { id: "return-request", title: "返品申請" },
      { id: "return-reply", title: "返信入力" },
      { id: "return-close", title: "お問い合わせの終了" },
    ],
  },
  {
    category: "mall",
    slug: "review",
    title: "レビュー投稿",
    description: "購入後に商品の体験や評価をレビューとして投稿します。",
    sections: [
      { id: "review-post", title: "レビュー投稿" },
    ],
  },
  {
    category: "mall",
    slug: "payout-account",
    title: "売上受取口座",
    description: "フリマの売上を受け取るための銀行口座を登録・管理します。",
    sections: [
      { id: "payout-account", title: "売上受取口座" },
    ],
  },
  {
    category: "mall",
    slug: "resale",
    title: "フリマ",
    description: "所有しているトークンをフリマへ出品できます。",
    sections: [
      { id: "resale-open", title: "フリマ出品画面を開く" },
      { id: "resale-create", title: "フリマへ出品する" },
      { id: "resale-edit", title: "出品内容を編集する" },
    ],
  },
  {
    category: "mall",
    slug: "market",
    title: "マーケット",
    description: "フリマに出品されている商品を閲覧し、商品詳細を確認して購入できます。",
    sections: [
      { id: "market-browse", title: "マーケットを見る" },
      { id: "market-chat", title: "出品者とチャットする" },
      { id: "market-reply", title: "出品者から返信" },
    ],
  },
  {
    category: "mall",
    slug: "trade",
    title: "取引",
    description: "フリマで成立した取引の配送、受取確認、評価などの手順を確認します。",
    sections: [
      { id: "trade-dispatch", title: "商品を発送する" },
    ],
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