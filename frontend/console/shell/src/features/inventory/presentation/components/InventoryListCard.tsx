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
import Empty from "../../../../shared/ui/empty";
import { ErrorMessage } from "../../../../shared/ui/error";
import Text from "../../../../shared/ui/text";

import "../../../../styles/inventory.css";

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
      <CardHeader className="inventory-list-card__header">
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
          <ErrorMessage>
            出品情報の取得に失敗しました: {error}
          </ErrorMessage>
        ) : items.length === 0 ? (
          <Empty
            compact
            description="この在庫の出品はまだありません。"
          />
        ) : (
          <div className="inventory-list-card__list">
            {items.map((item) => (
              <div
                key={item.id}
                className="inventory-list-card__item"
              >
                <Button
                  type="button"
                  variant="link"
                  onClick={() => onOpenList(item.id)}
                >
                  {item.readableId || item.id}
                </Button>

                <div className="inventory-list-card__metrics">
                  <div>
                    <Text as="div" tone="muted">
                      累計注文数
                    </Text>
                    <Text
                      as="div"
                      weight="medium"
                      className="inventory-list-card__metric-value"
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
                      className="inventory-list-card__metric-value"
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