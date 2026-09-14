// frontend/mall/src/features/landing/components/PricingPlanTable.tsx

export type SubscriptionPlanRow = {
  label: string;
  values: string[];
};

type PricingPlanTableProps = {
  subscriptionPlanColumns: string[];
  subscriptionPlanRows: SubscriptionPlanRow[];
};

export default function PricingPlanTable({
  subscriptionPlanColumns,
  subscriptionPlanRows,
}: PricingPlanTableProps) {
  return (
    <>
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
    </>
  );
}