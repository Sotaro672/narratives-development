//frontend\console\shell\src\features\inventory\presentation\components\InventoryListCard.tsx
import * as React from "react";
import { Tag } from "lucide-react";

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";
import { Button } from "../../../../shared/ui/button";

export type InventoryListCardProps = {
  onList: () => void;
};

const InventoryListCard: React.FC<InventoryListCardProps> = ({
  onList,
}) => {
  return (
    <Card>
      <CardHeader>
        <CardTitle>出品</CardTitle>
      </CardHeader>

      <CardContent className="space-y-4">
        <p className="text-sm text-slate-500">
          この在庫をマーケットへ出品します。
        </p>

        <Button
          type="button"
          className="w-full"
          onClick={onList}
        >
          <Tag size={16} className="mr-2" />
          出品する
        </Button>
      </CardContent>
    </Card>
  );
};

export default InventoryListCard;