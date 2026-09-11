// frontend/admin/shell/src/features/trade/presentation/components/TradeTable.tsx

import { useMemo } from "react";

import type { Trade } from "../../../../shared/type/trade";
import Table, { type TableColumn } from "../../../../shared/ui/Table/Table";
import { formatDateTime } from "../../../../shared/util/dateFormat";
import { useResaleTrades } from "../hooks/useResaleTrades";

type TradeTableProps = {
  resaleId: string;
};

export default function TradeTable({
  resaleId,
}: TradeTableProps) {
  const { trades, loading, error, reload } = useResaleTrades(resaleId);

  const columns = useMemo<TableColumn<Trade>[]>(
    () => [
      {
        key: "buyerAvatarName",
        header: "購入者",
        render: (trade) =>
          trade.buyerAvatarName || trade.buyerAvatarId || "-",
        sortValue: (trade) =>
          trade.buyerAvatarName || trade.buyerAvatarId,
        nowrap: true,
      },
      {
        key: "commentCount",
        header: "コメント数",
        render: (trade) => trade.commentCount,
        sortValue: (trade) => trade.commentCount,
        nowrap: true,
      },
      {
        key: "reportCount",
        header: "通報",
        render: (trade) => trade.reportCount,
        sortValue: (trade) => trade.reportCount,
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