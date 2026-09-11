// frontend\admin\shell\src\features\announcement\presentation\components\AnnouncementTable.tsx
import { useMemo } from "react";

import type { ContractAnnouncementRow } from "../../../../shared/type/contractDetail";
import Table, {
  type TableColumn,
  type TableFilterOption,
} from "../../../../shared/ui/Table/Table";
import { formatDateTime } from "../../../../shared/util/dateFormat";

type ContractAnnouncementTableProps = {
  announcements: ContractAnnouncementRow[];
};

const STATUS_LABELS: Record<string, string> = {
  published: "公開済み",
  draft: "下書き",
};

const STATUS_OPTIONS: TableFilterOption[] = Object.entries(
  STATUS_LABELS,
).map(([value, label]) => ({
  value,
  label,
}));

export default function ContractAnnouncementTable({
  announcements,
}: ContractAnnouncementTableProps) {
  const columns = useMemo<TableColumn<ContractAnnouncementRow>[]>(
    () => [
      {
        key: "title",
        header: "タイトル",
        render: (announcement) => announcement.title || "-",
        sortValue: (announcement) => announcement.title,
        nowrap: true,
      },
      {
        key: "tokenName",
        header: "対象トークン",
        render: (announcement) => announcement.tokenName || "-",
        sortValue: (announcement) => announcement.tokenName,
        nowrap: true,
      },
      {
        key: "status",
        header: "状態",
        render: (announcement) =>
          announcement.published
            ? STATUS_LABELS.published
            : STATUS_LABELS.draft,
        filter: {
          getValue: (announcement) =>
            announcement.published ? "published" : "draft",
          options: STATUS_OPTIONS,
        },
        nowrap: true,
      },
      {
        key: "targetAvatarCount",
        header: "送信対象数",
        render: (announcement) =>
          announcement.targetAvatarCount.toLocaleString(),
        sortValue: (announcement) => announcement.targetAvatarCount,
        nowrap: true,
      },
      {
        key: "createdAt",
        header: "作成日時",
        render: (announcement) =>
          announcement.createdAt
            ? formatDateTime(announcement.createdAt)
            : "-",
        sortValue: (announcement) =>
          announcement.createdAt
            ? new Date(announcement.createdAt).getTime()
            : 0,
        nowrap: true,
      },
      {
        key: "updatedAt",
        header: "更新日時",
        render: (announcement) =>
          announcement.updatedAt
            ? formatDateTime(announcement.updatedAt)
            : "-",
        sortValue: (announcement) =>
          announcement.updatedAt
            ? new Date(announcement.updatedAt).getTime()
            : 0,
        nowrap: true,
      },
    ],
    [],
  );

  return (
    <Table
      columns={columns}
      rows={announcements}
      getRowKey={(announcement) => announcement.id}
      emptyMessage="告知はありません。"
      filteredEmptyMessage="条件に一致する告知はありません。"
    />
  );
}