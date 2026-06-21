// frontend/amol/src/pages/SakeUsecasePage.tsx
import { useEffect } from "react";
import { useNavigate } from "react-router-dom";

import "../styles/page-layout.css";
import "../styles/examples-page.css";

import Layout from "../components/layout/Layout";

export default function SakeUsecasePage() {
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
      title="酒類での導入事例"
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
              <h2 className="examples-page__title">酒類での導入事例</h2>

              <p className="examples-page__description">
                AMOLでは、日本酒や焼酎、ワインなどの酒類に対して、
                産地、原料、生産者、製造工程などの情報をトークンに紐づけて届けることができます。
              </p>
            </div>

            <div className="examples-page__grid">
              <article className="examples-page-card">
                <h3 className="examples-page-card__title">登録できる情報</h3>

                <div className="examples-page-card__image-wrap">
                  <img
                    src="/sake.png"
                    alt="ラベルにQRコードが印字された一升瓶のイラスト"
                    className="examples-page-card__image"
                    loading="lazy"
                  />
                </div>

                <p className="examples-page-card__description">
                  生産者、産地、素材、製造年数、内容量、アルコール度数など、
                  酒類の商品情報を登録できます。ラベルだけでは伝えきれない詳細情報を、
                  QRコードを通じてお客様に届けられます。
                </p>
              </article>

              <article className="examples-page-card">
                <h3 className="examples-page-card__title">製造過程の共有</h3>

                <p className="examples-page-card__description">
                  酒米の生育状況、仕込み、発酵、熟成、蔵出しまでの流れをコンテンツとして掲載できます。
                  お客様に商品が完成するまでの過程を楽しんでもらうことで、
                  購入前後の体験価値を高められます。
                </p>
              </article>

              <article className="examples-page-card">
                <h3 className="examples-page-card__title">ファン化への活用</h3>

                <p className="examples-page-card__description">
                  購入者に限定情報や次回入荷案内、蔵元からのメッセージなどを届けることで、
                  一度きりの購買ではなく、継続的な関係づくりに活用できます。
                </p>
              </article>
            </div>
          </section>
        </div>
      </section>
    </Layout>
  );
}