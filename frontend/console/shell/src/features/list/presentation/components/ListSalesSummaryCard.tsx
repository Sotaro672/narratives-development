// frontend/console/shell/src/features/list/presentation/components/ListSalesSummaryCard.tsx

import { Card, CardContent } from "../../../../shared/ui/card";

import "../../../../styles/list.css";

type ListSalesSummaryCardProps = {
  totalOrderCount: number;
  totalSalesAmount: number;
};

export default function ListSalesSummaryCard({
  totalOrderCount,
  totalSalesAmount,
}: ListSalesSummaryCardProps) {
  return (
    <Card>
      <CardContent className="list-sales-summary__content">
        <div className="list-sales-summary__title">
          販売実績
        </div>

        <div className="list-sales-summary__grid">
          <div>
            <div className="list-sales-summary__label">
              累計注文数
            </div>

            <div className="list-sales-summary__value">
              {totalOrderCount.toLocaleString()}件
            </div>
          </div>

          <div>
            <div className="list-sales-summary__label">
              累計売上
            </div>

            <div className="list-sales-summary__value">
              ¥{totalSalesAmount.toLocaleString()}
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}