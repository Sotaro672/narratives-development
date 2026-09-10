// frontend/admin/shell/src/features/mint/presentation/components/MintTable.tsx

import { useMemo } from "react";

import type { Mint, MintStatus } from "../../../../shared/type/mint";
import Table, { type TableColumn, type TableFilterOption } from "../../../../shared/ui/Table/Table";
import { formatDateTime } from "../../../../shared/util/dateFormat";

type MintTableProps = {
  mints: Mint[];
};

const MINT_STATUSES: MintStatus[] = [
  "CREATED",
  "QUEUED",
  "MINTING",
  "PARTIALLY_MINTED",
  "MINTED",
  "FAILED_RETRYABLE",
  "FAILED_FATAL",
];

const STATUS_OPTIONS: TableFilterOption[] = MINT_STATUSES.map((status) => ({
  value: status,
  label: status,
}));

export default function MintTable({ mints }: MintTableProps) {
  const columns = useMemo<TableColumn<Mint>[]>(
    () => [
      {
        key: "createdAt",
        header: "作成日時",
        render: (mint) => formatDateTime(mint.createdAt),
        sortValue: (mint) => new Date(mint.createdAt).getTime(),
        nowrap: true,
      },
      {
        key: "status",
        header: "ステータス",
        render: (mint) => mint.status,
        sortValue: (mint) => mint.status,
        filter: {
          getValue: (mint) => mint.status,
          options: STATUS_OPTIONS,
        },
        nowrap: true,
      },
      {
        key: "brandName",
        header: "ブランド名",
        render: (mint) => mint.brandName || "-",
        sortValue: (mint) => mint.brandName,
        filter: {
          getValue: (mint) => mint.brandName,
          placeholder: "ブランド名で絞り込み",
        },
        minWidth: "180px",
      },
      {
        key: "tokenBlueprintName",
        header: "トークン名",
        render: (mint) => mint.tokenBlueprintName || "-",
        sortValue: (mint) => mint.tokenBlueprintName,
        filter: {
          getValue: (mint) => mint.tokenBlueprintName,
          placeholder: "トークン名で絞り込み",
        },
        minWidth: "200px",
      },
      {
        key: "productCount",
        header: "Products",
        render: (mint) => mint.productCount,
        sortValue: (mint) => mint.productCount,
        width: "100px",
        nowrap: true,
      },
      {
        key: "requestedBy",
        header: "申請者",
        render: (mint) => mint.requestedBy || "-",
        filter: {
          getValue: (mint) => mint.requestedBy || "",
          placeholder: "申請者で絞り込み",
        },
        minWidth: "180px",
      },
      {
        key: "mintedAt",
        header: "Mint完了日時",
        render: (mint) => formatDateTime(mint.mintedAt),
        sortValue: (mint) => mint.mintedAt ? new Date(mint.mintedAt).getTime() : null,
        nowrap: true,
      },
      {
        key: "scheduledBurnDate",
        header: "Burn予定日時",
        render: (mint) => formatDateTime(mint.scheduledBurnDate),
        sortValue: (mint) => mint.scheduledBurnDate ? new Date(mint.scheduledBurnDate).getTime() : null,
        nowrap: true,
      },
    ],
    [],
  );

  return (
    <Table
      columns={columns}
      rows={mints}
      getRowKey={(mint) =>
        [
          mint.createdAt,
          mint.status,
          mint.brandName,
          mint.tokenBlueprintName,
          mint.productCount,
          mint.requestedBy ?? "",
          mint.mintedAt ?? "",
          mint.scheduledBurnDate ?? "",
        ].join(":")
      }
      emptyMessage="Mintはありません。"
      filteredEmptyMessage="条件に一致するMintはありません。"
    />
  );
}