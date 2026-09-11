// frontend/admin/shell/src/features/trade/presentation/components/TradeTable.tsx

import { useMemo } from "react";

import type { Trade } from "../../../../shared/type/trade";
import Table, {
  type TableColumn,
  type TableFilterOption,
} from "../../../../shared/ui/Table/Table";
import { formatDateTime } from "../../../../shared/util/dateFormat";
import { useResaleTrades } from "../hooks/useResaleTrades";

type TradeTableProps = {
  resaleId: string;
};

const STATUS_LABELS: Record<Trade["status"], string> = {
  active: "進行中",
  closed: "終了",
};

const STATUS_OPTIONS: TableFilterOption[] = Object.entries(
  STATUS_LABELS,
).map(([value, label]) => ({
  value,
  label,
}));

export default function TradeTable({
  resaleId,
}: TradeTableProps) {
  const { trades, loading, error, reload } = useResaleTrades(resaleId);

  const columns = useMemo<TableColumn<Trade>[]>(
    () => [
      {
        key: "id",
        header: "取引ID",
        render: (trade) => trade.id,
        nowrap: true,
      },
      {
        key: "orderId",
        header: "注文ID",
        render: (trade) => trade.orderId,
        nowrap: true,
      },
      {
        key: "orderItemIndex",
        header: "明細",
        render: (trade) => trade.orderItemIndex,
        sortValue: (trade) => trade.orderItemIndex,
        nowrap: true,
      },
      {
        key: "buyerAvatarId",
        header: "購入者",
        render: (trade) => trade.buyerAvatarId || "-",
        nowrap: true,
      },
      {
        key: "seller",
        header: "出品者",
        render: (trade) => {
          if (trade.sellerType === "avatar") {
            return trade.sellerAvatarId || "-";
          }

          return trade.sellerBrandId || trade.sellerCompanyId || "-";
        },
        nowrap: true,
      },
      {
        key: "status",
        header: "状態",
        render: (trade) => STATUS_LABELS[trade.status] ?? trade.status,
        sortValue: (trade) => STATUS_LABELS[trade.status] ?? trade.status,
        filter: {
          getValue: (trade) => trade.status,
          options: STATUS_OPTIONS,
        },
        nowrap: true,
      },
      {
        key: "createdAt",
        header: "作成日時",
        render: (trade) => formatDateTime(trade.createdAt),
        sortValue: (trade) => new Date(trade.createdAt).getTime(),
        nowrap: true,
      },
      {
        key: "updatedAt",
        header: "更新日時",
        render: (trade) => formatDateTime(trade.updatedAt),
        sortValue: (trade) => new Date(trade.updatedAt).getTime(),
        nowrap: true,
      },
    ],
    [],
  );

  if (loading && !trades) {
    return <p>取引情報を取得しています...</p>;
  }

  if (error && !trades) {
    return (
      <div role="alert">
        <p>取引情報を取得できませんでした。</p>
        <p>{error}</p>
        <button type="button" onClick={() => void reload()}>
          再読み込み
        </button>
      </div>
    );
  }

  return (
    <Table
      columns={columns}
      rows={trades?.items ?? []}
      getRowKey={(trade) => trade.id}
      emptyMessage="取引はありません。"
      filteredEmptyMessage="条件に一致する取引はありません。"
    />
  );
}