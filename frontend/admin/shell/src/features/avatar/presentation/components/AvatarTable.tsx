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
        key: "createdAt",
        header: "登録日時",
        render: (avatar) => formatDateTime(avatar.createdAt),
        sortValue: (avatar) =>
          new Date(avatar.createdAt).getTime(),
        nowrap: true,
      },
      {
        key: "avatarName",
        header: "アバター名",
        render: (avatar) => avatar.avatarName || "-",
        sortValue: (avatar) => avatar.avatarName,
        filter: {
          getValue: (avatar) => avatar.avatarName,
          placeholder: "アバター名",
        },
        nowrap: true,
      },
      {
        key: "userId",
        header: "ユーザーID",
        render: (avatar) => avatar.userId || "-",
        filter: {
          getValue: (avatar) => avatar.userId,
          placeholder: "ユーザーID",
        },
        nowrap: true,
      },
      {
        key: "walletAddress",
        header: "ウォレットアドレス",
        render: (avatar) => avatar.walletAddress || "-",
        filter: {
          getValue: (avatar) => avatar.walletAddress ?? "",
          placeholder: "ウォレットアドレス",
        },
        nowrap: true,
      },
      {
        key: "updatedAt",
        header: "最終更新日時",
        render: (avatar) => formatDateTime(avatar.updatedAt),
        sortValue: (avatar) =>
          new Date(avatar.updatedAt).getTime(),
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