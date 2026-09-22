// frontend/console/shell/src/features/list/presentation/components/ListTargetProductCard.tsx

import { Card, CardContent } from "../../../../shared/ui/card";

import "../../../../styles/list.css";

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
      <CardContent className="list-target-product__content">
        <div className="list-target-product__title">
          対象商品
        </div>

        <div className="list-target-product__value">
          {displayProductName} / {displayTokenName}
        </div>
      </CardContent>
    </Card>
  );
}