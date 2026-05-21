// frontend/amol/src/pages/LandingPage.tsx
import { useRef } from "react";
import { useNavigate } from "react-router-dom";

import "../styles/page-layout.css";
import "../styles/landing-page.css";

import Layout from "../components/layout/Layout";
import Button from "../components/ui/Button";

export default function LandingPage() {
  const navigate = useNavigate();
  const salesSupportSectionRef = useRef<HTMLElement | null>(null);

  const scrollToSalesSupport = () => {
    salesSupportSectionRef.current?.scrollIntoView({
      behavior: "smooth",
      block: "start",
    });
  };

  return (
    <Layout title="AMOL" mode="landing">
      <section className="landing-page-hero">
        <div className="landing-page-hero__inner">
          <div className="landing-page-hero__content">
            <p className="landing-page-hero__eyebrow">
              ブロックチェーンで本物だけを届ける
            </p>

            <h1 className="landing-page-hero__title">AMOL</h1>

            <div className="page-actions">
              <Button
                variant="primary"
                onClick={() => navigate("/signup/select")}
              >
                新規登録
              </Button>
            </div>
          </div>

          <div className="landing-page-hero__image-wrap" aria-hidden="true">
            <img src="/hero.png" alt="" className="landing-page-hero__image" />
          </div>
        </div>
      </section>

      <section className="landing-page-section">
        <div className="landing-page-section__inner">
          <div className="landing-page-feature-grid">
            <article className="landing-page-feature-card">
              <h2 className="landing-page-feature-card__title">真贋証明</h2>
              <p className="landing-page-feature-card__text">
                商品のQRコードをスキャンするだけで、製品情報、コメント、所有履歴にアクセスでき、本物であると瞬時に分かります。
              </p>
              <div className="landing-page-feature-card__image-placeholder">
                <img
                  src="/scan.png"
                  alt="商品QRコードをスキャンした結果画面"
                  className="landing-page-feature-card__image"
                  loading="lazy"
                />
              </div>
            </article>

            <article
              className="landing-page-feature-card landing-page-feature-card--clickable"
              role="button"
              tabIndex={0}
              aria-label="営業支援の詳細へ移動"
              onClick={scrollToSalesSupport}
              onKeyDown={(event) => {
                if (event.key === "Enter" || event.key === " ") {
                  event.preventDefault();
                  scrollToSalesSupport();
                }
              }}
            >
              <h2 className="landing-page-feature-card__title">営業支援</h2>
              <p className="landing-page-feature-card__text">
                商品を誰が所有しているかがリアルタイムで分かり、販売後も新商品の情報を本当に興味のある人に届けることができます。
              </p>
              <div className="landing-page-feature-card__image-placeholder">
                <img
                  src="/comment.png"
                  alt="商品所有者とのコメント画面"
                  className="landing-page-feature-card__image"
                  loading="lazy"
                />
              </div>
            </article>
          </div>
        </div>
      </section>

      <section
        ref={salesSupportSectionRef}
        id="sales-support"
        className="landing-page-section landing-page-sales-support"
      >
        <div className="landing-page-section__inner">
          <div className="landing-page-sales-support__header">
            <p className="landing-page-sales-support__eyebrow">営業支援</p>
            <h2 className="landing-page-section__title landing-page-sales-support__title">
              購入後も、お客様とつながる
            </h2>
            <p className="landing-page-card__text landing-page-sales-support__lead">
              AMOLの電子名札を通じて、購入者は商品の感想や質問を投稿できます。
              生産者はお客様の声を直接受け取り、酒造りやファンづくりに活かすことができます。
            </p>
          </div>

          <div className="landing-page-sales-support__grid">
            <article className="landing-page-sales-support-card">
              <p className="landing-page-sales-support-card__label">
                これまでの課題
              </p>
              <h3 className="landing-page-sales-support-card__title">
                お客様の声が届きにくい
              </h3>
              <p className="landing-page-sales-support-card__text">
                美味しかったお酒でも、銘柄や蔵元の名前を思い出せず、
                購入後の感想や質問が生産者まで届きにくいことがあります。
              </p>
            </article>

            <div className="landing-page-sales-support-phone">
              <img
                src="/comment.png"
                alt="電子名札を通じて購入者とやり取りする画面"
                className="landing-page-sales-support-phone__image"
                loading="lazy"
              />
            </div>

            <article className="landing-page-sales-support-card">
              <p className="landing-page-sales-support-card__label">
                AMOLでできること
              </p>
              <h3 className="landing-page-sales-support-card__title">
                販売後も関係が続く
              </h3>
              <p className="landing-page-sales-support-card__text">
                購入者は飲んだ感想や質問を投稿でき、生産者は直接返信できます。
                商品をきっかけに、継続的なファンとの接点を作れます。
              </p>
            </article>
          </div>

          <div className="landing-page-sales-support__benefits">
            <article className="landing-page-sales-support-benefit landing-page-sales-support-benefit--with-image">
              <div className="landing-page-sales-support-benefit__content">
                <h3 className="landing-page-sales-support-benefit__title">
                  従来のSNSを用いてのお客様との繋がり
                </h3>
                <p className="landing-page-sales-support-benefit__text">
                  お客様からフォローをしていただく必要があり、再度購入したいと思っても商品名を思い出してもらえない場合があります。
                </p>
              </div>

              <div className="landing-page-sales-support-benefit__image-wrap">
                <img
                  src="/BeforeConnection.png"
                  alt="従来のSNSを用いたお客様との繋がり"
                  className="landing-page-sales-support-benefit__image"
                  loading="lazy"
                />
              </div>
            </article>

            <article className="landing-page-sales-support-benefit landing-page-sales-support-benefit--with-image">
              <div className="landing-page-sales-support-benefit__content">
                <h3 className="landing-page-sales-support-benefit__title">
                  電子名札を介したお客様との繋がり
                </h3>
                <p className="landing-page-sales-support-benefit__text">
                  商品を購入していただき、オプトイン操作をしていただいた全てのお客様にお知らせを送ることができます。
                </p>
              </div>

              <div className="landing-page-sales-support-benefit__image-wrap">
                <img
                  src="/AfterConnection.png"
                  alt="電子名札を介したお客様との繋がり"
                  className="landing-page-sales-support-benefit__image"
                  loading="lazy"
                />
              </div>
            </article>
          </div>
        </div>
      </section>
    </Layout>
  );
}