// frontend/console/shell/src/features/inventory/presentation/components/InventoryListCard.tsx

import * as React from "react";
import { Tag } from "lucide-react";

import { Button } from "../../../../shared/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";
import Text from "../../../../shared/ui/text";

export type InventoryListCardItem = {
  id: string;
  readableId: string;
  totalOrderCount: number;
  totalSalesAmount: number;
};

export type InventoryListCardProps = {
  items: InventoryListCardItem[];
  loading?: boolean;
  error?: string | null;
  onList?: () => void;
  onOpenList: (listId: string) => void;
};

const InventoryListCard: React.FC<InventoryListCardProps> = ({
  items,
  loading = false,
  error = null,
  onList,
  onOpenList,
}) => {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between gap-3">
        <CardTitle>出品</CardTitle>

        {onList ? (
          <Button type="button" onClick={onList}>
            <Tag size={16} />
            出品
          </Button>
        ) : null}
      </CardHeader>

      <CardContent>
        {loading ? (
          <Text as="div" tone="muted">
            出品情報を読み込み中です...
          </Text>
        ) : error ? (
          <Text as="div" tone="destructive" role="alert">
            出品情報の取得に失敗しました: {error}
          </Text>
        ) : items.length === 0 ? (
          <Text as="div" tone="muted">
            この在庫の出品はまだありません。
          </Text>
        ) : (
          <div className="divide-y divide-slate-200 rounded-md border border-slate-200">
            {items.map((item) => (
              <div
                key={item.id}
                className="px-3 py-3"
              >
                <Button
                  type="button"
                  variant="link"
                  onClick={() => onOpenList(item.id)}
                >
                  {item.readableId || item.id}
                </Button>

                <div className="mt-2 grid grid-cols-2 gap-3">
                  <div>
                    <Text as="div" tone="muted">
                      累計注文数
                    </Text>
                    <Text
                      as="div"
                      weight="medium"
                      className="mt-1"
                    >
                      {item.totalOrderCount.toLocaleString()}件
                    </Text>
                  </div>

                  <div>
                    <Text as="div" tone="muted">
                      累計売上
                    </Text>
                    <Text
                      as="div"
                      weight="medium"
                      className="mt-1"
                    >
                      ¥{item.totalSalesAmount.toLocaleString()}
                    </Text>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
};

export default InventoryListCard;