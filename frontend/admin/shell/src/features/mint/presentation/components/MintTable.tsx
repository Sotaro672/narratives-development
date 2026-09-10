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
        key: "id",
        header: "Mint ID",
        render: (mint) => mint.id,
        filter: {
          getValue: (mint) => mint.id,
          placeholder: "Mint IDで絞り込み",
        },
        minWidth: "180px",
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
        key: "brandId",
        header: "Brand ID",
        render: (mint) => mint.brandId,
        filter: {
          getValue: (mint) => mint.brandId,
          placeholder: "Brand IDで絞り込み",
        },
        minWidth: "180px",
      },
      {
        key: "tokenBlueprintId",
        header: "Token Blueprint ID",
        render: (mint) => mint.tokenBlueprintId,
        filter: {
          getValue: (mint) => mint.tokenBlueprintId,
          placeholder: "Token Blueprint IDで絞り込み",
        },
        minWidth: "200px",
      },
      {
        key: "products",
        header: "Products",
        render: (mint) => mint.products.length > 0 ? mint.products.join(", ") : "-",
        sortValue: (mint) => mint.products.length,
        filter: {
          getValue: (mint) => mint.products.join(" "),
          placeholder: "Product IDで絞り込み",
        },
        minWidth: "280px",
      },
      {
        key: "createdBy",
        header: "作成者",
        render: (mint) => mint.createdBy,
        filter: {
          getValue: (mint) => mint.createdBy,
          placeholder: "作成者で絞り込み",
        },
        minWidth: "180px",
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
      {
        key: "onChainTxSignature",
        header: "Tx Signature",
        render: (mint) => mint.onChainTxSignature || "-",
        filter: {
          getValue: (mint) => mint.onChainTxSignature || "",
          placeholder: "Signatureで絞り込み",
        },
        minWidth: "280px",
      },
    ],
    [],
  );

  return (
    <Table
      columns={columns}
      rows={mints}
      getRowKey={(mint) => mint.id}
      emptyMessage="Mintはありません。"
      filteredEmptyMessage="条件に一致するMintはありません。"
    />
  );
}