// frontend/console/product/src/presentation/component/printCard.tsx

import React from "react";

import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shell/src/shared/ui/card";
import { Button } from "../../../../shell/src/shared/ui/button";

type PrintCardProps = {
  printing: boolean;
  onClick: () => void;
  printed?: boolean; // ✅ 追加: 印刷済み（印刷結果表示モード）
};

const PrintCard: React.FC<PrintCardProps> = ({ printing, onClick, printed }) => {
  return (
    <Card className="print-card">
      <CardHeader>
        <CardTitle>商品IDタグを印刷</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="print-card__content flex justify-center">
          <Button
            variant="solid"
            size="lg"
            onClick={onClick}
            // ✅ 文字列に合わせて幅を自動にする（w-full を外す）
            className="w-auto"
            disabled={printing}
          >
            {printing ? "発行中..." : printed ? "印刷結果" : "印刷"}
          </Button>
        </div>
      </CardContent>
    </Card>
  );
};

export default PrintCard;
