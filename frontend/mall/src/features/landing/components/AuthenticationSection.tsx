// frontend/mall/src/features/landing/components/AuthenticationSection.tsx

import type { RefObject } from "react";

import blockchainNetworkImage from "../assets/BlockchainNetwork.png";
import scanningImage from "../assets/Scaning.png";
import AntiCopySection from "./AntiCopySection";

type AuthenticationSectionProps = {
  authenticationEyebrowRef: RefObject<HTMLParagraphElement>;
};

export default function AuthenticationSection({
  authenticationEyebrowRef,
}: AuthenticationSectionProps) {
  return (
    <section
      id="authentication"
      className="landing-page-section landing-page-service-case"
    >
      <div className="landing-page-section__inner">
        <div className="landing-page-service-case__header">
          <p
            ref={authenticationEyebrowRef}
            className="landing-page-sales-support__eyebrow"
          >
            真贋証明
          </p>

          <h2 className="landing-page-section__title landing-page-service-case__title">
            電子名札で見える商品の変遷
          </h2>

          <p className="landing-page-card__text landing-page-service-case__lead">
            届いた商品のQRコードをスキャンするだけで貴方の商品の所有権を閲覧、記録することができます。
          </p>
        </div>

        <div className="landing-page-service-case__top-grid">
          <article className="landing-page-service-case-card landing-page-service-case-card--proof">
            <div className="landing-page-service-case-card__content">
              <h3 className="landing-page-service-case-card__title">
                QRコードを読み取るだけで、現在の所有者を確認できる
              </h3>

              <p className="landing-page-service-case-card__text">
                購入した商品のQRコードを読み取るだけで所有者が購入されたお客様のアバター名に自動的に更新されます。
              </p>
            </div>

            <div className="landing-page-service-case-card__image-wrap">
              <img
                src={scanningImage}
                alt="日本酒ラベルのQRコードを読み取り、所有者情報が更新される流れ"
                className="landing-page-service-case-card__image"
                loading="lazy"
              />
            </div>
          </article>

          <article className="landing-page-service-case-card landing-page-service-case-card--sales">
            <div className="landing-page-service-case-card__content">
              <h3 className="landing-page-service-case-card__title">
                あなたの所有権を堅牢に守る仕組み
              </h3>

              <p className="landing-page-service-case-card__text">
                商品の所有権はブロックチェーンネットワークで記録され、外部からの改ざん攻撃から貴方の所有権を堅牢に守ります。
              </p>
            </div>

            <div className="landing-page-service-case-card__image-wrap">
              <img
                src={blockchainNetworkImage}
                alt="世界中のサーバーに所有者情報が記録される図"
                className="landing-page-service-case-card__image"
                loading="lazy"
              />
            </div>
          </article>
        </div>

        <AntiCopySection />
      </div>
    </section>
  );
}