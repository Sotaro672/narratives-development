// frontend/admin/shell/src/features/avatar/presentation/components/AvatarTable.tsx

import { useMemo } from "react";

import type { Avatar } from "../../../../shared/type/avatar";
import Table, {
  type TableColumn,
} from "../../../../shared/ui/Table/Table";
import { formatDateTime } from "../../../../shared/util/dateFormat";

type AvatarTableProps = {
  avatars: Avatar[];
};

export default function AvatarTable({
  avatars,
}: AvatarTableProps) {
  const columns = useMemo<TableColumn<Avatar>[]>(
    () => [
      {
        key: "avatarName",
        header: "アバター名",
        render: (avatar) => avatar.avatarName || "-",
        nowrap: true,
      },
      {
        key: "userName",
        header: "ユーザー氏名",
        render: (avatar) => avatar.userName || "-",
        nowrap: true,
      },
      {
        key: "reportCount",
        header: "通報数",
        render: (avatar) => avatar.reportCount,
        nowrap: true,
      },
      {
        key: "createdAt",
        header: "登録日時",
        render: (avatar) => formatDateTime(avatar.createdAt),
        sortValue: (avatar) => new Date(avatar.createdAt).getTime(),
        nowrap: true,
      },
      {
        key: "updatedAt",
        header: "最終更新日時",
        render: (avatar) => formatDateTime(avatar.updatedAt),
        sortValue: (avatar) => new Date(avatar.updatedAt).getTime(),
        nowrap: true,
      },
    ],
    [],
  );

  return (
    <Table
      columns={columns}
      rows={avatars}
      getRowKey={(avatar) => avatar.id}
      emptyMessage="アバターはありません。"
      filteredEmptyMessage="条件に一致するアバターはありません。"
    />
  );
}