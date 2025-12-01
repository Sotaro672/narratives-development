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
};

const PrintCard: React.FC<PrintCardProps> = ({ printing, onClick }) => {
  return (
    <Card className="print-card">
      <CardHeader>
        <CardTitle>商品IDタグ用 Product を発行する</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="print-card__content">
          <Button
            variant="solid"
            size="lg"
            onClick={onClick}
            className="w-full max-w-xs"
            disabled={printing}
          >
            {printing ? "発行中..." : "印刷用 Product を発行"}
          </Button>
        </div>
      </CardContent>
    </Card>
  );
};

export default PrintCard;
