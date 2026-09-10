// frontend/admin/shell/src/features/avatar/presentation/components/AvatarResaleTable.tsx

import { useMemo } from "react";

import type { AvatarResale } from "../../../../shared/type/avatar";
import Table, {
  type TableColumn,
  type TableFilterOption,
} from "../../../../shared/ui/Table/Table";
import { formatDateTime } from "../../../../shared/util/dateFormat";

type AvatarResaleTableProps = {
  resales: AvatarResale[];
  onResaleClick?: (resale: AvatarResale) => void;
};

const STATUS_LABELS: Record<string, string> = {
  listing: "出品中",
  suspended: "停止中",
  sold: "売却済み",
};

const STATUS_OPTIONS: TableFilterOption[] = Object.entries(
  STATUS_LABELS,
).map(([value, label]) => ({
  value,
  label,
}));

export default function AvatarResaleTable({
  resales,
  onResaleClick,
}: AvatarResaleTableProps) {
  const columns = useMemo<TableColumn<AvatarResale>[]>(
    () => [
      {
        key: "productName",
        header: "商品名",
        render: (resale) => resale.productName || "-",
        nowrap: true,
      },
      {
        key: "tokenName",
        header: "トークン名",
        render: (resale) => resale.tokenName || "-",
        nowrap: true,
      },
      {
        key: "status",
        header: "ステータス",
        render: (resale) =>
          STATUS_LABELS[resale.status] ?? resale.status,
        filter: {
          getValue: (resale) => resale.status,
          options: STATUS_OPTIONS,
        },
        nowrap: true,
      },
      {
        key: "reportCount",
        header: "通報数",
        render: (resale) => resale.reportCount,
        sortValue: (resale) => resale.reportCount,
        nowrap: true,
      },
      {
        key: "createdAt",
        header: "登録日時",
        render: (resale) => formatDateTime(resale.createdAt),
        sortValue: (resale) => new Date(resale.createdAt).getTime(),
        nowrap: true,
      },
      {
        key: "updatedAt",
        header: "最終更新日時",
        render: (resale) =>
          resale.updatedAt ? formatDateTime(resale.updatedAt) : "-",
        sortValue: (resale) =>
          resale.updatedAt ? new Date(resale.updatedAt).getTime() : 0,
        nowrap: true,
      },
    ],
    [],
  );

  return (
    <Table
      columns={columns}
      rows={resales}
      getRowKey={(resale) => resale.id}
      onRowClick={onResaleClick}
      emptyMessage="Resaleはありません。"
      filteredEmptyMessage="条件に一致するResaleはありません。"
    />
  );
}