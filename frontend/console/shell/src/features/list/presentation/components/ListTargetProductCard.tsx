// frontend/console/shell/src/features/list/presentation/components/ListTargetProductCard.tsx

import { Card, CardContent } from "../../../../shared/ui/card";

type ListTargetProductCardProps = {
  productName: string;
  tokenName: string;
};

export default function ListTargetProductCard({
  productName,
  tokenName,
}: ListTargetProductCardProps) {
  const displayProductName = productName || "未選択";
  const displayTokenName = tokenName || "未選択";

  return (
    <Card>
      <CardContent className="p-4">
        <div className="text-sm font-medium mb-2">対象商品</div>
        <div className="text-sm text-slate-800 break-words">
          {displayProductName} / {displayTokenName}
        </div>
      </CardContent>
    </Card>
  );
}