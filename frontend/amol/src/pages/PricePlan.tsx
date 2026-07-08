//frontend\amol\src\pages\PricePlan.tsx
import { useEffect } from "react";
import { useNavigate } from "react-router-dom";

import "../styles/page-layout.css";
import "../styles/price-plan-page.css";

import Layout from "../components/layout/Layout";
import Button from "../components/ui/Button";

const subscriptionPlanColumns = [
  "Starter",
  "Simple",
  "Grow",
  "Advanced",
  "Enterprise",
];

const subscriptionPlanRows = [
  {
    label: "料金",
    values: [
      "4,990円/月",
      "9,990円/月",
      "19,990円/月",
      "29,990円/月",
      "別途相談",
    ],
  },
  {
    label: "電子名札発行",
    values: ["〇", "〇", "〇", "〇", "〇"],
  },
  {
    label: "お問い合わせ",
    values: ["〇", "〇", "〇", "〇", "〇"],
  },
  {
    label: "レビュー",
    values: ["×", "〇", "〇", "〇", "〇"],
  },
  {
    label: "一斉告知",
    values: ["×", "×", "〇", "〇", "〇"],
  },
  {
    label: "メンバー招待",
    values: ["×", "×", "×", "〇", "〇"],
  },
  {
    label: "ブランド数",
    values: ["1", "1", "1", "無制限", "無制限"],
  },
];

const feeCards = [
  {
    label: "電子名札発行手数料",
    value: "10円/点",
    description: "発行点数分課金いたします。",
  },
  {
    label: "モール販売手数料",
    value: "売上の10%",
    description: "AMOLモール上で商品が販売された場合に発生します。",
    emphasis: "自社ECと接続する場合は販売手数料は発生しません。",
  },
  {
    label: "自社ECとの接続工事費",
    value: "別途相談",
    description:
      "接続先ECの仕様、必要な連携範囲、開発内容に応じて個別にお見積もりいたします。",
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
            </div>

            <section className="price-plan-page__subscription">
              <div className="price-plan-page__subscription-content">
                <div className="price-plan-fee-cards">
                  {feeCards.map((card) => (
                    <article key={card.label} className="price-plan-fee-card">
                      <p className="price-plan-fee-card__label">
                        {card.label}
                      </p>
                      <p className="price-plan-fee-card__value">
                        {card.value}
                      </p>
                      <p className="price-plan-fee-card__description">
                        {card.description}
                      </p>
                      {card.emphasis ? (
                        <p className="price-plan-fee-card__description price-plan-fee-card__description--emphasis">
                          {card.emphasis}
                        </p>
                      ) : null}
                    </article>
                  ))}
                </div>

                <h3 className="price-plan-page__table-title">基本料金表</h3>

                <div className="price-plan-table-wrap">
                  <table className="price-plan-table">
                    <thead>
                      <tr>
                        <th scope="col" className="price-plan-table__corner">
                          プラン
                        </th>
                        {subscriptionPlanColumns.map((column) => (
                          <th key={column} scope="col">
                            {column}
                          </th>
                        ))}
                      </tr>
                    </thead>

                    <tbody>
                      {subscriptionPlanRows.map((row) => (
                        <tr key={row.label}>
                          <th scope="row">{row.label}</th>
                          {row.values.map((value, index) => (
                            <td
                              key={`${row.label}-${subscriptionPlanColumns[index]}`}
                              className={
                                value === "〇"
                                  ? "price-plan-table__available"
                                  : value === "×"
                                    ? "price-plan-table__unavailable"
                                    : ""
                              }
                            >
                              {value}
                            </td>
                          ))}
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            </section>

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