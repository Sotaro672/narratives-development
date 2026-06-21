// frontend/amol/src/pages/AppareUsecase.tsx

import { useEffect } from "react";
import { useNavigate } from "react-router-dom";

import "../styles/page-layout.css";
import "../styles/examples-page.css";

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
      title="アパレル想定導入事例"
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
              <h3 className="examples-page__title">届けるお客様像</h3>

              <p className="examples-page__description">
                模造品が流通しているという報告が入ったが迅速な対策を講じれない。
                自社製品の二次流通先と繋がりたい。
              </p>
            </div>

            <div className="examples-page__grid">
              <article className="examples-page-card examples-page-card--static">
                <h3 className="examples-page-card__title">QRコード印刷例</h3>

                <div className="examples-page-card__image-wrap">
                  <img
                    src="/scan.png"
                    alt="首元のブランドタグにQRコードが付いた赤いTシャツのイラスト"
                    className="examples-page-card__image"
                    loading="lazy"
                  />
                </div>

                <p className="examples-page-card__description">
                  トップスで想定している印刷例です。QRコードをスキャンできる全ての形態に対応できます。
                </p>
              </article>

              <article className="examples-page-card examples-page-card--static">
                <h3 className="examples-page-card__title">
                  コピーされた場合の真贋対策
                </h3>

                <div className="examples-page-card__image-wrap">
                  <img
                    src="/antiCopy.png"
                    alt="首元のブランドタグにQRコードが付いた赤いTシャツのイラスト"
                    className="examples-page-card__image"
                    loading="lazy"
                  />
                </div>

                <p className="examples-page-card__description">
                  QRコードをコピーされたとしても商品の移譲履歴は１本しか伸びません。
                  ２本目以降はユーザー間でトークンを渡すことができないため、偽造業者は利益を稼ぐことができなくなります。
                </p>
              </article>

              <article className="examples-page-card examples-page-card--static">
                <h3 className="examples-page-card__title">現所有者との繋がり</h3>

                <p className="examples-page-card__description">
                  デザイナー、製造者は商品の現所有者へメッセージのやり取りをすることができます。
                  レビュー投稿、告知一斉送信、問い合わせをQRコードを介してやり取りすることができます。
                </p>
              </article>
            </div>
          </section>
        </div>
      </section>
    </Layout>
  );
}