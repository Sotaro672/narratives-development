// frontend/mall/src/pages/HowToUsePage.tsx

import { useNavigate } from "react-router-dom";

import Layout from "../components/layout/Layout";
import Button from "../components/ui/Button";

import "../styles/page-layout.css";
import "../styles/how-to-use-page.css";

export type HowToUseCategory = "seller" | "buyer";

export type HowToUseItem = {
  category: HowToUseCategory;
  slug: string;
  title: string;
  description: string;
};

export const sellerItems: HowToUseItem[] = [
  {
    category: "seller",
    slug: "brand-registration",
    title: "ブランド登録",
    description: "ブランドの名前、ロゴ、基本情報を入力し、ブランド専用ブロックチェーンウォレットを開設します。",
  },
  {
    category: "seller",
    slug: "member-invite",
    title: "メンバー招待",
    description: "ブランド運営に関わるメンバーを招待し、管理画面へのアクセス権を付与します。",
  },
  {
    category: "seller",
    slug: "product-design",
    title: "商品設計",
    description: "商品の名前、型番、採寸、色などの基本情報を登録します。",
  },
  {
    category: "seller",
    slug: "token-design",
    title: "トークン設計",
    description: "ブロックチェーントークンのアイコン画像とコンテンツを登録します。",
  },
  {
    category: "seller",
    slug: "production",
    title: "生産",
    description: "型番毎の生産数を記入し、商品毎に固有のQRコードを印刷します。",
  },
  {
    category: "seller",
    slug: "inspection",
    title: "検品",
    description: "検品スキャナーでQRコードをスキャンして検品結果を入力します。",
  },
  {
    category: "seller",
    slug: "mint",
    title: "ミント",
    description: "検品合格した商品にブロックチェーントークンを連携します。",
  },
  {
    category: "seller",
    slug: "listing",
    title: "出品",
    description: "すべての準備が完了した商品を購入者Mallに出品し、販売を開始します。",
  },
  {
    category: "seller",
    slug: "orders-reviews",
    title: "注文・レビュー確認",
    description: "Mallからの注文と購入者からのレビューを確認します。",
  },
  {
    category: "seller",
    slug: "announcement",
    title: "告知",
    description: "発行したトークンを所有するアバターへお知らせを一斉送信します。",
  },
];

export const buyerItems: HowToUseItem[] = [
  {
    category: "buyer",
    slug: "avatar-registration",
    title: "アバター登録",
    description: "購入者Mallでアカウントを作成し、アバター情報を登録します。",
  },
  {
    category: "buyer",
    slug: "purchase",
    title: "購入",
    description: "商品を購入し、届いた商品のQRコードをスキャンしてブロックチェーントークンを受け取ります。",
  },
  {
    category: "buyer",
    slug: "review",
    title: "レビュー投稿",
    description: "購入後に商品の体験や評価をレビューとして投稿します。",
  },
  {
    category: "buyer",
    slug: "inquiry",
    title: "お問い合わせ",
    description: "購入した商品や取引内容について出品者へお問い合わせを送信します。",
  },
  {
    category: "buyer",
    slug: "resale",
    title: "フリマ",
    description: "所有しているトークンをフリマへ出品できます。",
  },
];

type ItemListProps = {
  title: string;
  items: HowToUseItem[];
  onDetailClick: (item: HowToUseItem) => void;
};

function ItemList({ title, items, onDetailClick }: ItemListProps) {
  return (
    <section className="how-to-use-section">
      <h2 className="how-to-use-section__title">{title}</h2>

      <div className="how-to-use-section__items">
        {items.map((item) => (
          <details
            key={`${item.category}-${item.slug}`}
            className="how-to-use-item"
          >
            <summary className="how-to-use-item__summary">
              <span className="how-to-use-item__title">{item.title}</span>
              <span
                className="how-to-use-item__arrow"
                aria-hidden="true"
              />
            </summary>

            <div className="how-to-use-item__content">
              <p className="how-to-use-item__description">
                {item.description}
              </p>

              <button
                type="button"
                className="how-to-use-item__detail-link"
                onClick={() => onDetailClick(item)}
              >
                詳しく見る
              </button>
            </div>
          </details>
        ))}
      </div>
    </section>
  );
}

export default function HowToUsePage() {
  const navigate = useNavigate();

  const handleDetailClick = (item: HowToUseItem) => {
    navigate(`/how-to-use/${item.category}/${item.slug}`);
  };

  return (
    <Layout title="使い方" mode="landing">
      <main className="how-to-use-page">
        <div className="how-to-use-page__inner">
          <ItemList
            title="出品者Console"
            items={sellerItems}
            onDetailClick={handleDetailClick}
          />

          <ItemList
            title="購入者Mall"
            items={buyerItems}
            onDetailClick={handleDetailClick}
          />

          <div className="page-actions">
            <Button
              variant="primary"
              onClick={() => navigate("/signin/select")}
            >
              試作品を体験
            </Button>
          </div>
        </div>
      </main>
    </Layout>
  );
}