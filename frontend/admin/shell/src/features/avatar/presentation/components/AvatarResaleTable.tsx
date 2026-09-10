// frontend/admin/shell/src/features/avatar/presentation/components/AvatarResaleTable.tsx

import { useMemo } from "react";

import type { AvatarResale } from "../../../../shared/type/avatar";
import Table, { type TableColumn } from "../../../../shared/ui/Table/Table";
import { formatDateTime } from "../../../../shared/util/dateFormat";

type AvatarResaleTableProps = {
  resales: AvatarResale[];
};

const STATUS_LABELS: Record<string, string> = {
  listing: "出品中",
  suspended: "停止中",
  sold: "売却済み",
};

export default function AvatarResaleTable({
  resales,
}: AvatarResaleTableProps) {
  const columns = useMemo<TableColumn<AvatarResale>[]>(
    () => [
      {
        key: "status",
        header: "ステータス",
        render: (resale) => STATUS_LABELS[resale.status] ?? resale.status,
        sortValue: (resale) => resale.status,
        nowrap: true,
      },
      {
        key: "price",
        header: "価格",
        render: (resale) => `${resale.price.toLocaleString("ja-JP")}円`,
        sortValue: (resale) => resale.price,
        nowrap: true,
      },
      {
        key: "condition",
        header: "商品の状態",
        render: (resale) => resale.condition || "-",
        sortValue: (resale) => resale.condition,
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
      emptyMessage="Resaleはありません。"
      filteredEmptyMessage="条件に一致するResaleはありません。"
    />
  );
}