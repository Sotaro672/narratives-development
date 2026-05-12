// frontend/amol/src/pages/LandingPage.tsx
import { useNavigate } from "react-router-dom";

import "../styles/page-layout.css";
import "../styles/landing-page.css";

import Layout from "../components/layout/Layout";
import Button from "../components/ui/Button";

export default function LandingPage() {
  const navigate = useNavigate();

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

            <article className="landing-page-feature-card">
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
    </Layout>
  );
}