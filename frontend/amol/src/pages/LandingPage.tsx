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
              電子名札を通じて、生産者は販売した商品の現在の所有者がと繋がることができます。
            </p>
          </div>

          <div className="landing-page-sales-support__benefits">
            <article className="landing-page-sales-support-benefit landing-page-sales-support-benefit--with-image">
              <div className="landing-page-sales-support-benefit__content">
                <p className="landing-page-sales-support-card__label">
                  これまでの課題
                </p>
                <h3 className="landing-page-sales-support-benefit__title">
                  従来のSNSではお客様からフォローしてもらう必要があった。
                </h3>
                <p className="landing-page-sales-support-benefit__text">
                  従来のSNSでは折角商品を気に入ってもらっても、お客様からフォローをして頂かないと商品を忘れてしまう可能性があります。
                </p>



              <div className="landing-page-sales-support-benefit__image-wrap">
                <img
                  src="/BeforeConnection1.png"
                  alt="従来のSNSを用いたお客様との繋がり"
                  className="landing-page-sales-support-benefit__image"
                  loading="lazy"
                />
              </div>
              <h3 className="landing-page-sales-support-benefit__title">
                  二次流通を追跡できない
                </h3>
                <p className="landing-page-sales-support-benefit__text">
                  従来のSNSでは販売した商品が年々拡大する二次流通市場でどの様なお客様に購入されているのかを追跡することができません。
                </p>
                              <div className="landing-page-sales-support-benefit__image-wrap">
                <img
                  src="/BeforeConnection2.png"
                  alt="従来のSNSを用いたお客様との繋がり"
                  className="landing-page-sales-support-benefit__image"
                  loading="lazy"
                />
              </div>
              </div>
            </article>

            <article className="landing-page-sales-support-benefit landing-page-sales-support-benefit--with-image">
              <div className="landing-page-sales-support-benefit__content">
                <p className="landing-page-sales-support-card__label">
                  AMOLでできること
                </p>
                <h3 className="landing-page-sales-support-benefit__title">
                  電子名札を介したお客様との繋がり
                </h3>
                <p className="landing-page-sales-support-benefit__text">
                  商品を購入していただき、オプトイン操作をしていただいた全てのお客様にお知らせを送ることができます。
                </p>
              <div className="landing-page-sales-support-benefit__image-wrap">
                <img
                  src="/AfterConnection1.png"
                  alt="電子名札を介したお客様との繋がり"
                  className="landing-page-sales-support-benefit__image"
                  loading="lazy"
                />
              </div>
                
                <h3 className="landing-page-sales-support-benefit__title">
                  二次流通市場でも現在の所有者と繋がれる
                </h3>
                <p className="landing-page-sales-support-benefit__text">
                  お客様間で電子名札を譲渡することも可能です。二次流通市場で購入されたお客様にもお知らせをお届けすることができます。
                </p>
              <div className="landing-page-sales-support-benefit__image-wrap">
                <img
                  src="/AfterConnection2.png"
                  alt="電子名札を介したお客様との繋がり"
                  className="landing-page-sales-support-benefit__image"
                  loading="lazy"
                />
              </div>
              </div>
            </article>
          </div>
        </div>
      </section>
    </Layout>
  );
}