// frontend/admin/shell/src/features/company/presentation/components/ContractTokenBlueprintTable.tsx

import { useMemo } from "react";

import type { ContractTokenBlueprintRow } from "../../../../shared/type/contractDetail";
import Table, { type TableColumn } from "../../../../shared/ui/Table/Table";
import { formatDateTime } from "../../../../shared/util/dateFormat";

type ContractTokenBlueprintTableProps = {
  tokenBlueprints: ContractTokenBlueprintRow[];
  onTokenBlueprintClick?: (tokenBlueprint: ContractTokenBlueprintRow) => void;
};

export default function ContractTokenBlueprintTable({
  tokenBlueprints,
  onTokenBlueprintClick,
}: ContractTokenBlueprintTableProps) {
  const columns = useMemo<TableColumn<ContractTokenBlueprintRow>[]>(
    () => [
      {
        key: "name",
        header: "トークン",
        render: (tokenBlueprint) => tokenBlueprint.name || tokenBlueprint.id,
        sortValue: (tokenBlueprint) => tokenBlueprint.name || tokenBlueprint.id,
        filter: {
          getValue: (tokenBlueprint) =>
            [tokenBlueprint.name, tokenBlueprint.id].filter(Boolean).join(" "),
          placeholder: "トークン名・IDで絞り込み",
        },
        nowrap: true,
      },
      {
        key: "symbol",
        header: "シンボル",
        render: (tokenBlueprint) => tokenBlueprint.symbol || "-",
        sortValue: (tokenBlueprint) => tokenBlueprint.symbol,
        filter: {
          getValue: (tokenBlueprint) => tokenBlueprint.symbol,
          placeholder: "シンボルで絞り込み",
        },
        nowrap: true,
      },
      {
        key: "brandName",
        header: "ブランド",
        render: (tokenBlueprint) => tokenBlueprint.brandName || "-",
        sortValue: (tokenBlueprint) => tokenBlueprint.brandName,
        filter: {
          getValue: (tokenBlueprint) => tokenBlueprint.brandName,
          placeholder: "ブランド名で絞り込み",
        },
        nowrap: true,
      },
      {
        key: "assigneeName",
        header: "担当者",
        render: (tokenBlueprint) => tokenBlueprint.assigneeName || "-",
        sortValue: (tokenBlueprint) => tokenBlueprint.assigneeName,
        filter: {
          getValue: (tokenBlueprint) => tokenBlueprint.assigneeName,
          placeholder: "担当者名で絞り込み",
        },
        nowrap: true,
      },
      {
        key: "minted",
        header: "ミント",
        render: (tokenBlueprint) =>
          tokenBlueprint.minted ? "ミント済み" : "未ミント",
        sortValue: (tokenBlueprint) => tokenBlueprint.minted,
        filter: {
          getValue: (tokenBlueprint) =>
            tokenBlueprint.minted ? "minted" : "not_minted",
          options: [
            { value: "minted", label: "ミント済み" },
            { value: "not_minted", label: "未ミント" },
          ],
        },
        nowrap: true,
      },
      {
        key: "createdAt",
        header: "作成日時",
        render: (tokenBlueprint) => formatDateTime(tokenBlueprint.createdAt),
        sortValue: (tokenBlueprint) =>
          new Date(tokenBlueprint.createdAt).getTime(),
        nowrap: true,
      },
      {
        key: "updatedAt",
        header: "最終更新日時",
        render: (tokenBlueprint) => formatDateTime(tokenBlueprint.updatedAt),
        sortValue: (tokenBlueprint) =>
          new Date(tokenBlueprint.updatedAt).getTime(),
        nowrap: true,
      },
    ],
    [],
  );

  return (
    <Table
      columns={columns}
      rows={tokenBlueprints}
      getRowKey={(tokenBlueprint) => tokenBlueprint.id}
      onRowClick={onTokenBlueprintClick}
      emptyMessage="トークン設計はありません。"
      filteredEmptyMessage="条件に一致するトークン設計はありません。"
    />
  );
}