//frontend\amol\src\pages\PricePlan.tsx
import { useEffect } from "react";
import { useNavigate } from "react-router-dom";

import "../styles/page-layout.css";
import "../styles/price-plan-page.css";

import Layout from "../components/layout/Layout";
import Button from "../components/ui/Button";

const subscriptionPlanColumns = ["Starter", "Simple", "Grow", "Advanced"];

const subscriptionPlanRows = [
  {
    label: "料金",
    values: ["4,990円/月", "9,990円/月", "19,990円/月", "29,990円/月"],
  },
  {
    label: "ミント",
    values: ["〇", "〇", "〇", "〇"],
  },
  {
    label: "お問い合わせ",
    values: ["〇", "〇", "〇", "〇"],
  },
  {
    label: "レビュー",
    values: ["×", "〇", "〇", "〇"],
  },
  {
    label: "一斉告知",
    values: ["×", "×", "〇", "〇"],
  },
  {
    label: "メンバー招待",
    values: ["×", "×", "×", "〇"],
  },
  {
    label: "ブランド数",
    values: ["1", "1", "1", "無制限"],
  },
];

const feeCards = [
  {
    label: "ミント料金",
    value: "10円/点",
    description: "ミントには発行点数分、課金いたします。",
  },
  {
    label: "販売手数料",
    value: "売上の10%",
    description:
      "自社ECと接続する場合は販売手数料は発生しません。",
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
                    </article>
                  ))}
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