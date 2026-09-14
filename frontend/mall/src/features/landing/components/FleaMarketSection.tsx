// frontend/mall/src/features/landing/components/FleaMarketSection.tsx

import type { RefObject } from "react";

type FleaMarketSectionProps = {
  fleaMarketEyebrowRef: RefObject<HTMLParagraphElement>;
};

export default function FleaMarketSection({
  fleaMarketEyebrowRef,
}: FleaMarketSectionProps) {
  return (
    <section
      id="flea-market"
      className="landing-page-section landing-page-sales-support"
    >
      <div className="landing-page-section__inner">
        <div className="landing-page-sales-support__header">
          <p
            ref={fleaMarketEyebrowRef}
            className="landing-page-sales-support__eyebrow"
          >
            フリーマーケット
          </p>

          <h2 className="landing-page-section__title landing-page-sales-support__title">
            利益を上げながら模倣品対策
          </h2>

          <p className="landing-page-card__text landing-page-sales-support__lead">
            ブランドが利益を回収できるフリマを提供します。
          </p>
        </div>

        <div className="landing-page-sales-support__benefits">
          <article className="landing-page-sales-support-benefit landing-page-sales-support-benefit--with-image">
            <div className="landing-page-sales-support-benefit__content">
              <p className="landing-page-sales-support-card__label">
                真贋証明をフリマにまで届ける
              </p>

              <h3 className="landing-page-sales-support-benefit__title">
                真贋証明付きで出品できるフリマ
              </h3>

              <p className="landing-page-sales-support-benefit__text">
                AMOLのフリマでは製造時にブロックチェーントークンを連携した電子名札を発行した商品のみ出品されます。
                本物であることが証明されているため、従来のフリマよりも高く販売できることが期待できます。
                結果、店舗での購入価格とフリマでの販売価格の差額が縮まり、
                これまで手が届かなかったお客様を値下げをせずに新規顧客として迎えられるようになります。
              </p>

              <div className="landing-page-sales-support-benefit__image-wrap">
                <img
                  src="/freemarket_expect.jpg"
                  alt="二次流通市場でブランドに利益が還元されにくい状態"
                  className="landing-page-sales-support-benefit__image"
                  loading="lazy"
                />
              </div>
            </div>
          </article>

          <article className="landing-page-sales-support-benefit landing-page-sales-support-benefit--with-image">
            <div className="landing-page-sales-support-benefit__content">
              <p className="landing-page-sales-support-card__label">
                ブランドが利益を回収できるフリマ
              </p>

              <h3 className="landing-page-sales-support-benefit__title">
                フリーマーケット売上の5%をブランド様に還元
              </h3>

              <p className="landing-page-sales-support-benefit__text">
                AMOL登録商品がフリーマーケットで販売された場合、売上の5%をブランド様に還元します。長く使われる商品ほど、二次流通市場でも継続的な利益機会をつくることができます。
              </p>

              <div className="landing-page-sales-support-benefit__image-wrap">
                <img
                  src="/freemarket_5percent.jpg"
                  alt="フリーマーケットでの売上の一部がブランドに還元される状態"
                  className="landing-page-sales-support-benefit__image"
                  loading="lazy"
                />
              </div>
            </div>
          </article>
        </div>
      </div>
    </section>
  );
}