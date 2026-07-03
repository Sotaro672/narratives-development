// frontend/amol/src/pages/MarketDetailPage.tsx
import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import Layout from "../components/layout/Layout";
import {
  fetchMarketResaleById,
  type MarketResaleListing,
} from "../features/market/marketApi";

import "../styles/page-layout.css";
import "../styles/market-detail-page.css";

export default function MarketDetailPage() {
  const navigate = useNavigate();
  const { resaleId } = useParams<{ resaleId: string }>();

  const [item, setItem] = useState<MarketResaleListing | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string>("");

  useEffect(() => {
    let cancelled = false;

    async function load() {
      if (!resaleId) {
        setError("出品情報が見つかりません。");
        setLoading(false);
        return;
      }

      setLoading(true);
      setError("");

      try {
        const data = await fetchMarketResaleById(resaleId);

        if (!cancelled) {
          setItem(data);
        }
      } catch (err) {
        if (!cancelled) {
          setError(
            err instanceof Error
              ? err.message
              : "出品情報の取得に失敗しました。",
          );
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void load();

    return () => {
      cancelled = true;
    };
  }, [resaleId]);

  const title = item?.productName || item?.tokenName || "マーケット詳細";

  const priceLabel =
    typeof item?.price === "number"
      ? `${item.price.toLocaleString("ja-JP")}円`
      : "価格未設定";

  return (
<Layout
  title={title}
  titleClickable={false}
  showBackButton
  onBackButtonClick={() => navigate(-1)}
  hideAnnouncementButton
  hideSettingsButton
  hideHamburgerMenu
>
      <div className="page-layout market-detail-page">
        {loading ? (
          <div className="market-detail-page__state">
            <p>読み込み中です...</p>
          </div>
        ) : null}

        {!loading && error ? (
          <div className="market-detail-page__state market-detail-page__state--error">
            <p>{error}</p>
          </div>
        ) : null}

        {!loading && !error && item ? (
          <section className="market-detail-page__card">
            <div className="market-detail-page__image-wrap">
              {item.imageUrl ? (
                <img
                  src={item.imageUrl}
                  alt={item.productName || item.tokenName || "出品画像"}
                  className="market-detail-page__image"
                />
              ) : (
                <div className="market-detail-page__image-placeholder">
                  No Image
                </div>
              )}
            </div>

            <div className="market-detail-page__content">
              <p className="market-detail-page__brand">
                {item.brandName || "ブランド名未設定"}
              </p>

              <h1 className="market-detail-page__title">
                {item.productName || item.tokenName || "商品名未設定"}
              </h1>

              <p className="market-detail-page__price">{priceLabel}</p>

              {item.condition ? (
                <dl className="market-detail-page__meta">
                  <div className="market-detail-page__meta-row">
                    <dt>状態</dt>
                    <dd>{item.condition}</dd>
                  </div>
                </dl>
              ) : null}

              {item.description ? (
                <div className="market-detail-page__description">
                  <h2>商品説明</h2>
                  <p>{item.description}</p>
                </div>
              ) : null}
            </div>
          </section>
        ) : null}
      </div>
    </Layout>
  );
}