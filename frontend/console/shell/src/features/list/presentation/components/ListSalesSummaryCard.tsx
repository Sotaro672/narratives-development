// frontend/console/shell/src/features/list/presentation/components/ListSalesSummaryCard.tsx

import { Card, CardContent } from "../../../../shared/ui/card";

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
      <CardContent className="p-4">
        <div className="text-sm font-medium mb-3">販売実績</div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <div className="text-xs text-slate-500">
              累計注文数
            </div>

            <div className="mt-1 text-sm font-medium text-slate-900">
              {totalOrderCount.toLocaleString()}件
            </div>
          </div>

          <div>
            <div className="text-xs text-slate-500">
              累計売上
            </div>

            <div className="mt-1 text-sm font-medium text-slate-900">
              ¥{totalSalesAmount.toLocaleString()}
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}