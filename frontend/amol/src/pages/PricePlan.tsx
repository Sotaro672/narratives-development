// frontend/amol/src/pages/PricePlan.tsx
import { useEffect } from "react";
import { useNavigate } from "react-router-dom";

import "../styles/page-layout.css";
import "../styles/price-plan-page.css";

import Layout from "../components/layout/Layout";
import Button from "../components/ui/Button";

const pricePlans = [
  {
    name: "AMOL MALL 出品",
    description:
      "AMOL MALLへ商品を出品し、AMOL上で販売・トークン移譲・レビュー投稿まで行う導入パターンです。",
    fees: [
      {
        label: "基本使用料",
        value: "月額4,990円",
      },
      {
        label: "電子名札発行手数料",
        value: "10円 / 点",
      },
      {
        label: "販売手数料",
        value: "販売額の10%",
      },
    ],
    flow: [
      "AMOL Consoleで商品設計を登録",
      "商品ごとに電子名札を発行",
      "AMOL MALLへ出品",
      "販売後、トークンが購入者のAMOL Avatar Walletへ移譲",
      "購入者はAMOL上でコンテンツ閲覧・レビュー投稿",
    ],
  },
  {
    name: "自社EC 接続",
    description:
      "既存の自社ECとAMOLを接続し、自社EC上で販売しながら、電子名札とトークン移譲をAMOLで利用する導入パターンです。",
    fees: [
      {
        label: "基本使用料",
        value: "月額4,990円",
      },
      {
        label: "電子名札発行手数料",
        value: "10円 / 点",
      },
      {
        label: "販売手数料",
        value: "なし",
      },
    ],
    developmentFee: {
      title: "接続開発費",
      description:
        "既存ECの仕様や連携範囲に応じて、個別に接続開発費が発生します。",
    },
    flow: [
      "AMOL Consoleで商品設計を登録",
      "商品ごとに電子名札を発行",
      "自社ECへ出品",
      "販売後、トークンが購入者のAMOL Avatar Walletへ移譲",
      "購入者はAMOL上でコンテンツ閲覧・レビュー投稿",
    ],
  },
];

export default function PricePlan() {
  const navigate = useNavigate();

  useEffect(() => {
    window.scrollTo({
      top: 0,
      left: 0,
      behavior: "auto",
    });
  }, []);

  return (
    <Layout title="AMOL" mode="landing">
      <section className="landing-page-section">
        <div className="landing-page-section__inner">
          <section id="price-plan" className="price-plan-page">
            <div className="price-plan-page__header">
              <h2 className="price-plan-page__title">料金プラン</h2>
              <p className="price-plan-page__description">
                AMOLではお客様の自社ECサイトとの接続工事も対応しております。
              </p>
            </div>

            <div className="price-plan-page__grid">
              {pricePlans.map((plan) => {
                const hasDevelopmentFee = Boolean(plan.developmentFee);

                return (
                  <article key={plan.name} className="price-plan-card">
                    <div className="price-plan-card__header">
                      <h3 className="price-plan-card__title">{plan.name}</h3>
                      <p className="price-plan-card__description">
                        {plan.description}
                      </p>
                    </div>

                    <div
                      className={[
                        "price-plan-card__section",
                        "price-plan-card__fee-grid",
                        hasDevelopmentFee
                          ? "price-plan-card__fee-grid--split"
                          : "price-plan-card__fee-grid--single",
                      ].join(" ")}
                    >
                      <div className="price-plan-card__fee-box">
                        <h4 className="price-plan-card__section-title">費用</h4>
                        <ul className="price-plan-card__list">
                          {plan.fees.map((fee) => (
                            <li
                              key={fee.label}
                              className="price-plan-card__list-item"
                            >
                              <span className="price-plan-card__fee-label">
                                {fee.label}
                              </span>
                              <span className="price-plan-card__fee-value">
                                {fee.value}
                              </span>
                            </li>
                          ))}
                        </ul>
                      </div>

                      {plan.developmentFee && (
                        <div className="price-plan-card__fee-box price-plan-card__development-fee-box">
                          <h4 className="price-plan-card__section-title">
                            {plan.developmentFee.title}
                          </h4>
                          <p className="price-plan-card__development-fee-description">
                            {plan.developmentFee.description}
                          </p>
                        </div>
                      )}
                    </div>

                    <div className="price-plan-card__section">
                      <h4 className="price-plan-card__section-title">
                        利用の流れ
                      </h4>
                      <ol className="price-plan-card__flow">
                        {plan.flow.map((item, index) => (
                          <li
                            key={item}
                            className={[
                              "price-plan-card__flow-item",
                              index === 2
                                ? "price-plan-card__flow-item--emphasis"
                                : "",
                            ].join(" ")}
                          >
                            {item}
                          </li>
                        ))}
                      </ol>
                    </div>
                  </article>
                );
              })}
            </div>

            <div className="page-actions">
              <Button variant="primary" onClick={() => navigate("/how-to-use")}>
                使い方解説
              </Button>
            </div>
          </section>
        </div>
      </section>
    </Layout>
  );
}