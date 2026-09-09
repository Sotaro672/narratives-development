// frontend/admin/shell/src/features/company/presentation/components/ContractListTable.tsx

import { useMemo } from "react";

import type { ContractListRow } from "../../../../shared/type/contractDetail";
import Table, { type TableColumn } from "../../../../shared/ui/Table/Table";
import { formatDateTime } from "../../../../shared/util/dateFormat";

type ContractListTableProps = {
  lists: ContractListRow[];
  onListClick?: (list: ContractListRow) => void;
};

function formatListStatus(status: string): string {
  switch (status) {
    case "listing":
      return "出品中";
    case "suspended":
      return "停止中";
    default:
      return status || "-";
  }
}

export default function ContractListTable({
  lists,
  onListClick,
}: ContractListTableProps) {
  const columns = useMemo<TableColumn<ContractListRow>[]>(
    () => [
      {
        key: "title",
        header: "出品",
        render: (list) => list.title || list.readableId || list.id,
        sortValue: (list) => list.title || list.readableId || list.id,
        filter: {
          getValue: (list) =>
            [list.title, list.readableId, list.id].filter(Boolean).join(" "),
          placeholder: "出品名・IDで絞り込み",
        },
        nowrap: true,
      },
      {
        key: "productName",
        header: "商品",
        render: (list) => list.productName || "-",
        sortValue: (list) => list.productName,
        filter: {
          getValue: (list) => list.productName,
          placeholder: "商品名で絞り込み",
        },
        nowrap: true,
      },
      {
        key: "tokenName",
        header: "トークン設計",
        render: (list) => list.tokenName || "-",
        sortValue: (list) => list.tokenName,
        filter: {
          getValue: (list) => list.tokenName,
          placeholder: "トークン名で絞り込み",
        },
        nowrap: true,
      },
      {
        key: "brandName",
        header: "ブランド",
        render: (list) => list.brandName || "-",
        sortValue: (list) => list.brandName,
        filter: {
          getValue: (list) => list.brandName,
          placeholder: "ブランド名で絞り込み",
        },
        nowrap: true,
      },
      {
        key: "assigneeName",
        header: "担当者",
        render: (list) => list.assigneeName || "-",
        sortValue: (list) => list.assigneeName,
        filter: {
          getValue: (list) => list.assigneeName,
          placeholder: "担当者名で絞り込み",
        },
        nowrap: true,
      },
      {
        key: "status",
        header: "状態",
        render: (list) => formatListStatus(list.status),
        sortValue: (list) => formatListStatus(list.status),
        filter: {
          getValue: (list) => list.status,
          options: [
            { value: "listing", label: "出品中" },
            { value: "suspended", label: "停止中" },
          ],
        },
        nowrap: true,
      },
      {
        key: "createdAt",
        header: "作成日時",
        render: (list) => formatDateTime(list.createdAt),
        sortValue: (list) => new Date(list.createdAt).getTime(),
        nowrap: true,
      },
    ],
    [],
  );

  return (
    <Table
      columns={columns}
      rows={lists}
      getRowKey={(list) => list.id}
      onRowClick={onListClick}
      emptyMessage="出品はありません。"
      filteredEmptyMessage="条件に一致する出品はありません。"
    />
  );
}