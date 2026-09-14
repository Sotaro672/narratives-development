// frontend/mall/src/features/landing/components/LandingFeatureOverview.tsx

type LandingFeatureOverviewProps = {
  onAuthenticationClick: () => void;
  onFleaMarketClick: () => void;
  onSalesSupportClick: () => void;
};

export default function LandingFeatureOverview({
  onAuthenticationClick,
  onFleaMarketClick,
  onSalesSupportClick,
}: LandingFeatureOverviewProps) {
  const handleKeyDown = (
    event: React.KeyboardEvent<HTMLElement>,
    onActivate: () => void,
  ) => {
    if (
      event.key !== "Enter" &&
      event.key !== " "
    ) {
      return;
    }

    event.preventDefault();
    onActivate();
  };

  return (
    <section className="landing-page-section">
      <div className="landing-page-section__inner">
        <div className="landing-page-feature-grid">
          <article
            className="landing-page-feature-card landing-page-feature-card--clickable"
            role="button"
            tabIndex={0}
            aria-label="真贋証明の詳細へ移動"
            onClick={onAuthenticationClick}
            onKeyDown={(event) =>
              handleKeyDown(
                event,
                onAuthenticationClick,
              )
            }
          >
            <h2 className="landing-page-feature-card__title">
              真贋証明
            </h2>

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
            aria-label="フリーマーケットの詳細へ移動"
            onClick={onFleaMarketClick}
            onKeyDown={(event) =>
              handleKeyDown(
                event,
                onFleaMarketClick,
              )
            }
          >
            <h2 className="landing-page-feature-card__title">
              フリーマーケット
            </h2>

            <p className="landing-page-feature-card__text">
              フリーマーケットでの売上の5%をブランド様に還元します。
            </p>

            <div className="landing-page-feature-card__image-placeholder">
              <img
                src="/2ndCustomer.png"
                alt="フリーマーケットで二次流通した商品の所有者が更新される図"
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
            onClick={onSalesSupportClick}
            onKeyDown={(event) =>
              handleKeyDown(
                event,
                onSalesSupportClick,
              )
            }
          >
            <h2 className="landing-page-feature-card__title">
              営業支援
            </h2>

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
  );
}