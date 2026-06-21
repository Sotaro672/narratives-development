// frontend/amol/src/pages/CosmeticUsecase.tsx
import { useEffect } from "react";
import { useNavigate } from "react-router-dom";

import "../styles/page-layout.css";
import "../styles/examples-page.css";

import Layout from "../components/layout/Layout";

export default function CosmeticUsecase() {
  const navigate = useNavigate();

  useEffect(() => {
    window.scrollTo({
      top: 0,
      left: 0,
      behavior: "auto",
    });
  }, []);

  return (
    <Layout
      title="化粧品での導入事例"
      mode="landing"
      showBackButton
      titleClickable={false}
      hideSettingsButton
      hideAnnouncementButton
      onBackButtonClick={() => {
        navigate("/use-cases");
      }}
    >
      <section className="landing-page-section">
        <div className="landing-page-section__inner">
          <section className="examples-page">
            <div className="examples-page__header">
              <h2 className="examples-page__title">化粧品での導入事例</h2>

              <p className="examples-page__description">
                化粧品領域での導入事例は現在実装中です。
                今後、成分情報、製造背景、使用方法、ブランドコンテンツなどを
                商品に紐づけて届ける活用を想定しています。
              </p>
            </div>

            <div className="examples-page__grid">
              <article className="examples-page-card">
                <h3 className="examples-page-card__title">登録予定の情報</h3>

                <div className="examples-page-card__image-wrap">
                  <img
                    src="/cosmetics.png"
                    alt="QRコード付きラベルが付いた化粧品ボトルのイラスト"
                    className="examples-page-card__image"
                    loading="lazy"
                  />
                </div>

                <p className="examples-page-card__description">
                  商品名、ブランド、成分、使用方法、容量、製造情報などを登録し、
                  お客様が購入後も確認できる商品情報として提供する想定です。
                </p>
              </article>

              <article className="examples-page-card">
                <h3 className="examples-page-card__title">
                  ブランドコンテンツ
                </h3>

                <p className="examples-page-card__description">
                  商品に込めた思想、開発ストーリー、使用シーン、How to コンテンツなどを掲載し、
                  パッケージだけでは伝えきれないブランド体験を補完できます。
                </p>
              </article>

              <article className="examples-page-card">
                <h3 className="examples-page-card__title">今後の展開</h3>

                <p className="examples-page-card__description">
                  化粧品カテゴリは現在実装中です。
                  今後、商品情報管理、購入者向けコンテンツ配信、継続利用を促す体験設計などに対応予定です。
                </p>
              </article>
            </div>
          </section>
        </div>
      </section>
    </Layout>
  );
}