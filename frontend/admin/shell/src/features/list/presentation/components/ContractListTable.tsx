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

function createOptions(values: Array<string | null | undefined>) {
  return Array.from(
    new Set(
      values
        .map((value) => value?.trim())
        .filter((value): value is string => Boolean(value)),
    ),
  )
    .sort((a, b) => a.localeCompare(b, "ja"))
    .map((value) => ({
      value,
      label: value,
    }));
}

export default function ContractListTable({
  lists,
  onListClick,
}: ContractListTableProps) {
  const productOptions = useMemo(
    () => createOptions(lists.map((list) => list.productName)),
    [lists],
  );

  const tokenOptions = useMemo(
    () => createOptions(lists.map((list) => list.tokenName)),
    [lists],
  );

  const brandOptions = useMemo(
    () => createOptions(lists.map((list) => list.brandName)),
    [lists],
  );

  const columns = useMemo<TableColumn<ContractListRow>[]>(
    () => [
      {
        key: "title",
        header: "出品",
        render: (list) => list.title || list.readableId || list.id,
        nowrap: true,
      },
      {
        key: "productName",
        header: "商品",
        render: (list) => list.productName || "-",
        filter: {
          getValue: (list) => list.productName,
          options: productOptions,
        },
        nowrap: true,
      },
      {
        key: "tokenName",
        header: "トークン設計",
        render: (list) => list.tokenName || "-",
        filter: {
          getValue: (list) => list.tokenName,
          options: tokenOptions,
        },
        nowrap: true,
      },
      {
        key: "brandName",
        header: "ブランド",
        render: (list) => list.brandName || "-",
        filter: {
          getValue: (list) => list.brandName,
          options: brandOptions,
        },
        nowrap: true,
      },
      {
        key: "reportCount",
        header: "通報数",
        render: (list) => list.reportCount,
        sortValue: (list) => list.reportCount,
        nowrap: true,
      },
      {
        key: "status",
        header: "状態",
        render: (list) => formatListStatus(list.status),
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
    [brandOptions, productOptions, tokenOptions],
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