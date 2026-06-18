// frontend/amol/src/pages/PricePlan.tsx
import "../styles/page-layout.css";
import "../styles/vision-page.css";

import Layout from "../components/layout/Layout";

const pricePlans = [
  {
    name: "Starter",
    price: "要相談",
    description:
      "小規模にAMOLを試したい事業者向けのプランです。まずは一部商品への電子名札付与から開始できます。",
    features: [
      "電子名札付商品の登録",
      "商品の真正性証明",
      "基本的な商品情報の表示",
    ],
  },
  {
    name: "Business",
    price: "要相談",
    description:
      "ブランドやメーカーが本格的に商品管理・販売へ活用するためのプランです。",
    features: [
      "複数商品の電子名札管理",
      "検品・ミント申請フロー",
      "販売・注文管理",
      "ブランドページ運用",
    ],
    recommended: true,
  },
  {
    name: "Enterprise",
    price: "要相談",
    description:
      "大規模な商品流通や独自運用に合わせて、個別設計・連携を行うプランです。",
    features: [
      "個別要件に応じた導入支援",
      "既存システムとの連携相談",
      "大規模運用向けの管理設計",
      "専用サポート",
    ],
  },
];

export default function PricePlan() {
  return (
    <Layout title="AMOL" mode="landing">
      <section className="landing-page-section">
        <div className="landing-page-section__inner">
          <section id="price-plan" className="landing-page-vision">
            <div className="landing-page-vision__header">
              <h2 className="landing-page-vision__title">料金プラン</h2>
              <p className="landing-page-vision__lead">
                導入規模に合わせたプランをご用意しています
              </p>
              <p className="landing-page-vision__description">
                AMOLは、事業者の商品数・運用体制・導入目的に応じて、
                最適なプランをご提案します。詳細な料金はお問い合わせください。
              </p>
            </div>

            <div className="landing-page-vision__steps">
              {pricePlans.map((plan, index) => (
                <div key={plan.name} className="landing-page-vision-step-wrap">
                  <article
                    className={[
                      "landing-page-vision-step",
                      plan.recommended
                        ? "landing-page-vision-step--current"
                        : "",
                    ]
                      .filter(Boolean)
                      .join(" ")}
                  >
                    <div className="landing-page-vision-step__content">
                      <div className="landing-page-vision-step__meta">
                        <span className="landing-page-vision-step__step">
                          Plan {index + 1}
                        </span>
                        {plan.recommended && (
                          <span className="landing-page-vision-step__badge">
                            おすすめ
                          </span>
                        )}
                      </div>

                      <h3 className="landing-page-vision-step__title">
                        {plan.name}
                      </h3>
                      <p className="landing-page-vision-step__description">
                        {plan.price}
                      </p>
                      <p className="landing-page-vision-step__description">
                        {plan.description}
                      </p>

                      <ul className="landing-page-team-card__career">
                        {plan.features.map((feature) => (
                          <li
                            key={feature}
                            className="landing-page-team-card__career-item"
                          >
                            {feature}
                          </li>
                        ))}
                      </ul>
                    </div>
                  </article>

                  {index < pricePlans.length - 1 && (
                    <div className="landing-page-vision-step__connector" />
                  )}
                </div>
              ))}
            </div>
          </section>
        </div>
      </section>
    </Layout>
  );
}