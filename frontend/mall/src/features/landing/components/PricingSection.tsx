// frontend/mall/src/features/landing/components/PricingSection.tsx

type SubscriptionPlanRow = {
  label: string;
  values: string[];
};

type PricingSectionProps = {
  subscriptionPlanColumns: string[];
  subscriptionPlanRows: SubscriptionPlanRow[];
};

export default function PricingSection({
  subscriptionPlanColumns,
  subscriptionPlanRows,
}: PricingSectionProps) {
  return (
    <section
      id="pricing"
      className="landing-page-section landing-page-pricing"
    >
      <div className="landing-page-section__inner">
        <div className="landing-page-sales-support__header">
          <p className="landing-page-sales-support__eyebrow">
            利用料金
          </p>

          <h2 className="landing-page-section__title landing-page-sales-support__title">
            本番運用時の料金体系
          </h2>

          <p className="landing-page-card__text landing-page-sales-support__lead">
            現在試作品段階です。本番運用リリース時は以下の料金体系を予定しております。
          </p>
        </div>

        <div className="landing-page-pricing-grid">
          <article className="landing-page-pricing-card">
            <p className="landing-page-pricing-card__label">
              基本利用料金
            </p>

            <h3 className="landing-page-pricing-card__price">
              4,990円/月～
            </h3>

            <p className="landing-page-pricing-card__text">
              試験運用価格であり、今後金額が上下する可能性があります。
            </p>
          </article>

          <article className="landing-page-pricing-card">
            <p className="landing-page-pricing-card__label">
              電子名札発行手数料
            </p>

            <h3 className="landing-page-pricing-card__price">
              10円/点
            </h3>

            <p className="landing-page-pricing-card__text">
              発行した点数に応じて課金されます。
            </p>
          </article>

          <article className="landing-page-pricing-card">
            <p className="landing-page-pricing-card__label">
              販売手数料
            </p>

            <h3 className="landing-page-pricing-card__price">
              売上の10%
            </h3>

            <p className="landing-page-pricing-card__text">
              AMOLモール上で商品が販売された場合に発生します。
            </p>
          </article>
        </div>

        <h3 className="price-plan-page__table-title">
          基本料金表
        </h3>

        <div className="price-plan-table-wrap">
          <table className="price-plan-table">
            <thead>
              <tr>
                <th
                  scope="col"
                  className="price-plan-table__corner"
                >
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
                  <th scope="row">
                    {row.label}
                  </th>

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
  );
}