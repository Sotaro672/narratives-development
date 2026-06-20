// frontend/amol/src/pages/AppareUsecase.tsx
import { useEffect } from "react";
import { useNavigate } from "react-router-dom";

import "../styles/page-layout.css";
import "../styles/examples.css";

import Layout from "../components/layout/Layout";

export default function AppareUsecase() {
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
      title="アパレルでの導入事例"
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
              <h2 className="examples-page__title">アパレルでの導入事例</h2>

              <p className="examples-page__description">
                AMOLでは、衣服に紐づく情報をトークンとして管理し、
                商品の背景や品質情報、ブランドの世界観をお客様へ届けることができます。
              </p>
            </div>

            <div className="examples-page__grid">
              <article className="examples-page-card">
                <h3 className="examples-page-card__title">登録できる情報</h3>

                <div className="examples-page-card__image-wrap">
                  <img
                    src="/shirts.png"
                    alt="首元のブランドタグにQRコードが付いた赤いTシャツのイラスト"
                    className="examples-page-card__image"
                    loading="lazy"
                  />
                </div>

                <p className="examples-page-card__description">
                  色、採寸、素材、GSM、洗濯表示など、アパレル商品に必要な情報を登録できます。
                  商品ごとの仕様やケア情報を、購入後も確認できる形で届けられます。
                </p>
              </article>

              <article className="examples-page-card">
                <h3 className="examples-page-card__title">
                  ブランド体験の拡張
                </h3>

                <p className="examples-page-card__description">
                  商品に付与されたQRコードから、ブランドストーリー、着用イメージ、
                  コーディネート提案、制作背景などのコンテンツへ誘導できます。
                  タグや下げ札だけでは伝えきれない情報を、デジタル上で補完できます。
                </p>
              </article>

              <article className="examples-page-card">
                <h3 className="examples-page-card__title">購入後の活用</h3>

                <p className="examples-page-card__description">
                  お客様は商品に紐づくトークンを通じて、保有商品の情報を確認できます。
                  将来的には、限定コンテンツ、メンテナンス案内、コミュニティ施策などにも活用できます。
                </p>
              </article>
            </div>
          </section>
        </div>
      </section>
    </Layout>
  );
}